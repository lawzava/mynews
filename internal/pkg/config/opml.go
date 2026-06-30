package config

import (
	"encoding/xml"
	"errors"
	"fmt"
	"mynews/internal/pkg/logger"
	"os"
)

var errNoFeedsInOPML = errors.New("no feeds found in OPML file")

type opmlDoc struct {
	XMLName  xml.Name      `xml:"opml"`
	Outlines []opmlOutline `xml:"body>outline"`
}

type opmlOutline struct {
	XMLURL   string        `xml:"xmlUrl,attr"`
	Outlines []opmlOutline `xml:"outline"`
}

// feedURLs collects this outline's feed URL and those of its nested outlines.
func (o *opmlOutline) feedURLs() []string {
	var urls []string

	if o.XMLURL != "" {
		urls = append(urls, o.XMLURL)
	}

	for idx := range o.Outlines {
		urls = append(urls, o.Outlines[idx].feedURLs()...)
	}

	return urls
}

// importOPMLFile reads an OPML export and writes a new config whose stdout app
// contains every discovered feed. It refuses to clobber an existing config.
func importOPMLFile(opmlPath, configPath string, log *logger.Log) error {
	data, err := os.ReadFile(opmlPath)
	if err != nil {
		return fmt.Errorf("reading OPML file: %w", err)
	}

	var doc opmlDoc

	err = xml.Unmarshal(data, &doc)
	if err != nil {
		return fmt.Errorf("parsing OPML file: %w", err)
	}

	var urls []string

	for idx := range doc.Outlines {
		urls = append(urls, doc.Outlines[idx].feedURLs()...)
	}

	if len(urls) == 0 {
		return errNoFeedsInOPML
	}

	sources := make([]fileStructureSource, len(urls))

	for idx, url := range urls {
		sources[idx] = fileStructureSource{
			URL:                 url,
			IgnoreStoriesBefore: "",
			MustIncludeAnyOf:    nil,
			MustExcludeAnyOf:    nil,
			StatusPage:          false,
			Interests:           nil,
		}
	}

	fileStruct := leanFileStructure(sources)

	log.Info(fmt.Sprintf("Imported %d feeds from '%s'", len(urls), opmlPath))

	return writeConfigFile(configPath, &fileStruct)
}
