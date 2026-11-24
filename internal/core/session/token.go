package session

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/ports"
)

// Read-only API tokens (sb-api(3), Read-only API tokens; sb-identity(7); SB-2306): a
// session the person names, for their own scripts, that asks and searches as them and
// does nothing else, for at most MaxTokenDays. The store keeps the token's SHA-256 as its
// key, never the token.

const (
	MaxTokenDays     = 90
	DefaultTokenDays = 30
	MaxTokenName     = 60
	// TokenPrefix starts every token, so a scanner or a log redactor knows one.
	TokenPrefix = "sbt_"
)

// ErrToken is a token request outside the rules: a name, or days outside 1 to 90.
var ErrToken = errors.New("session: a token needs a name of 1 to 60 characters and 1 to 90 days")

// NewToken makes one token for user: the token, shown to the person once, and the session
// to store, keyed by TokenKey(token). days 0 is DefaultTokenDays.
func NewToken(rnd io.Reader, tenant, user, name string, days int, now time.Time) (string, ports.Session, error) {
	name = strings.TrimSpace(name)
	if days == 0 {
		days = DefaultTokenDays
	}
	if name == "" || utf8.RuneCountInString(name) > MaxTokenName || days < 1 || days > MaxTokenDays || !strings.HasPrefix(user, "usr_") {
		return "", ports.Session{}, ErrToken
	}
	secret, id := make([]byte, 32), make([]byte, 8)
	if _, err := io.ReadFull(rnd, secret); err != nil {
		return "", ports.Session{}, err
	}
	if _, err := io.ReadFull(rnd, id); err != nil {
		return "", ports.Session{}, err
	}
	token := TokenPrefix + base64.RawURLEncoding.EncodeToString(secret)
	return token, ports.Session{ID: TokenKey(token), Tenant: tenant, User: user, SignedIn: now, Validated: now,
		Token: "tok_" + hex.EncodeToString(id), Name: name, Expires: now.AddDate(0, 0, days)}, nil
}

// TokenKey is the store key of a token: its SHA-256.
func TokenKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return "tokh_" + hex.EncodeToString(h[:])
}

// IsToken is whether a bearer value has the shape of a token.
func IsToken(v string) bool {
	return strings.HasPrefix(v, TokenPrefix) && len(v) == len(TokenPrefix)+43
}

// TokenRoute is whether an API token may make this request: ask, search and the read-only
// MCP endpoint (SB-3801, served only with sb-cell(5) mcp: true), nothing else.
func TokenRoute(method, path string) bool {
	return method == "POST" && (path == "/v1/ask" || path == "/v1/mcp") || method == "GET" && path == "/v1/search"
}
