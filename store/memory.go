package store

type MemoryDB struct {
	storage map[string]bool
}

func (s MemoryDB) New() (Store, error) {
	s.storage = make(map[string]bool)
	return &s, nil
}

func (s *MemoryDB) PutKey(key string) error {
	s.storage[key] = true
	return nil
}

func (s MemoryDB) KeyExists(key string) (bool, error) {
	if _, ok := s.storage[key]; ok {
		return true, nil
	}

	return false, nil
}
