package store

import "sync"

type MemoryDB struct {
	storage map[string]bool
	mux     *sync.RWMutex
}

func (s *MemoryDB) New() (Store, error) {
	s.storage = make(map[string]bool)
	s.mux = &sync.RWMutex{}

	return s, nil
}

func (s *MemoryDB) PutKey(key string) error {
	s.mux.Lock()
	s.storage[key] = true
	s.mux.Unlock()

	return nil
}

func (s MemoryDB) KeyExists(key string) (bool, error) {
	s.mux.RLock()
	_, ok := s.storage[key]
	s.mux.RUnlock()

	if ok {
		return true, nil
	}

	return false, nil
}
