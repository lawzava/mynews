package store

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // postgres driver
)

type postgres struct {
	db *sql.DB
}

func newPostgres(accessDetails string) (*postgres, error) {
	db, err := sql.Open("postgres", accessDetails)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS parsed_news (story_id varchar(32))") // varchar() should match identity key length
	if err != nil {
		return nil, fmt.Errorf("ensuring initial table: %w", err)
	}

	_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS parsed_news_story_id_idx ON parsed_news(story_id)")
	if err != nil {
		return nil, fmt.Errorf("ensuring unique index on initial table: %w", err)
	}

	return &postgres{db}, nil
}

func (s *postgres) PutKey(key string) error {
	if _, err := s.db.Exec("INSERT INTO parsed_news(story_id) VALUES($1)", key); err != nil {
		return fmt.Errorf("failed to insert new story id: %w", err)
	}

	return nil
}

func (s *postgres) KeyExists(key string) (bool, error) {
	var exists bool

	if err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM parsed_news WHERE story_id = $1)", key).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check story id existence: %w", err)
	}

	return exists, nil
}
