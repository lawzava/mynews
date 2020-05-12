package store

import (
	"sync"
	"time"
)

type MemoryDB struct {
	storage map[string]time.Time
	mux     *sync.RWMutex
}

func (s *MemoryDB) New() (Store, error) {
	s.storage = make(map[string]time.Time)
	s.mux = &sync.RWMutex{}

	return s, nil
}

func (s *MemoryDB) PutKey(key string) error {
	s.mux.Lock()
	s.storage[key] = time.Now()
	s.mux.Unlock()

	return nil
}

func (s *MemoryDB) KeyExists(key string) (bool, error) {
	s.mux.RLock()
	_, ok := s.storage[key]
	s.mux.RUnlock()

	if ok {
		s.mux.Lock()
		s.storage[key] = time.Now()
		s.mux.Unlock()

		return true, nil
	}

	return false, nil
}

func (s *MemoryDB) CleanupBefore(before time.Time) {
	s.mux.Lock()
	for key, lastSeenAt := range s.storage {
		if lastSeenAt.Before(before) {
			delete(s.storage, key)
		}
	}
	s.mux.Unlock()
}
