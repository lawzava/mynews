package hackernews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mynews/internal/pkg/safehttp"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiBaseURL      = "https://hacker-news.firebaseio.com"
	requestTimeout  = 10 * time.Second
	maxResponseSize = 1 << 20
)

var (
	errInvalidCommentsURL = errors.New("invalid Hacker News comments URL")
	errResponseTooLarge   = errors.New("hacker news API response exceeds size limit")
	errScoreMissing       = errors.New("hacker news API response has no score")
	errBadStatus          = errors.New("hacker news API returned bad status")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return newClient(apiBaseURL, safehttp.Client(requestTimeout))
}

func newClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (client *Client) StoryScore(ctx context.Context, commentsURL string) (int, error) {
	storyID, err := storyIDFromCommentsURL(commentsURL)
	if err != nil {
		return 0, err
	}

	target := fmt.Sprintf("%s/v0/item/%d.json", client.baseURL, storyID)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("create Hacker News score request: %w", err)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("fetch Hacker News score: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: %d", errBadStatus, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return 0, fmt.Errorf("read Hacker News score: %w", err)
	}

	if len(body) > maxResponseSize {
		return 0, errResponseTooLarge
	}

	var item struct {
		Score *int `json:"score"`
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		return 0, fmt.Errorf("decode Hacker News score: %w", err)
	}

	if item.Score == nil {
		return 0, errScoreMissing
	}

	return *item.Score, nil
}

func storyIDFromCommentsURL(commentsURL string) (int64, error) {
	parsed, err := url.Parse(commentsURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "news.ycombinator.com" || parsed.Path != "/item" {
		return 0, fmt.Errorf("%w: %q", errInvalidCommentsURL, commentsURL)
	}

	storyID, err := strconv.ParseInt(parsed.Query().Get("id"), 10, 64)
	if err != nil || storyID <= 0 {
		return 0, fmt.Errorf("%w: %q", errInvalidCommentsURL, commentsURL)
	}

	return storyID, nil
}
