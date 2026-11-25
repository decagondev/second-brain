package gdrive

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Scope is the only OAuth scope the connector asks for (read-only).
const Scope = "https://www.googleapis.com/auth/drive.readonly"

// Registrar learns secret values so logs can redact them (secrets.Registrar).
type Registrar interface{ Register(secret string) }

// ServiceAccount mints access tokens with domain-wide delegation. The key is read from
// the vault at call time through Key; tokens are cached per subject until shortly before
// they expire, and registered with the redactor.
type ServiceAccount struct {
	Key    func(ctx context.Context) (string, error) // vault://tenants/T/connectors/docs
	HTTP   *http.Client
	Redact Registrar
	Now    func() time.Time
	// Scope is the one OAuth scope its tokens carry; "" is Scope, the Drive one. The
	// calendar (SB-4202) asks for calendar.events.readonly.
	Scope string

	mu    sync.Mutex
	cache map[string]token
}

type token struct {
	value string
	until time.Time
}

type keyFile struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// Token returns an access token for subject ("" means the service account itself).
func (s *ServiceAccount) Token(ctx context.Context, subject string) (string, error) {
	now := s.Now()
	s.mu.Lock()
	if t, ok := s.cache[subject]; ok && now.Before(t.until) {
		s.mu.Unlock()
		return t.value, nil
	}
	s.mu.Unlock()

	raw, err := s.Key(ctx)
	if err != nil {
		return "", err
	}
	var k keyFile
	if err := json.Unmarshal([]byte(raw), &k); err != nil || k.ClientEmail == "" || k.PrivateKey == "" || k.TokenURI == "" {
		return "", errors.New("gdrive: service account key is not a valid key file")
	}
	block, _ := pem.Decode([]byte(k.PrivateKey))
	if block == nil {
		return "", errors.New("gdrive: service account private key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", errors.New("gdrive: service account private key is not PKCS#8")
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("gdrive: service account key is not RSA")
	}
	scope := s.Scope
	if scope == "" {
		scope = Scope
	}
	claims := map[string]any{"iss": k.ClientEmail, "scope": scope, "aud": k.TokenURI, "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}
	if subject != "" {
		claims["sub"] = subject
	}
	assertion, err := signJWT(rsaKey, claims)
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {assertion}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("gdrive: token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gdrive: token endpoint returned %d", resp.StatusCode)
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&t); err != nil || t.AccessToken == "" {
		return "", errors.New("gdrive: token endpoint returned no access token")
	}
	if s.Redact != nil {
		s.Redact.Register(t.AccessToken)
	}
	life := time.Duration(t.ExpiresIn) * time.Second
	if life <= 0 {
		life = time.Hour
	}
	s.mu.Lock()
	if s.cache == nil {
		s.cache = map[string]token{}
	}
	s.cache[subject] = token{t.AccessToken, now.Add(life - time.Minute)}
	s.mu.Unlock()
	return t.AccessToken, nil
}

func signJWT(k *rsa.PrivateKey, claims map[string]any) (string, error) {
	b64 := base64.RawURLEncoding.EncodeToString
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	p, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	in := b64(h) + "." + b64(p)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, k, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return in + "." + b64(sig), nil
}
