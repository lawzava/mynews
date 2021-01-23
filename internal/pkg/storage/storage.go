package storage

import (
	"encoding/json"
	"fmt"
	"mynews/internal/pkg/logger"
	"os"
	"sync"
	"time"
)

type Storage struct {
	store map[string]time.Time
	mux   *sync.RWMutex
}

func New() Storage {
	var s Storage

	s.store = make(map[string]time.Time)
	s.mux = &sync.RWMutex{}

	return s
}

func (s *Storage) PutKey(key string) error {
	s.mux.Lock()
	s.store[key] = time.Now()
	s.mux.Unlock()

	return nil
}

func (s *Storage) KeyExists(key string) (bool, error) {
	s.mux.RLock()
	_, ok := s.store[key]
	s.mux.RUnlock()

	if ok {
		s.mux.Lock()
		s.store[key] = time.Now()
		s.mux.Unlock()

		return true, nil
	}

	return false, nil
}

func (s *Storage) CleanupBefore(before time.Time) {
	s.mux.Lock()
	for key, lastSeenAt := range s.store {
		if lastSeenAt.Before(before) {
			delete(s.store, key)
		}
	}
	s.mux.Unlock()
}

func (s *Storage) DumpToFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsExist(err) {
		err = os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("removing old dump file: %w", err)
		}
	}

	dataFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("initializing data file: %w", err)
	}

	defer func() { _ = dataFile.Close() }()

	if err = json.NewEncoder(dataFile).Encode(s.store); err != nil {
		return fmt.Errorf("writing to data file: %w", err)
	}

	return nil
}

func (s *Storage) RecoverFromFile(filePath string, log *logger.Log) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Warn(fmt.Sprintf("File '%s' does not exist", filePath))

		return nil
	}

	dataFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}

	defer func() { _ = dataFile.Close() }()

	jsonParser := json.NewDecoder(dataFile)
	if err = jsonParser.Decode(&s.store); err != nil {
		return fmt.Errorf("decoding config file: %w", err)
	}

	return nil
}
