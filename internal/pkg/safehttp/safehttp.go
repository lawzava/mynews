// Package safehttp builds HTTP clients that refuse connections to non-public IP
// addresses, preventing SSRF when fetching attacker-influenceable URLs such as
// article links taken from feed items or model-download redirects.
package safehttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

const maxRedirects = 10

var (
	// ErrBlockedAddress is returned when a request targets a non-public address.
	ErrBlockedAddress   = errors.New("blocked non-public address")
	errTooManyRedirects = errors.New("too many redirects")
)

// Client returns an http.Client whose connections are rejected when the resolved
// peer IP is loopback, private, link-local (incl. cloud metadata), multicast, or
// unspecified. The check runs per dial, so it also covers redirects and DNS
// rebinding (the resolved IP is validated, not the hostname).
func Client(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{ //nolint:exhaustruct // timeout + control are the only settings we need
		Timeout: timeout,
		Control: controlBlockPrivate,
	}

	transport := &http.Transport{ //nolint:exhaustruct // default transport behavior is fine apart from the safe dialer
		DialContext:         dialer.DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: timeout,
	}

	return &http.Client{ //nolint:exhaustruct // only these fields matter
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: checkRedirect,
	}
}

func controlBlockPrivate(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("splitting dial address: %w", err)
	}

	parsedIP := net.ParseIP(host)
	if parsedIP == nil || !isPublic(parsedIP) {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, address)
	}

	return nil
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errTooManyRedirects
	}

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q", ErrBlockedAddress, req.URL.Scheme)
	}

	return nil
}

func isPublic(parsedIP net.IP) bool {
	return !parsedIP.IsLoopback() &&
		!parsedIP.IsPrivate() &&
		!parsedIP.IsLinkLocalUnicast() &&
		!parsedIP.IsLinkLocalMulticast() &&
		!parsedIP.IsMulticast() &&
		!parsedIP.IsUnspecified()
}
