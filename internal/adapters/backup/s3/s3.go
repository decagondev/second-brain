// Package s3 is a write-once backup target on any S3-compatible object store with object
// lock (sb-audit(7), WORM storage; SB-1603): the self-hosted cell's anchors and its
// off-site backup copy. Every object is written once (If-None-Match: *), in compliance
// mode until its retention date, which nobody can shorten or delete before, the store's
// own root included. Requests are signed with AWS Signature Version 4, written here on
// the standard library, so no module is added.
package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // Content-MD5 is what S3 demands for object-lock writes, not a security hash
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/backup"
)

// MinRetentionDays is the shortest default retention a bucket may have: seven years
// (ADR-0012), counted as 2,555 days.
const MinRetentionDays = 7 * 365

// Bucket is one bucket on one store.
type Bucket struct {
	Endpoint   string // http(s)://HOST[:PORT], path-style addressing
	BucketName string // the bucket
	Region     string // "" is us-east-1
	// Creds returns {"access_key","secret_key"} as JSON (vault://tenants/TENANT/worm), at
	// call time.
	Creds func(ctx context.Context) (string, error)
	HTTP  *http.Client
	Label string // name in drill reports
	Away  bool   // off-site
	Now   func() time.Time
}

var _ backup.Target = Bucket{}

// Errors.
var (
	ErrExists    = errors.New("s3: object exists; copies are write-once")
	ErrNotLocked = errors.New("s3: the bucket has no compliance-mode object lock of at least seven years; refused")
)

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*(/[a-z0-9][a-z0-9._-]*)*$`)

// Name implements backup.Target.
func (b Bucket) Name() string { return b.Label }

// Medium implements backup.Target.
func (b Bucket) Medium() string { return "object-store" }

// Offsite implements backup.Target.
func (b Bucket) Offsite() bool { return b.Away }

func (b Bucket) region() string {
	if b.Region == "" {
		return "us-east-1"
	}
	return b.Region
}

func (b Bucket) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

type creds struct {
	Access string `json:"access_key"`
	Secret string `json:"secret_key"`
}

// do signs and sends one request; a non-2xx answer is an error with its code.
func (b Bucket) do(ctx context.Context, method, key string, query url.Values, hdr http.Header, body []byte) (*http.Response, error) {
	raw, err := b.Creds(ctx)
	if err != nil {
		return nil, err
	}
	var c creds
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Access == "" || c.Secret == "" {
		return nil, errors.New("s3: credentials are not {access_key, secret_key}")
	}
	path := "/" + b.BucketName
	if key != "" {
		path += "/" + key
	}
	u := strings.TrimRight(b.Endpoint, "/") + path
	if q := query.Encode(); q != "" {
		u += "?" + q
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, vs := range hdr {
		req.Header[k] = vs
	}
	Sign(req, body, c.Access, c.Secret, b.region(), b.now())
	hc := b.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: time.Minute}
	}
	return hc.Do(req)
}

func failed(resp *http.Response, what string) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var e struct{ Code, Message string }
	_ = xml.Unmarshal(b, &e)
	return fmt.Errorf("s3: %s: HTTP %d %s", what, resp.StatusCode, e.Code)
}

// Check refuses a bucket without object lock in compliance mode for at least seven years,
// as the GCP adapter refuses an unlocked bucket.
func (b Bucket) Check(ctx context.Context) error {
	resp, err := b.do(ctx, http.MethodGet, "", url.Values{"object-lock": {""}}, nil, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotLocked
	}
	if resp.StatusCode/100 != 2 {
		return failed(resp, "object-lock configuration")
	}
	var cfg struct {
		Enabled string `xml:"ObjectLockEnabled"`
		Rule    struct {
			Default struct {
				Mode  string `xml:"Mode"`
				Days  int    `xml:"Days"`
				Years int    `xml:"Years"`
			} `xml:"DefaultRetention"`
		} `xml:"Rule"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return fmt.Errorf("s3: object-lock configuration unreadable: %w", err)
	}
	d := cfg.Rule.Default
	if cfg.Enabled != "Enabled" || d.Mode != "COMPLIANCE" || d.Days+365*d.Years < MinRetentionDays {
		return ErrNotLocked
	}
	return nil
}

// Put implements backup.Target: write once, locked in compliance mode until until.
func (b Bucket) Put(ctx context.Context, name string, data []byte, until time.Time) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("s3: bad object name %q", name)
	}
	sum := md5.Sum(data) //nolint:gosec // see the import
	h := http.Header{}
	h.Set("Content-MD5", base64.StdEncoding.EncodeToString(sum[:]))
	h.Set("If-None-Match", "*")
	h.Set("X-Amz-Object-Lock-Mode", "COMPLIANCE")
	h.Set("X-Amz-Object-Lock-Retain-Until-Date", until.UTC().Format(time.RFC3339))
	resp, err := b.do(ctx, http.MethodPut, name, nil, h, data)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusPreconditionFailed || resp.StatusCode == http.StatusConflict:
		return ErrExists
	case resp.StatusCode/100 != 2:
		return failed(resp, "put "+name)
	}
	return nil
}

// Get implements backup.Target.
func (b Bucket) Get(ctx context.Context, name string) ([]byte, error) {
	if !nameRe.MatchString(name) {
		return nil, fmt.Errorf("s3: bad object name %q", name)
	}
	resp, err := b.do(ctx, http.MethodGet, name, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return nil, failed(resp, "get "+name)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<30))
}

// List implements backup.Target: every object under prefix, sorted.
func (b Bucket) List(ctx context.Context, prefix string) ([]string, error) {
	var out []string
	token := ""
	for page := 0; page < 10000; page++ {
		q := url.Values{"list-type": {"2"}, "prefix": {prefix}}
		if token != "" {
			q.Set("continuation-token", token)
		}
		resp, err := b.do(ctx, http.MethodGet, "", q, nil, nil)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode/100 != 2 {
			err := failed(resp, "list")
			_ = resp.Body.Close()
			return nil, err
		}
		var r struct {
			Contents []struct {
				Key string `xml:"Key"`
			} `xml:"Contents"`
			Truncated bool   `xml:"IsTruncated"`
			Next      string `xml:"NextContinuationToken"`
		}
		err = xml.NewDecoder(resp.Body).Decode(&r)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("s3: list unreadable: %w", err)
		}
		for _, c := range r.Contents {
			out = append(out, c.Key)
		}
		if !r.Truncated || r.Next == "" {
			sort.Strings(out)
			return out, nil
		}
		token = r.Next
	}
	return nil, errors.New("s3: list never ended")
}

// Sign adds AWS Signature Version 4 headers to req for service s3: x-amz-date,
// x-amz-content-sha256 and Authorization over every header set so far.
func Sign(req *http.Request, body []byte, access, secret, region string, now time.Time) {
	amzDate := now.UTC().Format("20060102T150405Z")
	day := amzDate[:8]
	ph := sha256.Sum256(body)
	payload := hex.EncodeToString(ph[:])
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payload)
	host := req.URL.Host
	if req.Host != "" {
		host = req.Host
	}
	names := []string{"host"}
	vals := map[string]string{"host": host}
	for k, vs := range req.Header {
		lk := strings.ToLower(k)
		if lk == "authorization" {
			continue
		}
		names = append(names, lk)
		vals[lk] = strings.Join(vs, ",")
	}
	sort.Strings(names)
	var ch strings.Builder
	for _, n := range names {
		ch.WriteString(n + ":" + strings.TrimSpace(vals[n]) + "\n")
	}
	signed := strings.Join(names, ";")
	canonical := strings.Join([]string{req.Method, uriEncode(req.URL.EscapedPath(), false), canonicalQuery(req.URL.Query()), ch.String(), signed, payload}, "\n")
	cr := sha256.Sum256([]byte(canonical))
	scope := day + "/" + region + "/s3/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hex.EncodeToString(cr[:])
	k := hmacSHA256([]byte("AWS4"+secret), day)
	k = hmacSHA256(k, region)
	k = hmacSHA256(k, "s3")
	k = hmacSHA256(k, "aws4_request")
	sig := hex.EncodeToString(hmacSHA256(k, toSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+access+"/"+scope+",SignedHeaders="+signed+",Signature="+sig)
}

func hmacSHA256(key []byte, s string) []byte {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write([]byte(s))
	return m.Sum(nil)
}

// uriEncode encodes as SigV4 wants: unreserved characters stay, and '/' stays in a path.
func uriEncode(s string, encodeSlash bool) string {
	if decoded, err := url.PathUnescape(s); err == nil {
		s = decoded
	}
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9', c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		case c == '/' && !encodeSlash:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func canonicalQuery(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vs := append([]string{}, q[k]...)
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, uriEncode(k, true)+"="+uriEncode(v, true))
		}
	}
	return strings.Join(parts, "&")
}

// CreateLocked makes the bucket with object lock, and sets its default retention to
// compliance mode for years years (the install step; SB-1603). A bucket that exists
// already is kept; its lock is then whatever Check finds.
func (b Bucket) CreateLocked(ctx context.Context, years int) error {
	h := http.Header{}
	h.Set("X-Amz-Bucket-Object-Lock-Enabled", "true")
	resp, err := b.do(ctx, http.MethodPut, "", nil, h, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("s3: create bucket: HTTP %d", resp.StatusCode)
	}
	cfg := fmt.Sprintf(`<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><ObjectLockEnabled>Enabled</ObjectLockEnabled>`+
		`<Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Years>%d</Years></DefaultRetention></Rule></ObjectLockConfiguration>`, years)
	sum := md5.Sum([]byte(cfg)) //nolint:gosec // see the import
	h = http.Header{}
	h.Set("Content-MD5", base64.StdEncoding.EncodeToString(sum[:]))
	h.Set("Content-Type", "application/xml")
	resp, err = b.do(ctx, http.MethodPut, "", url.Values{"object-lock": {""}}, h, []byte(cfg))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return failed(resp, "object-lock configuration")
	}
	return b.Check(ctx)
}

// Enforced writes a probe object and tries to delete it, version and all, before its
// retention ends: a store that lets the delete through doesn't enforce the lock, whatever
// its configuration says (LAW-11). The probe stays, locked, as the proof.
func (b Bucket) Enforced(ctx context.Context) error {
	name := "probe/" + b.now().Format("20060102t150405.000000000")
	name = strings.ReplaceAll(name, ".", "-")
	if err := b.Put(ctx, name, []byte("probe"), b.now().Add(24*time.Hour)); err != nil {
		return err
	}
	resp, err := b.do(ctx, http.MethodHead, name, nil, nil, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	q := url.Values{}
	if v := resp.Header.Get("X-Amz-Version-Id"); v != "" {
		q.Set("versionId", v)
	}
	resp, err = b.do(ctx, http.MethodDelete, name, q, nil, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode/100 == 2 {
		if got, gerr := b.Get(ctx, name); gerr != nil || string(got) != "probe" {
			return fmt.Errorf("%w: a locked object was deleted", ErrNotLocked)
		}
	}
	return nil
}
