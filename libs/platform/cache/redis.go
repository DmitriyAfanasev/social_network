// Package cache содержит адаптеры кэша для общих сервисных компонентов.
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis реализует кэширование байтовых значений в Redis.
type Redis struct {
	client *redis.Client
}

// NewRedis создаёт Redis-адаптер для кэширования байтовых значений.
func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client}
}

// Get получает значение из кэша или возвращает nil, если ключ отсутствует.
func (c *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return value, err
}

// Set сохраняет значение в кэше на заданное время.
func (c *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete удаляет переданные ключи из кэша.
func (c *Redis) Delete(ctx context.Context, key ...string) error {
	return c.client.Del(ctx, key...).Err()
}
