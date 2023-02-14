package storage

import (
	"bufio"
	"encoding/json"
	"os"
)

const bufMaxBytes = 1024

type FileStorage struct {
	file     *os.File
	filename string
}

func NewFileStorage(filename string) (*FileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	return &FileStorage{
		file:     file,
		filename: filename,
	}, nil
}

func (fs *FileStorage) CloseFS() error {
	return fs.file.Close()
}

func (fs *FileStorage) LoadFS() ([]Row, error) {
	file, err := os.OpenFile(fs.filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	buf := make([]byte, bufMaxBytes)
	scanner.Buffer(buf, bufMaxBytes)
	data := make([]Row, 0)
	for scanner.Scan() {
		rawRow := scanner.Bytes()
		var row Row
		err := json.Unmarshal(rawRow, &row)
		if err == nil {
			data = append(data, row)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return data, nil
}

func (fs *FileStorage) InsertFS(value Row) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = fs.file.Write(data)
	if err != nil {
		return err
	}
	err = fs.file.Sync()
	if err != nil {
		return err
	}
	return nil
}
