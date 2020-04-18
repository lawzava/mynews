package store

type memory struct {
	storage map[string]bool
}

func newMemory() *memory {
	storage := make(map[string]bool)

	return &memory{storage}
}

func (s *memory) PutKey(key string) error {
	s.storage[key] = true
	return nil
}

func (s *memory) KeyExists(key string) (bool, error) {
	if _, ok := s.storage[key]; ok {
		return true, nil
	}

	return false, nil
}
