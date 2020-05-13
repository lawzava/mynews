package storage

import (
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
