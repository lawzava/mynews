package parser

import (
	"encoding/xml"
	"fmt"
	"mynews/internal/pkg/timeparser"
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

func parseRSS(body []byte) (items []Item, err error) {
	var r rssFeed

	if err = xml.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("failed to read RSS feed: %w", err)
	}

	for _, feedItem := range r.Items {
		item := Item{Title: feedItem.Title, Link: feedItem.Link, PublishedAt: feedItem.PubDate}

		item.PublishedAtParsed, err = timeparser.ParseUTC(feedItem.PubDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed item publish date: %w", err)
		}

		items = append(items, item)
	}

	return items, nil
}
