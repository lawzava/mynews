package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"mynews/internal/pkg/logger"
	"os"
	"sync"
	"time"
)

type Storage struct {
	store map[string]map[string]time.Time
	mux   *sync.RWMutex
}

func New() Storage {
	var s Storage

	s.store = make(map[string]map[string]time.Time)
	s.mux = &sync.RWMutex{}

	return s
}

func (s *Storage) PutKey(app, key string) error {
	s.mux.Lock()

	if s.store[app] == nil {
		s.store[app] = make(map[string]time.Time)
	}

	s.store[app][key] = time.Now()

	s.mux.Unlock()

	return nil
}

func (s *Storage) KeyExists(app, key string) (bool, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.store[app] == nil {
		s.store[app] = make(map[string]time.Time)
	}

	if _, ok := s.store[app][key]; ok {
		s.store[app][key] = time.Now()

		return true, nil
	}

	return false, nil
}

func (s *Storage) CleanupBefore(app string, before time.Time) {
	s.mux.Lock()
	if s.store[app] == nil {
		s.store[app] = make(map[string]time.Time)
	}

	for key, lastSeenAt := range s.store[app] {
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

func (s *Storage) RecoverFromFile(filePath string, log *logger.Log, legacyAppName string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Warn(fmt.Sprintf("File '%s' does not exist", filePath))

		return nil
	}

	dataFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}

	defer func() { _ = dataFile.Close() }()

	var dataFileContents map[string]interface{}

	err = json.NewDecoder(dataFile).Decode(&dataFileContents)
	if err != nil {
		return fmt.Errorf("decoding config file: %w", err)
	}

	return s.parseFileContents(dataFileContents, legacyAppName)
}

var (
	ErrBadInputValue = errors.New("bad input value")
	ErrBadTimeValue  = errors.New("bad time value")
)

func (s *Storage) parseFileContents(fileContents map[string]interface{}, legacyAppName string) error {
	for key, value := range fileContents {
		if val, ok := value.(string); ok {
			if s.store[legacyAppName] == nil {
				s.store[legacyAppName] = make(map[string]time.Time)
			}

			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			s.store[legacyAppName][key] = t

			continue
		}

		val, ok := value.(map[string]interface{})
		if !ok {
			return ErrBadInputValue
		}

		for story, timeInterface := range val {
			timeString, ok := timeInterface.(string)
			if !ok {
				return ErrBadTimeValue
			}

			t, err := time.Parse(time.RFC3339, timeString)
			if err != nil {
				return fmt.Errorf("failed to parse mapped time: %w", err)
			}

			if s.store[key] == nil {
				s.store[key] = make(map[string]time.Time)
			}

			s.store[key][story] = t
		}
	}

	return nil
}
