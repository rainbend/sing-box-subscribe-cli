package subconv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rainbend/sing-box-subscribe-cli/internal/templates"
)

func LoadBytes(ctx context.Context, location, userAgent string, timeout time.Duration) ([]byte, error) {
	parsed, err := url.Parse(location)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return fetch(ctx, location, userAgent, timeout)
	}
	if isExplicitPath(location) {
		data, err := os.ReadFile(location)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", location, err)
		}
		return data, nil
	}
	if data, ok, err := templates.Read(location); err != nil {
		return nil, fmt.Errorf("read template %q: %w", location, err)
	} else if ok {
		return data, nil
	}
	data, err := os.ReadFile(location)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", location, err)
	}
	return data, nil
}

func isExplicitPath(location string) bool {
	if location == "" {
		return false
	}
	return location == "." || location == ".." || os.IsPathSeparator(location[0]) ||
		len(location) >= 2 && location[:2] == "./" ||
		len(location) >= 3 && location[:3] == "../" ||
		strings.ContainsAny(location, `/\`)
}

func fetch(ctx context.Context, rawURL, userAgent string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request %q: %w", rawURL, err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %q: HTTP %s", rawURL, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response %q: %w", rawURL, err)
	}
	return body, nil
}
