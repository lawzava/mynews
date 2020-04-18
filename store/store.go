package store

import (
	"errors"
	"news/validate"
)

type Store interface {
	PutKey(key string) error
	KeyExists(key string) (bool, error)
}

type Config struct {
	Type
	AccessDetails string
}

func (c Config) Validate() error {
	if c.Type == TypePostgres || c.Type == TypeRedis {
		if err := validate.RequiredString(c.AccessDetails, "storage access details"); err != nil {
			return err
		}
	}

	return nil
}

func New(cfg Config) (Store, error) {
	switch cfg.Type {
	case TypeMemory:
		return newMemory(), nil
	case TypeRedis:
		return newRedis(cfg.AccessDetails)
	case TypePostgres:
		return newPostgres(cfg.AccessDetails)
	}

	return nil, errors.New("storage type not supported")
}
