package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Store описывает минимальный контракт хранилища кэша.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// JSON предоставляет версионированные ключи и JSON-сериализацию для read-моделей.
type JSON struct {
	store     Store
	namespace string
	version   string
}

// NewJSON создаёт JSON-кэш с изолированным namespace и версией ключей.
func NewJSON(store Store, namespace string, version string) (*JSON, error) {
	if store == nil {
		return nil, errors.New("cache store is required")
	}
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(version) == "" {
		return nil, errors.New("cache namespace and version are required")
	}
	return &JSON{store: store, namespace: namespace, version: version}, nil
}

// Key строит явный ключ кэша из namespace, версии и частей ресурса.
func (c *JSON) Key(parts ...string) string {
	key := c.namespace + ":" + c.version
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// Get декодирует read-модель; false означает отсутствие ключа.
func (c *JSON) Get(ctx context.Context, key string, target any) (bool, error) {
	payload, err := c.store.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if payload == nil {
		return false, nil
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return false, fmt.Errorf("decode cached value: %w", err)
	}
	return true, nil
}

// Set сериализует и сохраняет read-модель с ограниченным TTL.
func (c *JSON) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("cache TTL must be positive")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cached value: %w", err)
	}
	return c.store.Set(ctx, key, payload, ttl)
}

// Delete удаляет read-модели после изменения источника данных.
func (c *JSON) Delete(ctx context.Context, keys ...string) error {
	return c.store.Delete(ctx, keys...)
}
