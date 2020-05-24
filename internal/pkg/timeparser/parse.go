package timeparser

import (
	"errors"
	"strings"
	"time"
)

var (
	errEmptyInputString  = errors.New("input string is empty")
	errFailedtoParseTime = errors.New("failed to parse time")
)

func ParseUTC(ts string) (t time.Time, err error) {
	d := strings.TrimSpace(ts)
	if d == "" {
		return t, errEmptyInputString
	}

	defer func() { t = t.UTC() }()

	for _, f := range &dateFormats {
		if t, err = time.Parse(f, d); err == nil {
			return
		}
	}

	for _, f := range &dateFormatsWithNamedZone {
		t, err = time.Parse(f, d)
		if err != nil {
			continue
		}

		var loc *time.Location

		loc, err = time.LoadLocation(t.Location().String())
		if err != nil {
			return t, nil
		}

		if t, err = time.ParseInLocation(f, ts, loc); err == nil {
			return t, nil
		}
	}

	return t, errFailedtoParseTime
}
