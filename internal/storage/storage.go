package storage

import (
	"errors"
	"strconv"
	"sync"
)

type Storage interface {
	Insert(baseURL string) string
	Get(key string) (string, error)
}

type StorageDB struct {
	sync.RWMutex
	urls    map[string]string
	counter int
}

func New() StorageDB {
	return StorageDB{
		counter: 1,
		urls:    map[string]string{},
	}
}

func (db *StorageDB) Insert(baseURL string) string {
	db.Lock()
	defer db.Unlock()

	db.counter++
	key := strconv.Itoa(db.counter)
	db.urls[key] = baseURL

	return key
}

func (db *StorageDB) Get(key string) (string, error) {
	db.RLock()
	defer db.RUnlock()

	baseURL, ok := db.urls[key]
	if !ok {
		return "", errors.New("key not valid")
	}

	return baseURL, nil
}
