package storage

import (
	"errors"
	"strconv"
	"sync"
)

type Storage struct {
	sync.RWMutex
	links   map[string]string
	counter int
}

func New() *Storage {
	return &Storage{
		counter: 1,
		links:   map[string]string{},
	}
}

func (m *Storage) Insert(url string) string {
	m.Lock()
	defer m.Unlock()

	m.counter++
	key := strconv.Itoa(m.counter)
	m.links[key] = url

	return key
}

func (m *Storage) Get(key string) (string, error) {
	m.RLock()
	defer m.RUnlock()

	url, is_valid := m.links[key]
	if !is_valid {
		return "", errors.New("Key not valid.")
	}

	return url, nil
}
