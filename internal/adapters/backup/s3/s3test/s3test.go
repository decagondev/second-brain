// Package s3test is a recorded fake of an S3 store with object lock, for the s3 adapter's
// tests and the LAW-11 law test (SB-1603): write-once by If-None-Match, compliance
// retention kept per object version, the bucket's lock configuration, ListObjectsV2
// paging, and a delete that the lock refuses inside an object's retention. It behaves as
// versitygw does for the calls the adapter makes.
package s3test

import (
	"encoding/xml"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Locked is a bucket's lock configuration: compliance mode for seven years.
const Locked = `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Years>7</Years></DefaultRetention></Rule></ObjectLockConfiguration>`

// Fake serves one bucket, Bucket, to requests signed with access key Access.
type Fake struct {
	Bucket string
	Access string
	Lock   string // the object-lock configuration XML; "" = none
	PageAt int    // list at most this many keys a page; 0 = all
	// Unenforced makes the store accept a delete whatever the lock says, as a store that
	// only claims object lock does.
	Unenforced bool
	Now        func() time.Time

	mu     sync.Mutex
	objs   map[string][]byte
	retain map[string]time.Time
}

// New is a fake with a seven-year compliance lock.
func New(bucket, access string) *Fake {
	return &Fake{Bucket: bucket, Access: access, Lock: Locked}
}

// Objects is the number of objects held.
func (f *Fake) Objects() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.objs)
}

func (f *Fake) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.objs == nil {
		f.objs, f.retain = map[string][]byte{}, map[string]time.Time{}
	}
	if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential="+f.Access+"/") || r.Header.Get("X-Amz-Content-Sha256") == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	key := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/"+f.Bucket), "/")
	q := r.URL.Query()
	switch {
	case r.Method == http.MethodGet && key == "" && q.Has("object-lock"):
		if f.Lock == "" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, "<Error><Code>ObjectLockConfigurationNotFoundError</Code></Error>")
			return
		}
		_, _ = io.WriteString(w, f.Lock)
	case r.Method == http.MethodGet && key == "" && q.Get("list-type") == "2":
		f.list(w, q.Get("prefix"), q.Get("continuation-token"))
	case r.Method == http.MethodPut && key != "":
		if r.Header.Get("If-None-Match") != "*" || r.Header.Get("Content-MD5") == "" || r.Header.Get("X-Amz-Object-Lock-Mode") != "COMPLIANCE" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, ok := f.objs[key]; ok {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		until, err := time.Parse(time.RFC3339, r.Header.Get("X-Amz-Object-Lock-Retain-Until-Date"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b, _ := io.ReadAll(r.Body)
		f.objs[key], f.retain[key] = b, until
	case r.Method == http.MethodHead:
		if _, ok := f.objs[key]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("X-Amz-Version-Id", "v1")
	case r.Method == http.MethodDelete:
		if _, ok := f.objs[key]; !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !f.Unenforced && f.now().Before(f.retain[key]) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, "<Error><Code>AccessDenied</Code><Message>object is WORM protected</Message></Error>")
			return
		}
		delete(f.objs, key)
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodGet:
		b, ok := f.objs[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (f *Fake) list(w http.ResponseWriter, prefix, token string) {
	var keys []string
	for k := range f.objs {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	start := 0
	if token != "" {
		for i, k := range keys {
			if k == token {
				start = i
			}
		}
	}
	end := len(keys)
	if f.PageAt > 0 && start+f.PageAt < end {
		end = start + f.PageAt
	}
	type c struct{ Key string }
	res := struct {
		XMLName   xml.Name `xml:"ListBucketResult"`
		Contents  []c
		Truncated bool   `xml:"IsTruncated"`
		Next      string `xml:"NextContinuationToken,omitempty"`
	}{Truncated: end < len(keys)}
	for _, k := range keys[start:end] {
		res.Contents = append(res.Contents, c{k})
	}
	if res.Truncated {
		res.Next = keys[end]
	}
	_ = xml.NewEncoder(w).Encode(res)
}
