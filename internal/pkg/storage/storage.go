package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"mynews/internal/pkg/logger"
	"os"
	"path/filepath"
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

// CleanupAllBefore removes keys last seen before the cutoff across every app,
// bounding storage growth. KeyExists refreshes a key's last-seen time, so keys
// for stories still appearing in a feed are never pruned.
func (s *Storage) CleanupAllBefore(before time.Time) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for app, keys := range s.store {
		for key, lastSeenAt := range keys {
			if lastSeenAt.Before(before) {
				delete(s.store[app], key)
			}
		}
	}
}

// DumpToFile atomically persists the store to filePath. It writes to a temporary
// file in the same directory and renames it into place, so an interrupted dump
// never leaves a truncated or partial data file.
func (s *Storage) DumpToFile(filePath string) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), ".mynews-data-*.tmp")
	if err != nil {
		return fmt.Errorf("initializing data file: %w", err)
	}

	tmpName := tmpFile.Name()

	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename below succeeds

	// Hold the read lock for the snapshot: PutKey/KeyExists/CleanupAllBefore all take
	// the write lock, so without this the encoder races a concurrent map write.
	s.mux.RLock()
	err = json.NewEncoder(tmpFile).Encode(s.store)
	s.mux.RUnlock()

	if err != nil {
		_ = tmpFile.Close()

		return fmt.Errorf("writing to data file: %w", err)
	}

	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("closing data file: %w", err)
	}

	err = os.Rename(tmpName, filePath)
	if err != nil {
		return fmt.Errorf("replacing data file: %w", err)
	}

	return nil
}

func (s *Storage) RecoverFromFile(filePath string, log *logger.Log, legacyAppName string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		log.Warn(fmt.Sprintf("File '%s' does not exist", filePath))

		return nil
	}

	dataFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}

	defer func() { _ = dataFile.Close() }()

	var dataFileContents map[string]any

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

func (s *Storage) parseFileContents(fileContents map[string]any, legacyAppName string) error {
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

		val, ok := value.(map[string]any)
		if !ok {
			return ErrBadInputValue
		}

		for story, timeInterface := range val {
			timeString, ok := timeInterface.(string)
			if !ok {
				return ErrBadTimeValue
			}

			timestamp, err := time.Parse(time.RFC3339, timeString)
			if err != nil {
				return fmt.Errorf("failed to parse mapped time: %w", err)
			}

			if s.store[key] == nil {
				s.store[key] = make(map[string]time.Time)
			}

			s.store[key][story] = timestamp
		}
	}

	return nil
}
