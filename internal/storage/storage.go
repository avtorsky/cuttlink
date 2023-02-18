package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func NewDB(db *sql.DB) (*StorageDB, error) {
	return &StorageDB{dsn: db}, nil
}

func (db *StorageDB) Ping(ctx context.Context) error {
	if db.dsn != nil {
		return db.dsn.PingContext(ctx)
	}
	return errors.New("dsn not defined")
}

func (db *StorageDB) Insert(ctx context.Context, baseURL string, sessionID string) (string, error) {
	if db.dsn != nil {
		query := "INSERT INTO cuttlink(user_id, origin_url) VALUES($1, $2) RETURNING id"
		var id string
		if err := db.dsn.QueryRowContext(ctx, query, sessionID, baseURL).Scan(&id); err != nil {
			return "", err
		}
		return fmt.Sprint(id), nil
	}
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
	if db.dsn != nil {
		query := "SELECT origin_url FROM cuttlink WHERE id=$1"
		var rowDB Row
		if err := db.dsn.QueryRowContext(ctx, query, key).Scan(&rowDB.Value); err != nil {
			return "", err
		}
		return rowDB.Value, nil
	}
	db.RLock()
	defer db.RUnlock()
	row, ok := db.urls[key]
	if !ok {
		return "", errors.New("key not valid")
	}
	return row.Value, nil
}

func (db *StorageDB) GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error) {
	if db.dsn != nil {
		query := "SELECT id, user_id, origin_url FROM cuttlink WHERE user_id=$1 ORDER BY id"
		items := make([]Row, 0)
		rows, err := db.dsn.QueryContext(ctx, query, sessionID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var r Row
			err = rows.Scan(&r.Key, &r.UUID, &r.Value)
			if err != nil {
				return nil, err
			}
			items = append(items, r)
		}
		err = rows.Err()
		if err != nil {
			return nil, err
		}
		dataDB := make(map[string]string)
		for item := range items {
			row := items[item]
			dataDB[fmt.Sprintf(row.Key)] = row.Value
		}
		return dataDB, nil
	}
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
