package parser

import (
	"fmt"
	"time"
)

type Item struct {
	Title             string
	Link              string
	PublishedAt       string
	PublishedAtParsed *time.Time
}

func Parse(url string) ([]Item, error) {
	body, err := fromURL(url)
	if err != nil {
		return nil, fmt.Errorf("parsing from url: %w", err)
	}

	items, err := parseRSS(body)
	if err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	return items, nil
}
