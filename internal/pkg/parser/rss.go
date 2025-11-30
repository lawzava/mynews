package parser

import (
	"encoding/xml"
	"fmt"
	"mynews/internal/pkg/timeparser"
	"time"
)

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`

	Items []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

func parseRSS(body []byte) ([]Item, error) {
	var feed rssFeed

	err := xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, errInvalidFeedType
	}

	items := make([]Item, len(feed.Items))

	for itemIdx := range feed.Items {
		items[itemIdx] = Item{
			Title:             feed.Items[itemIdx].Title,
			Link:              feed.Items[itemIdx].Link,
			PublishedAt:       feed.Items[itemIdx].PubDate,
			PublishedAtParsed: time.Time{},
		}

		publishedParsedAt, err := timeparser.ParseUTC(feed.Items[itemIdx].PubDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed item publish date: %w", err)
		}

		items[itemIdx].PublishedAtParsed = publishedParsedAt
	}

	return items, nil
}
