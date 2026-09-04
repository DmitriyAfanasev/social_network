// Package redisadapter содержит Redis-адаптеры realtime messaging-сервиса.
package redisadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/redis/go-redis/v9"

	"general-project/messaging/internal/ports"
)

const (
	presenceKey          = "messaging:v1:presence"
	presenceChannel      = "messaging:v1:presence:events"
	presenceMaxAge       = 30 * time.Second
	realtimeChannel      = "messaging:v1:realtime"
	defaultInflightLimit = 1024
)

var (
	// ErrBackpressure означает, что Redis realtime admission limit исчерпан.
	ErrBackpressure = errors.New("messaging realtime backpressure limit exceeded")
	acquireInflight = `
		local current = redis.call('INCRBY', KEYS[1], 1)
		redis.call('EXPIRE', KEYS[1], 30)
		if current > tonumber(ARGV[1]) then
			redis.call('DECRBY', KEYS[1], 1)
			return 0
		end
		return 1`
	releaseInflight = `local current = redis.call('DECRBY', KEYS[1], 1) if current < 0 then redis.call('SET', KEYS[1], 0) end return current`
)

// RedisRealtimeBroker реализует межинстансовый Pub/Sub через Redis.
type RedisRealtimeBroker struct {
	client  *redis.Client
	channel string
	pubsub  *redis.PubSub
	limit   int
}

// NewRedisRealtimeBroker создаёт Redis Pub/Sub adapter.
func NewRedisRealtimeBroker(client *redis.Client) *RedisRealtimeBroker {
	return NewRedisRealtimeBrokerWithLimit(client, defaultInflightLimit)
}

// NewRedisRealtimeBrokerWithLimit создаёт broker с Redis-backed admission limit.
func NewRedisRealtimeBrokerWithLimit(client *redis.Client, limit int) *RedisRealtimeBroker {
	if limit < 1 {
		limit = defaultInflightLimit
	}
	return &RedisRealtimeBroker{client: client, channel: realtimeChannel, limit: limit}
}

// Publish публикует событие всем экземплярам messaging-сервиса.
func (b *RedisRealtimeBroker) Publish(ctx context.Context, event ports.RealtimeEvent) error {
	if len(event.RecipientIDs) > 1000 || len(event.Payload) > 256*1024 {
		return ErrBackpressure
	}
	key := b.channel + ":inflight"
	allowed, err := b.client.Eval(ctx, acquireInflight, []string{key}, b.limit).Int()
	if err != nil {
		return err
	}
	if allowed == 0 {
		return ErrBackpressure
	}
	defer func() {
		_ = b.client.Eval(ctx, releaseInflight, []string{key}).Err()
	}()
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, b.channel, payload).Err()
}

// Subscribe принимает события из Redis и передаёт их локальному hub.
func (b *RedisRealtimeBroker) Subscribe(ctx context.Context, handler func(context.Context, ports.RealtimeEvent) error) error {
	b.pubsub = b.client.Subscribe(ctx, b.channel)
	defer b.pubsub.Close()
	if _, err := b.pubsub.Receive(ctx); err != nil {
		return err
	}
	messages := b.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case message, ok := <-messages:
			if !ok {
				return nil
			}
			var event ports.RealtimeEvent
			if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
				continue
			}
			if err := handler(ctx, event); err != nil {
				return err
			}
		}
	}
}

// Close закрывает Pub/Sub-соединение.
func (b *RedisRealtimeBroker) Close() error {
	if b.pubsub == nil {
		return nil
	}
	return b.pubsub.Close()
}

// RedisPresenceStore хранит presence в сортированном Redis-множестве.
type RedisPresenceStore struct {
	client *redis.Client
}

// NewRedisPresenceStore создаёт Redis-хранилище presence.
func NewRedisPresenceStore(client *redis.Client) *RedisPresenceStore {
	return &RedisPresenceStore{client: client}
}

// Heartbeat обновляет timestamp пользователя и удаляет протухшие записи.
func (p *RedisPresenceStore) Heartbeat(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	now := time.Now().UnixMilli()
	if err := p.client.ZAdd(ctx, presenceKey, redis.Z{Score: float64(now), Member: userID.String()}).Err(); err != nil {
		return err
	}
	cutoff := time.Now().Add(-ttl).UnixMilli()
	if err := p.client.ZRemRangeByScore(ctx, presenceKey, "-inf", fmt.Sprintf("%d", cutoff)).Err(); err != nil {
		return err
	}
	payload, err := json.Marshal(ports.PresenceEvent{UserID: userID, Online: true})
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, presenceChannel, payload).Err()
}

// IsOnline проверяет наличие свежего heartbeat пользователя.
func (p *RedisPresenceStore) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	now := time.Now().UnixMilli()
	if err := p.client.ZRemRangeByScore(ctx, presenceKey, "-inf", fmt.Sprintf("%d", now-presenceMaxAge.Milliseconds())).Err(); err != nil {
		return false, err
	}
	score, err := p.client.ZScore(ctx, presenceKey, userID.String()).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return score >= float64(now-presenceMaxAge.Milliseconds()), nil
}

// SubscribePresence принимает presence-события от всех экземпляров messaging.
func (p *RedisPresenceStore) SubscribePresence(ctx context.Context, handler func(context.Context, ports.PresenceEvent) error) error {
	pubsub := p.client.Subscribe(ctx, presenceChannel)
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}
	events := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case message, ok := <-events:
			if !ok {
				return nil
			}
			var event ports.PresenceEvent
			if err := json.Unmarshal([]byte(message.Payload), &event); err != nil || event.UserID == uuid.Nil() {
				continue
			}
			if err := handler(ctx, event); err != nil {
				return err
			}
		}
	}
}
