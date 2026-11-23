package wire

import (
	"fmt"
	"os"
	"strings"

	"example.com/second-brain/internal/core/box"
)

// ProxyFromFile sets the process's HTTPS_PROXY from the file SB_HTTPS_PROXY_FILE names,
// when it names one (sb-cell(5), Environment; SB-1806). The file holds a proxy URL that
// may carry credentials, so the URL never sits in the container's environment, where
// `docker inspect` would show it. Each service calls this first, before any HTTP client
// reads the environment; Go's clients then use the proxy for https:// and honour NO_PROXY.
// Setting both SB_HTTPS_PROXY_FILE and HTTPS_PROXY is refused.
func ProxyFromFile(getenv func(string) string, setenv func(k, v string) error) error {
	path := getenv("SB_HTTPS_PROXY_FILE")
	if path == "" {
		return nil
	}
	if getenv("HTTPS_PROXY") != "" || getenv("https_proxy") != "" {
		return fmt.Errorf("%w: SB_HTTPS_PROXY_FILE and HTTPS_PROXY are both set; set one", ErrConfig)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%w: SB_HTTPS_PROXY_FILE can't be read", ErrConfig)
	}
	raw := strings.TrimSpace(string(b))
	if err := box.CheckProxyFile(raw); err != nil {
		return fmt.Errorf("%w: SB_HTTPS_PROXY_FILE: %v", ErrConfig, err)
	}
	return setenv("HTTPS_PROXY", raw)
}
