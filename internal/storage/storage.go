package storage

import (
	"errors"
	"strconv"
	"sync"
)

type Storage interface {
	LoadFS() (map[string]string, error)
	InsertFS(key string, value string) error
	CloseFS() error
}

type StorageDB struct {
	sync.RWMutex
	urls    map[string]string
	counter int
	storage Storage
}

func New(file Storage) *StorageDB {
	data, err := file.LoadFS()
	if err != nil {
		panic(err)
	}
	return &StorageDB{
		counter: peekIntegerFromStack(data),
		urls:    data,
		storage: file,
	}
}

func (db *StorageDB) Insert(baseURL string) (string, error) {
	db.Lock()
	defer db.Unlock()

	db.counter++
	key := strconv.Itoa(db.counter)
	db.urls[key] = baseURL
	if err := db.storage.InsertFS(key, baseURL); err != nil {
		return "", err
	}

	return key, nil
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

func peekIntegerFromStack(data map[string]string) int {
	peekValue := 1
	for keyString := range data {
		keyInteger, err := strconv.Atoi(keyString)
		if err == nil && peekValue < keyInteger {
			peekValue = keyInteger
		}
	}

	return peekValue
}
