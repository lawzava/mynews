package parser

import (
	"encoding/xml"
	"fmt"
)

type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`

	Items []Item `xml:"channel>item"`
}

type RSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

func parseRSS(body []byte) (items []Item, err error) {
	var rssFeed RSSFeed

	if err = xml.Unmarshal(body, &rssFeed); err != nil {
		return nil, fmt.Errorf("failed to read RSS feed: %w", err)
	}

	for _, item := range rssFeed.Items {
		items = append(items, Item{Title: item.Title, Link: item.Link, PublishedAt: item.PublishedAt})
	}

	return items, nil
}
