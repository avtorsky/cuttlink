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
	Value string `db:"original_url"`
}

type Storager interface {
	GetURL(ctx context.Context, key string) (string, error)
	GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error)
	SetURL(ctx context.Context, url string, sessionID string) (string, error)
	SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error)
	Ping(ctx context.Context) error
	Close() error
}

type InMemoryStorage struct {
	sync.RWMutex
	urls    map[string]Row
	counter int
}

type FileStorage struct {
	sync.RWMutex
	urls    map[string]Row
	counter int
	storage *File
}

type DB struct {
	sync.RWMutex
	storage *sql.DB
}

func NewInMemoryStorage() (*InMemoryStorage, error) {
	data := make(map[string]Row)
	return &InMemoryStorage{
		urls:    data,
		counter: 1,
	}, nil
}

func NewFileStorage(fs *File) (*FileStorage, error) {
	store, err := fs.LoadFS()
	if err != nil {
		return nil, err
	}
	data := make(map[string]Row)
	for item := range store {
		data[store[item].Key] = store[item]
	}
	return &FileStorage{
		urls:    data,
		counter: peekIntegerFromStack(store),
		storage: fs,
	}, nil
}

func NewDB(db *sql.DB) (*DB, error) {
	return &DB{storage: db}, nil
}

func (ms *InMemoryStorage) GetURL(ctx context.Context, key string) (string, error) {
	ms.RLock()
	defer ms.RUnlock()
	row, ok := ms.urls[key]
	if !ok {
		return "", errors.New("invalid key")
	}
	return row.Value, nil
}

func (ms *InMemoryStorage) GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error) {
	data := make(map[string]string)
	for _, row := range ms.urls {
		if row.UUID == sessionID {
			data[row.Key] = row.Value
		}
	}
	return data, nil
}

func (ms *InMemoryStorage) SetURL(ctx context.Context, url string, sessionID string) (string, error) {
	ms.Lock()
	defer ms.Unlock()
	ms.counter++
	key := strconv.Itoa(ms.counter)
	row := Row{
		Key:   key,
		UUID:  sessionID,
		Value: url,
	}
	ms.urls[key] = row
	return key, nil
}

func (ms *InMemoryStorage) SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error) {
	return nil, errors.New("in-memory storage invalid method")
}

func (ms *InMemoryStorage) Ping(ctx context.Context) error {
	return errors.New("in-memory storage invalid method")
}

func (ms *InMemoryStorage) Close() error {
	return errors.New("in-memory storage invalid method")
}

func (fs *FileStorage) GetURL(ctx context.Context, key string) (string, error) {
	fs.RLock()
	defer fs.RUnlock()
	row, ok := fs.urls[key]
	if !ok {
		return "", errors.New("invalid key")
	}
	return row.Value, nil
}

func (fs *FileStorage) GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error) {
	data := make(map[string]string)
	for _, row := range fs.urls {
		if row.UUID == sessionID {
			data[row.Key] = row.Value
		}
	}
	return data, nil
}

func (fs *FileStorage) SetURL(ctx context.Context, url string, sessionID string) (string, error) {
	fs.Lock()
	defer fs.Unlock()
	fs.counter++
	key := strconv.Itoa(fs.counter)
	row := Row{
		Key:   key,
		UUID:  sessionID,
		Value: url,
	}
	fs.urls[key] = row
	if err := fs.storage.InsertFS(row); err != nil {
		return "", err
	}
	return key, nil
}

func (fs *FileStorage) SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error) {
	return nil, errors.New("file storage invalid method")
}

func (fs *FileStorage) Ping(ctx context.Context) error {
	return errors.New("file storage invalid method")
}

func (fs *FileStorage) Close() error {
	return fs.storage.CloseFS()
}

func (db *DB) GetURL(ctx context.Context, key string) (string, error) {
	query := "SELECT original_url FROM cuttlink WHERE id=$1"
	var row Row
	if err := db.storage.QueryRowContext(ctx, query, key).Scan(&row.Value); err != nil {
		return "", err
	}
	return row.Value, nil
}

func (db *DB) GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error) {
	query := "SELECT id, user_id, original_url FROM cuttlink WHERE user_id=$1 ORDER BY id"
	items := make([]Row, 0)
	rows, err := db.storage.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row Row
		err = rows.Scan(&row.Key, &row.UUID, &row.Value)
		if err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	data := make(map[string]string)
	for item := range items {
		row := items[item]
		data[fmt.Sprintf(row.Key)] = row.Value
	}
	return data, nil
}

func (db *DB) SetURL(ctx context.Context, url string, sessionID string) (string, error) {
	query := "INSERT INTO cuttlink(user_id, original_url) VALUES($1, $2) RETURNING id"
	var id string
	if err := db.storage.QueryRowContext(ctx, query, sessionID, url).Scan(&id); err != nil {
		return "", err
	}
	return fmt.Sprint(id), nil
}

// to be done after checkout to iter12
func (db *DB) SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error) {
	return nil, errors.New("database invalid method")
}

func (db *DB) Ping(ctx context.Context) error {
	return db.storage.PingContext(ctx)
}

func (db *DB) Close() error {
	return db.storage.Close()
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
