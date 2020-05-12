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
		return nil, fmt.Errorf("parsing feed '%s' from url: %w", url, err)
	}

	return nil, nil
}
