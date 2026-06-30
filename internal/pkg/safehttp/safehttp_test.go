package safehttp_test

import (
	"errors"
	"mynews/internal/pkg/safehttp"
	"testing"
	"time"
)

func TestClientBlocksNonPublicAddresses(t *testing.T) {
	t.Parallel()

	client := safehttp.Client(2 * time.Second)

	targets := []string{
		"http://127.0.0.1:80/",                     // loopback
		"http://[::1]:80/",                         // IPv6 loopback
		"http://10.0.0.1/",                         // private
		"http://192.168.1.1/",                      // private
		"http://169.254.169.254/latest/meta-data/", // cloud metadata (link-local)
		"http://0.0.0.0/",                          // unspecified
	}

	for _, target := range targets {
		response, err := client.Get(target)
		if err == nil {
			_ = response.Body.Close()

			t.Errorf("%s: expected a blocked-address error, got nil", target)

			continue
		}

		if !errors.Is(err, safehttp.ErrBlockedAddress) {
			t.Errorf("%s: error %v does not wrap ErrBlockedAddress", target, err)
		}
	}
}
