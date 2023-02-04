package storage

import (
	"errors"
	"strconv"
	"sync"
)

type Storage interface {
	LoadFS() ([]Row, error)
	InsertFS(value Row) error
	CloseFS() error
}

type StorageDB struct {
	sync.RWMutex
	urls    map[string]Row
	counter int
	storage Storage
}

func New(s Storage) (*StorageDB, error) {
	data, err := s.LoadFS()
	if err != nil {
		return nil, err
	}
	dataMap := make(map[string]Row)
	for item := range data {
		dataMap[data[item].Key] = data[item]
	}
	return &StorageDB{
		counter: peekIntegerFromStack(data),
		urls:    dataMap,
		storage: s,
	}, nil
}

func (db *StorageDB) Insert(baseURL string, sessionID string) (string, error) {
	db.Lock()
	defer db.Unlock()

	db.counter++
	key := strconv.Itoa(db.counter)
	row := Row{
		UUID:  sessionID,
		Key:   key,
		Value: baseURL,
	}
	db.urls[key] = row
	if err := db.storage.InsertFS(row); err != nil {
		return "", err
	}
	return key, nil
}

func (db *StorageDB) Get(key string) (string, error) {
	db.RLock()
	defer db.RUnlock()
	row, ok := db.urls[key]
	if !ok {
		return "", errors.New("key not valid")
	}
	return row.Value, nil
}

func (db *StorageDB) GetUserURLs(sessionID string) (map[string]string, error) {
	data := make(map[string]string)
	for _, row := range db.urls {
		if row.UUID == sessionID {
			data[row.Key] = row.Value
		}
	}
	return data, nil
}

func peekIntegerFromStack(data []Row) int {
	peekValue := 1
	for item := range data {
		keyString := data[item].Key
		keyInteger, err := strconv.Atoi(keyString)
		if err == nil && peekValue < keyInteger {
			peekValue = keyInteger
		}
	}
	return peekValue
}
