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
)

var (
	errBadResponseCode = errors.New("bad response code")
	errFeedTooLarge    = errors.New("feed response exceeds size limit")
)

func fromURL(url string) ([]byte, error) {
	//nolint:exhaustruct // no need to set any other fields
	client := http.Client{Timeout: httpTimeout}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
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
		return nil, errBadResponseCode
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
