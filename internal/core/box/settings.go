package box

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrSetting is an install setting the box refuses. Its messages never repeat a password
// or a proxy's credentials: they name the setting and what's wrong with it.
var ErrSetting = errors.New("setting refused")

// CheckProxy checks an outbound proxy for SB_HTTPS_PROXY (sb-selfhost(7), Outbound proxy;
// SB-1806): an http:// or https:// URL with a host and nothing else. A URL that carries
// credentials is refused, because install.env and compose.env are not secret; those go in
// a file (SB_HTTPS_PROXY_FILE, CheckProxyFile).
func CheckProxy(raw string) error {
	u, err := proxyURL(raw)
	if err != nil {
		return err
	}
	if u.User != nil {
		return fmt.Errorf("%w: SB_HTTPS_PROXY carries credentials; put the whole URL in a file named by SB_HTTPS_PROXY_FILE instead", ErrSetting)
	}
	return nil
}

// CheckProxyFile checks the content of SB_HTTPS_PROXY_FILE: the same URL, credentials
// allowed.
func CheckProxyFile(raw string) error {
	_, err := proxyURL(raw)
	return err
}

// ProxyHost is the proxy's host:port, for saying which proxy is set without its
// credentials.
func ProxyHost(raw string) string {
	u, err := proxyURL(raw)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func proxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return nil, fmt.Errorf("%w: the proxy must be one URL, http://HOST:PORT", ErrSetting)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: the proxy isn't a URL", ErrSetting) // url's error repeats the URL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: the proxy must be http:// or https://", ErrSetting)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("%w: the proxy names no host", ErrSetting)
	}
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%w: the proxy is a host and port, with no path or query", ErrSetting)
	}
	return u, nil
}

// CheckDSN checks the external database's connection string (SB_DATABASE_DSN_FILE;
// sb-selfhost(7), A database on another machine; SB-1807). It must say sslmode=verify-full,
// so the cell checks the server's certificate and name, unless the host is this machine's
// loopback. Both forms are read: postgres://… and key=value pairs.
func CheckDSN(dsn string) error {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return fmt.Errorf("%w: the database's connection string is empty", ErrSetting)
	}
	host, mode, err := dsnParts(dsn)
	if err != nil {
		return err
	}
	if host == "" {
		return fmt.Errorf("%w: the connection string names no host", ErrSetting)
	}
	if strings.HasPrefix(host, "/") {
		return fmt.Errorf("%w: a Unix socket isn't another machine; the bundled database is SB_DATABASE=bundled", ErrSetting)
	}
	if mode == "verify-full" || loopback(host) {
		return nil
	}
	if mode == "" {
		mode = "unset (prefer)"
	}
	return fmt.Errorf("%w: sslmode is %s; a database on another machine needs sslmode=verify-full", ErrSetting, mode)
}

// DSNHost is the connection string's host and port, for saying which database is used
// without its password.
func DSNHost(dsn string) string {
	host, _, err := dsnParts(strings.TrimSpace(dsn))
	if err != nil {
		return ""
	}
	return host
}

func dsnParts(dsn string) (host, mode string, err error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, perr := url.Parse(dsn)
		if perr != nil {
			return "", "", fmt.Errorf("%w: the connection string isn't a URL", ErrSetting)
		}
		q := u.Query()
		host = u.Hostname()
		if h := q.Get("host"); host == "" && h != "" {
			host = h
		}
		return host, q.Get("sslmode"), nil
	}
	if strings.Contains(dsn, "://") {
		return "", "", fmt.Errorf("%w: the connection string must be postgres://… or key=value pairs", ErrSetting)
	}
	for _, kv := range strings.Fields(dsn) {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return "", "", fmt.Errorf("%w: the connection string's pairs must be key=value", ErrSetting)
		}
		v = strings.Trim(v, "'")
		switch k {
		case "host":
			host = v
		case "sslmode":
			mode = v
		}
	}
	return host, mode, nil
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// DSNParams are the connection string's libpq keywords (host, port, user, password,
// dbname, sslmode, …), for a pg_service.conf section that lets a Postgres client connect
// with the password neither in its arguments nor in its environment (SB-1807).
func DSNParams(dsn string) (map[string]string, error) {
	dsn = strings.TrimSpace(dsn)
	if err := CheckDSN(dsn); err != nil {
		return nil, err
	}
	out := map[string]string{}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("%w: the connection string isn't a URL", ErrSetting)
		}
		if u.User != nil {
			out["user"] = u.User.Username()
			if pw, ok := u.User.Password(); ok {
				out["password"] = pw
			}
		}
		if h := u.Hostname(); h != "" {
			out["host"] = h
		}
		if p := u.Port(); p != "" {
			out["port"] = p
		}
		if db := strings.TrimPrefix(u.Path, "/"); db != "" {
			out["dbname"] = db
		}
		for k, vs := range u.Query() {
			if len(vs) > 0 {
				out[k] = vs[len(vs)-1]
			}
		}
	} else {
		for _, kv := range strings.Fields(dsn) {
			k, v, _ := strings.Cut(kv, "=")
			out[k] = strings.Trim(v, "'")
		}
	}
	for k, v := range out {
		if strings.ContainsAny(k+v, "\r\n") || !keyword(k) {
			return nil, fmt.Errorf("%w: the connection string has a parameter a service file can't hold", ErrSetting)
		}
	}
	return out, nil
}

func keyword(k string) bool {
	if k == "" {
		return false
	}
	for _, c := range k {
		if (c < 'a' || c > 'z') && c != '_' {
			return false
		}
	}
	return true
}
