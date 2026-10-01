//nolint:testpackage // exercises the unexported feed fetcher
package parser

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const statusHNRejectsHTTP2 = 419

// newHNLikeServer mimics news.ycombinator.com since 2026-10-01: it answers
// Go's HTTP/2 client with 419 and serves the feed over HTTP/1.1.
func newHNLikeServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 {
			http.Error(writer, "Sorry", statusHNRejectsHTTP2)

			return
		}

		_, _ = writer.Write([]byte(`<rss version="2.0"><channel></channel></rss>`))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	client := newClient()
	transport, _ := client.Transport.(*http.Transport)
	serverTransport, _ := server.Client().Transport.(*http.Transport)
	// Keep the client's own TLS config: its ALPN list decides whether the
	// server may pick HTTP/2, so replacing it would hide the bug under test.
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12} //nolint:exhaustruct // defaults
	}

	transport.TLSClientConfig.RootCAs = serverTransport.TLSClientConfig.RootCAs

	return server, client
}

func TestFromURLFetchesFeedFromHTTP2RejectingServer(t *testing.T) {
	t.Parallel()

	server, client := newHNLikeServer(t)

	body, err := fromURL(t.Context(), client, server.URL)
	if err != nil {
		t.Fatalf("fromURL: %v", err)
	}

	if !strings.Contains(string(body), "<rss") {
		t.Fatalf("body = %q, want RSS feed", body)
	}
}

func TestFromURLReportsStatusCode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Sorry", statusHNRejectsHTTP2)
	}))
	t.Cleanup(server.Close)

	_, err := fromURL(t.Context(), newClient(), server.URL)
	if err == nil {
		t.Fatal("fromURL succeeded, want bad response code error")
	}

	if !strings.Contains(err.Error(), "419") {
		t.Fatalf("error = %q, want it to name status 419", err)
	}
}
