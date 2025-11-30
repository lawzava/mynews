package timeparser

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	errEmptyInputString  = errors.New("input string is empty")
	errFailedtoParseTime = errors.New("failed to parse time")
)

func ParseUTC(timestampString string) (time.Time, error) {
	datetime := strings.TrimSpace(timestampString)
	if datetime == "" {
		return time.Time{}, errEmptyInputString
	}

	for _, f := range &dateFormats {
		parsedTimestamp, err := time.Parse(f, datetime)
		if err == nil {
			return parsedTimestamp, nil
		}
	}

	for _, format := range &dateFormatsWithNamedZone {
		timestamp, err := time.Parse(format, datetime)
		if err != nil {
			continue
		}

		var loc *time.Location

		loc, err = time.LoadLocation(timestamp.Location().String())
		if err != nil {
			return timestamp, fmt.Errorf("failed to parse time: %w", err)
		}

		timestamp, err = time.ParseInLocation(format, timestampString, loc)
		if err == nil {
			return timestamp, nil
		}
	}

	return time.Time{}.UTC(), errFailedtoParseTime
}
