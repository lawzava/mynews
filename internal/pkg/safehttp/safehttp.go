// Package safehttp builds HTTP clients that refuse connections to non-public IP
// addresses, preventing SSRF when fetching attacker-influenceable URLs such as
// article links taken from feed items or model-download redirects.
package safehttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
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

	addr, err := netip.ParseAddr(host)
	if err != nil || !isPublic(addr) {
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

// blockedPrefixes are special-use ranges not covered by the netip.Addr helpers
// (CGNAT, NAT64, IETF/benchmark/test/reserved blocks) that could still reach
// internal infrastructure.
//
//nolint:gochecknoglobals // read-only denylist
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // "this host"
	netip.MustParsePrefix("100.64.0.0/10"),   // CGNAT / cloud carrier
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved / future use
	netip.MustParsePrefix("2001:db8::/32"),   // IPv6 documentation
	netip.MustParsePrefix("64:ff9b::/96"),    // NAT64
}

func isPublic(addr netip.Addr) bool {
	addr = addr.Unmap() // normalize IPv4-mapped IPv6 so IPv4 rules apply

	if !addr.IsValid() ||
		addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() {
		return false
	}

	for idx := range blockedPrefixes {
		if blockedPrefixes[idx].Contains(addr) {
			return false
		}
	}

	return true
}
