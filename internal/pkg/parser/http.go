package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	httpTimeout  = 30 * time.Second
	maxFeedBytes = 32 << 20 // 32 MiB cap on a single feed response

	// Connection settings below match http.DefaultTransport.
	tlsHandshakeTimeout = 10 * time.Second
	idleConnTimeout     = 90 * time.Second
	maxIdleConns        = 100
)

var (
	errBadResponseCode = errors.New("bad response code")
	errFeedTooLarge    = errors.New("feed response exceeds size limit")
)

// feedClient is shared so idle connections are pooled and reaped instead of
// leaking one transport per fetch.
//
//nolint:gochecknoglobals // stateless apart from its connection pool
var feedClient = newClient()

// newClient speaks HTTP/1.1 only: since 2026-10-01 news.ycombinator.com
// answers Go's HTTP/2 client with 419 while serving HTTP/1.1 normally.
// The transport is built fresh because a clone of http.DefaultTransport still
// advertises h2 over ALPN, letting the server pick a protocol we won't speak.
func newClient() *http.Client {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)

	transport := &http.Transport{ //nolint:exhaustruct // defaults apart from HTTP/1.1-only
		Proxy:               http.ProxyFromEnvironment,
		Protocols:           protocols,
		TLSHandshakeTimeout: tlsHandshakeTimeout,
		IdleConnTimeout:     idleConnTimeout,
		MaxIdleConns:        maxIdleConns,
	}

	return &http.Client{ //nolint:exhaustruct // no need to set any other fields
		Timeout:   httpTimeout,
		Transport: transport,
	}
}

func fromURL(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: %w", err)
	}

	req.Header.Set("User-Agent", "Mynews/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request http request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: %d", errBadResponseCode, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	if int64(len(body)) > maxFeedBytes {
		return nil, errFeedTooLarge
	}

	return body, nil
}
