package storage

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync"
)

type Row struct {
	Key   string `db:"id"`
	UUID  string `db:"user_id"`
	Value string `db:"origin_url"`
}

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
	dsn     *sql.DB
}

func NewKV(s Storage) (*StorageDB, error) {
	data, err := s.LoadFS()
	if err != nil {
		return nil, err
	}
	dataMap := make(map[string]Row)
	for item := range data {
		dataMap[data[item].Key] = data[item]
	}
	return &StorageDB{
		urls:    dataMap,
		counter: peekIntegerFromStack(data),
		storage: s,
	}, nil
}

func (db *StorageDB) Insert(baseURL string, sessionID string) (string, error) {
	db.Lock()
	defer db.Unlock()
	db.counter++
	key := strconv.Itoa(db.counter)
	row := Row{
		Key:   key,
		UUID:  sessionID,
		Value: baseURL,
	}
	db.urls[key] = row
	if err := db.storage.InsertFS(row); err != nil {
		return "", err
	}
	return key, nil
}

func (db *StorageDB) Get(ctx context.Context, key string) (string, error) {
	db.RLock()
	defer db.RUnlock()
	row, ok := db.urls[key]
	if !ok {
		return "", errors.New("key not valid")
	}
	return row.Value, nil
}

func (db *StorageDB) GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error) {
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
