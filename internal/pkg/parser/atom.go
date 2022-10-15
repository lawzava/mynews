package parser

import (
	"encoding/xml"
	"fmt"
	"mynews/internal/pkg/timeparser"
	"time"
)

type atomFeed struct {
	XMLName xml.Name   `xml:"http://www.w3.org/2005/Atom feed"`
	Items   []atomItem `xml:"entry"`
}

type atomItem struct {
	Title   string   `xml:"title"`
	Updated string   `xml:"updated"`
	Link    atomLink `xml:"link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

func parseAtom(body []byte) ([]Item, error) {
	var feed atomFeed

	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse Atom feed: %w", err)
	}

	items := make([]Item, len(feed.Items))

	for itemIdx := range feed.Items {
		items[itemIdx] = Item{
			Title:             feed.Items[itemIdx].Title,
			Link:              feed.Items[itemIdx].Link.Href,
			PublishedAt:       feed.Items[itemIdx].Updated,
			PublishedAtParsed: time.Time{},
		}

		publishedAtParsed, err := timeparser.ParseUTC(feed.Items[itemIdx].Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed item publish date: %w", err)
		}

		items[itemIdx].PublishedAtParsed = publishedAtParsed
	}

	return items, nil
}
