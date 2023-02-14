package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const postgresShema = `
CREATE TABLE IF NOT EXISTS cuttlink (
	id SERIAL,
	user_id VARCHAR(36) NOT NULL,
	origin_url text NOT NULL
)`

func NewDB(db *sql.DB) (*StorageDB, error) {
	return &StorageDB{dsn: db}, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(postgresShema)
	return err
}

func (db *StorageDB) PingDB(ctx context.Context) error {
	return db.dsn.PingContext(ctx)
}

func (db *StorageDB) InsertDB(ctx context.Context, baseURL string, sessionID string) (string, error) {
	query := "INSERT INTO cuttlink(user_id, origin_url) VALUES($1, $2) RETURNING id"
	var id string
	if err := db.dsn.QueryRowContext(ctx, query, sessionID, baseURL).Scan(&id); err != nil {
		return "", err
	}
	return fmt.Sprint(id), nil
}

func (db *StorageDB) GetDB(ctx context.Context, key string) (string, error) {
	query := "SELECT origin_url FROM cuttlink WHERE id=$1"
	var rowDB Row
	if err := db.dsn.QueryRowContext(ctx, query, key).Scan(&rowDB.Value); err != nil {
		return "", err
	}
	return rowDB.Value, nil
}

func (db *StorageDB) GetUserURLsDB(ctx context.Context, sessionID string) (map[string]string, error) {
	query := "SELECT id, origin_url, user_id FROM cuttlink WHERE user_id=$1 ORDER BY id"
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
