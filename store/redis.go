package store

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

type redisDB struct {
	conn redis.Conn
}

func newRedis(accessDetails string) (*redisDB, error) {
	conn, err := redis.DialURL(accessDetails)
	if err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &redisDB{
		conn: conn,
	}, nil
}

func (s *redisDB) PutKey(key string) error {
	_, err := s.conn.Do("SET", key, true)
	if err != nil {
		return fmt.Errorf("adding key '%s' to storage: %w", key, err)
	}

	return nil
}

func (s *redisDB) KeyExists(key string) (bool, error) {
	found, err := redis.Int(s.conn.Do("EXISTS", key))
	if err != nil {
		return false, fmt.Errorf("checking for key '%s' existence: %w", key, err)
	}

	if found > 0 {
		return true, nil
	}

	return false, nil
}
