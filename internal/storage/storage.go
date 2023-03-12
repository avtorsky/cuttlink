package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/avtorsky/cuttlink/internal/workers"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

const dbResponseTimeout = 10 * time.Second

type Row struct {
	Key       string `db:"id"`
	UUID      string `db:"user_id"`
	Value     string `db:"original_url"`
	IsDeleted bool   `db:"is_deleted"`
}

type DuplicateURLError struct {
	Key string
	Err error
}

type Storager interface {
	GetURL(ctx context.Context, key string) (*Row, error)
	GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error)
	SetURL(ctx context.Context, url string, sessionID string) (string, error)
	SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error)
	UpdateBatchURL(ctx context.Context, task workers.DeleteTask) error
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
	storage *sqlx.DB
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

func NewDB(db *sqlx.DB) (*DB, error) {
	return &DB{storage: db}, nil
}

func (e *DuplicateURLError) Error() string {
	return fmt.Sprintf("key exists: %s, %s", e.Key, e.Err.Error())
}

func (e *DuplicateURLError) Unwrap() error {
	return e.Err
}

func NewDuplicateURLError(key string, err error) error {
	return &DuplicateURLError{
		Key: key,
		Err: err,
	}
}

func (ms *InMemoryStorage) GetURL(ctx context.Context, key string) (*Row, error) {
	ms.RLock()
	defer ms.RUnlock()

	row, ok := ms.urls[key]
	if !ok {
		return nil, errors.New("invalid key")
	}

	return &Row{
		Value:     row.Value,
		IsDeleted: row.IsDeleted,
	}, nil
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
		Key:       key,
		UUID:      sessionID,
		Value:     url,
		IsDeleted: false,
	}
	ms.urls[key] = row

	return key, nil
}

func (ms *InMemoryStorage) SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error) {
	return nil, errors.New("in-memory storage invalid method")
}

func (ms *InMemoryStorage) UpdateBatchURL(ctx context.Context, task workers.DeleteTask) error {
	for _, key := range task.Keys {
		row, ok := ms.urls[key]
		if !ok {
			continue
		}
		if row.UUID != task.UUID {
			continue
		}
		row.IsDeleted = true
		ms.urls[key] = row
	}
	return nil
}

func (ms *InMemoryStorage) Ping(ctx context.Context) error {
	return errors.New("in-memory storage invalid method")
}

func (ms *InMemoryStorage) Close() error {
	return errors.New("in-memory storage invalid method")
}

func (fs *FileStorage) GetURL(ctx context.Context, key string) (*Row, error) {
	fs.RLock()
	defer fs.RUnlock()

	row, ok := fs.urls[key]
	if !ok {
		return nil, errors.New("invalid key")
	}

	return &Row{
		Value:     row.Value,
		IsDeleted: row.IsDeleted,
	}, nil
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
		Key:       key,
		UUID:      sessionID,
		Value:     url,
		IsDeleted: false,
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

func (fs *FileStorage) UpdateBatchURL(ctx context.Context, task workers.DeleteTask) error {
	for _, key := range task.Keys {
		row, ok := fs.urls[key]
		if !ok {
			continue
		}
		if row.UUID != task.UUID {
			continue
		}
		row.IsDeleted = true
		fs.urls[key] = row
		if err := fs.storage.InsertFS(row); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileStorage) Ping(ctx context.Context) error {
	return errors.New("file storage invalid method")
}

func (fs *FileStorage) Close() error {
	return fs.storage.CloseFS()
}

func (db *DB) GetURL(ctx context.Context, key string) (*Row, error) {
	ctxDB, cancel := context.WithTimeout(ctx, dbResponseTimeout)
	defer cancel()

	query := "SELECT original_url, is_deleted FROM cuttlink WHERE id=$1"
	var row Row
	if err := db.storage.GetContext(ctxDB, &row, query, key); err != nil {
		return nil, err
	}

	return &Row{
		Value:     row.Value,
		IsDeleted: row.IsDeleted,
	}, nil
}

func (db *DB) GetUserURLs(ctx context.Context, sessionID string) (map[string]string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, dbResponseTimeout)
	defer cancel()

	query := "SELECT id, user_id, original_url FROM cuttlink WHERE user_id=$1 AND is_deleted=FALSE ORDER BY id"
	items := make([]Row, 0)
	err := db.storage.SelectContext(ctxDB, &items, query, sessionID)
	if err != nil {
		return nil, err
	}

	data := make(map[string]string)
	for item := range items {
		row := items[item]
		data[fmt.Sprint(row.Key)] = row.Value
	}

	return data, nil
}

func (db *DB) SetURL(ctx context.Context, url string, sessionID string) (string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, dbResponseTimeout)
	defer cancel()

	query := "INSERT INTO cuttlink(user_id, original_url) VALUES($1, $2) RETURNING id"
	var id string
	err := db.storage.GetContext(ctxDB, &id, query, sessionID, url)
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		var row Row
		q := "SELECT * FROM cuttlink WHERE original_url=$1"
		if e := db.storage.GetContext(ctx, &row, q, url); e != nil {
			return "", e
		}
		return "", NewDuplicateURLError(row.Key, err)
	} else if err != nil {
		return "", err
	}

	return id, nil
}

func (db *DB) SetBatchURL(ctx context.Context, urlBatch []string, sessionID string) ([]string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, dbResponseTimeout)
	defer cancel()

	if len(urlBatch) == 0 {
		return make([]string, 0), nil
	}
	data := make([]map[string]interface{}, len(urlBatch))
	for item := range urlBatch {
		data[item] = map[string]interface{}{
			"user_id":      sessionID,
			"original_url": urlBatch[item],
		}
	}

	tx, err := db.storage.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := "INSERT INTO cuttlink(user_id, original_url) VALUES(:user_id, :original_url) RETURNING id"
	rows, err := db.storage.NamedQueryContext(ctxDB, query, data)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	result := make([]string, len(urlBatch))
	item := 0
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		result[item] = id
		item++
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (db *DB) UpdateBatchURL(ctx context.Context, task workers.DeleteTask) error {
	ctxDB, cancel := context.WithTimeout(ctx, dbResponseTimeout)
	defer cancel()

	tx, err := db.storage.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "UPDATE cuttlink SET is_deleted = TRUE WHERE id = any($1) AND user_id = $2"
	stmt, err := tx.PrepareContext(ctxDB, query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.ExecContext(ctxDB, task.Keys, task.UUID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) Ping(ctx context.Context) error {
	ctxDB, cancel := context.WithTimeout(ctx, dbResponseTimeout)
	defer cancel()

	return db.storage.PingContext(ctxDB)
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
