package store

import "errors"

type Repository interface {
	PutKey(key string) error
	KeyExists(key string) (bool, error)
}

func New(storageType Type, accessDetails string) (Repository, error) {
	switch storageType {
	case TypeMemory:
		return newMemory(), nil
	case TypeRedis:
		return newRedis(accessDetails)
	case TypePostgres:
		return newPostgres(accessDetails)
	}

	return nil, errors.New("storage type not supported")
}
