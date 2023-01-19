package storage

import (
	"errors"
	"strconv"
	"sync"
)

type Storage interface {
	Insert(baseURL string) (string, error)
	Get(key string) (string, error)
}

type StorageDB struct {
	sync.RWMutex
	urls    map[string]string
	counter int
	storage FileStorageSignature
}

func New(file FileStorageSignature) *StorageDB {
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

func (db *StorageDB) Insert(baseURL string) string {
	db.Lock()
	defer db.Unlock()

	db.counter++
	key := strconv.Itoa(db.counter)
	db.urls[key] = baseURL
	db.storage.InsertFS(key, baseURL)

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
