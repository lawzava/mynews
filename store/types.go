package store

import (
	"errors"
	"strings"
)

type Type uint

const (
	TypeMemory Type = iota
	TypeRedis
	TypePostgres
)

func ParseType(s string) (Type, error) {
	switch strings.ToUpper(s) {
	case "MEMORY":
		return TypeMemory, nil
	case "REDIS":
		return TypeRedis, nil
	case "POSTGRES":
		return TypePostgres, nil
	}

	return 0, errors.New("storage type not recognized")
}
