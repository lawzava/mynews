package store

type Config struct {
	MemoryDB
	PostgresDB
	RedisDB
}

type Store interface {
	New() (Store, error)
	PutKey(key string) error
	KeyExists(key string) (bool, error)
}
