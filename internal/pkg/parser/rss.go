package parser

import (
	"encoding/xml"
	"fmt"
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

	for _, item := range r.Items {
		items = append(items, Item{Title: item.Title, Link: item.Link, PublishedAt: item.PubDate})
	}

	return items, nil
}
