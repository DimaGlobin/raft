package repository

import (
	"errors"
	"sync"
)

var _ Repository = (*KvRepository)(nil)

type KvRepository struct {
	mu    sync.RWMutex
	store map[string]string
}

func New() Repository {
	return &KvRepository{
		store: make(map[string]string),
	}
}

func (k *KvRepository) Create(key, value string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if _, exists := k.store[key]; exists {
		return errors.New("key already exists")
	}

	k.store[key] = value
	return nil
}

func (k *KvRepository) Get(key string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	value, exists := k.store[key]
	if !exists {
		return "", errors.New("key not found")
	}

	return value, nil
}

func (k *KvRepository) Update(key, value string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if _, exists := k.store[key]; !exists {
		return errors.New("key not found")
	}

	k.store[key] = value
	return nil
}

func (k *KvRepository) Delete(key string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if _, exists := k.store[key]; !exists {
		return errors.New("key not found")
	}

	delete(k.store, key)
	return nil
}
