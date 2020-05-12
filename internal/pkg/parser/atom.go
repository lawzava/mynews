package parser

import (
	"encoding/xml"
	"fmt"
	"mynews/internal/pkg/timeparser"
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

func parseAtom(body []byte) (items []Item, err error) {
	var r atomFeed

	if err = xml.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("failed to parse Atom feed: %w", err)
	}

	for _, feedItem := range r.Items {
		item := Item{Title: feedItem.Title, Link: feedItem.Link.Href, PublishedAt: feedItem.Updated}

		item.PublishedAtParsed, err = timeparser.ParseUTC(feedItem.Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed item publish date: %w", err)
		}

		items = append(items, item)
	}

	return items, nil
}
