package store

import (
	"fmt"
	"mynews/internal/pkg/validate"

	"github.com/gomodule/redigo/redis"
)

type RedisDB struct {
	conn     redis.Conn
	RedisURI string
}

func (s RedisDB) New() (Store, error) {
	if err := validate.RequiredString(s.RedisURI, "redis access details"); err != nil {
		return nil, err
	}

	conn, err := redis.DialURL(s.RedisURI)
	if err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	s.conn = conn

	return s, nil
}

func (s RedisDB) PutKey(key string) error {
	_, err := s.conn.Do("SET", key, true)
	if err != nil {
		return fmt.Errorf("adding key '%s' to storage: %w", key, err)
	}

	return nil
}

func (s RedisDB) KeyExists(key string) (bool, error) {
	found, err := redis.Int(s.conn.Do("EXISTS", key))
	if err != nil {
		return false, fmt.Errorf("checking for key '%s' existence: %w", key, err)
	}

	if found > 0 {
		return true, nil
	}

	return false, nil
}
