// Package store хранит короткоживущие control-plane-сессии звонков в Redis.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"general-project/call-signaling/internal/domain"
	"general-project/call-signaling/internal/ports"
	"github.com/redis/go-redis/v9"
)

const sessionPrefix = "calls:session:"

// RedisStore — адаптер пространства hash-ключей calls:session:<id>.
type RedisStore struct {
	client *redis.Client
	ttl    int
}

// NewRedisStore создаёт хранилище с TTL в секундах.
func NewRedisStore(client *redis.Client, ttlSeconds int) *RedisStore {
	return &RedisStore{client: client, ttl: ttlSeconds}
}

// Save сохраняет сессию без изменения срока действия ключа.
func (s *RedisStore) Save(ctx context.Context, session domain.Session) error {
	key := sessionPrefix + session.CallID.String()
	return s.client.HSet(ctx, key, map[string]any{
		"call_id": session.CallID, "caller_id": session.CallerID, "callee_id": session.CalleeID,
		"call_type": string(session.CallType), "status": string(session.Status),
	}).Err()
}

// Create сохраняет начальную сессию и назначает ей срок действия.
func (s *RedisStore) Create(ctx context.Context, session domain.Session) error {
	key := sessionPrefix + session.CallID.String()
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"call_id": session.CallID, "caller_id": session.CallerID, "callee_id": session.CalleeID,
		"call_type": string(session.CallType), "status": string(session.Status),
	})
	pipe.Expire(ctx, key, timeDurationSeconds(s.ttl))
	_, err := pipe.Exec(ctx)
	return err
}

// Get загружает сессию; ErrNotFound означает истёкшее или неизвестное приглашение.
func (s *RedisStore) Get(ctx context.Context, callID uuid.UUID) (domain.Session, error) {
	values, err := s.client.HGetAll(ctx, sessionPrefix+callID.String()).Result()
	if err != nil {
		return domain.Session{}, err
	}
	if len(values) == 0 {
		return domain.Session{}, ports.ErrNotFound
	}
	callIDValue, err := uuid.Parse(values["call_id"])
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid call_id: %w", err)
	}
	callerID, err := uuid.Parse(values["caller_id"])
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid caller_id: %w", err)
	}
	calleeID, err := uuid.Parse(values["callee_id"])
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid callee_id: %w", err)
	}
	return domain.Session{CallID: callIDValue, CallerID: callerID, CalleeID: calleeID, CallType: domain.CallType(values["call_type"]), Status: domain.CallStatus(values["status"])}, nil
}

// Refresh продлевает TTL активного звонка, пока сокеты браузеров подключены.
func (s *RedisStore) Refresh(ctx context.Context, callID uuid.UUID) error {
	updated, err := s.client.Expire(ctx, sessionPrefix+callID.String(), timeDurationSeconds(s.ttl)).Result()
	if err != nil {
		return err
	}
	if !updated {
		return ports.ErrNotFound
	}
	return nil
}

// Transition атомарно проверяет участника и меняет состояние в Redis.
func (s *RedisStore) Transition(ctx context.Context, callID uuid.UUID, actorID uuid.UUID, action string) (domain.Session, error) {
	const script = `
local key = KEYS[1]
if redis.call('EXISTS', key) == 0 then return {0, 'not_found'} end
local status = redis.call('HGET', key, 'status')
local callee = redis.call('HGET', key, 'callee_id')
local caller = redis.call('HGET', key, 'caller_id')
local actor = ARGV[1]
local action = ARGV[2]
if (action == 'accept' or action == 'reject') and callee ~= actor then return {0, 'only_callee'} end
if action == 'accept' and status ~= 'ringing' then return {0, 'invalid_state'} end
if action == 'reject' and status ~= 'ringing' then return {0, 'invalid_state'} end
if action == 'end' and actor ~= callee and actor ~= caller then return {0, 'not_participant'} end
if action == 'end' and status ~= 'ringing' and status ~= 'active' then return {0, 'invalid_state'} end
local next = action == 'accept' and 'active' or action == 'reject' and 'rejected' or 'ended'
redis.call('HSET', key, 'status', next)
redis.call('EXPIRE', key, ARGV[3])
return {1, next}
`
	result, err := s.client.Eval(ctx, script, []string{sessionPrefix + callID.String()}, actorID.String(), action, s.ttl).Result()
	if err != nil {
		return domain.Session{}, err
	}
	items, ok := result.([]any)
	if !ok || len(items) != 2 {
		return domain.Session{}, errors.New("invalid transition response")
	}
	if fmt.Sprint(items[0]) != "1" {
		if fmt.Sprint(items[1]) == "not_found" {
			return domain.Session{}, ports.ErrNotFound
		}
		return domain.Session{}, fmt.Errorf("%s", fmt.Sprint(items[1]))
	}
	return s.Get(ctx, callID)
}

func timeDurationSeconds(seconds int) time.Duration { return time.Duration(seconds) * time.Second }
