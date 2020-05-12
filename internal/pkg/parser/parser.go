package parser

import (
	"errors"
	"fmt"
	"time"
)

type Item struct {
	Title             string
	Link              string
	PublishedAt       string
	PublishedAtParsed time.Time
}

var errInvalidFeedType = errors.New("invalid feed type")

func ParseURL(url string) ([]Item, error) {
	body, err := fromURL(url)
	if err != nil {
		return nil, fmt.Errorf("parsing from url: %w", err)
	}

	items, err := parseRSS(body)
	if err != nil {
		if errors.Is(err, errInvalidFeedType) {
			items, err = parseAtom(body)
			if err != nil {
				return nil, fmt.Errorf("parsing Atom feed: %w", err)
			}
		} else {
			return nil, fmt.Errorf("parsing RSS feed: %w", err)
		}
	}

	return items, nil
}
