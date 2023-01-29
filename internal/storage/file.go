package storage

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

const bufMaxBytes = 1024

type FileStorage struct {
	file     *os.File
	filename string
}

type Row struct {
	Key   string
	Value string
}

func NewFileStorage(filename string) *FileStorage {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic(err)
	}

	return &FileStorage{
		file:     file,
		filename: filename,
	}
}

func (fs *FileStorage) CloseFS() error {
	return fs.file.Close()
}

func (fs *FileStorage) LoadFS() (map[string]string, error) {
	file, err := os.OpenFile(fs.filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	buf := make([]byte, bufMaxBytes)
	scanner.Buffer(buf, bufMaxBytes)
	data := make(map[string]string)

	for scanner.Scan() {
		rawRow := scanner.Bytes()
		var row Row
		err := json.Unmarshal(rawRow, &row)
		if err == nil {
			data[row.Key] = row.Value
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}

	return data, nil
}

func (fs *FileStorage) InsertFS(key string, value string) error {
	row := Row{Key: key, Value: value}
	data, err := json.Marshal(row)
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
