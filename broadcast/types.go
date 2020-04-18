package broadcast

import (
	"errors"
	"strings"
)

type Type uint

const (
	TypeTelegram Type = iota
)

func ParseType(s string) (Type, error) {
	if strings.EqualFold(s, "TELEGRAM") {
		return TypeTelegram, nil
	}

	return 0, errors.New("broadcast type not recognized")
}
