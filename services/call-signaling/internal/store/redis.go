/*
Package store persists the small control-plane record of a call in Redis.

Call state is short-lived and shared by all signaling replicas, which makes
Redis a good fit: reads are fast, TTL removes abandoned invitations, and a Lua
script can validate an actor and change a state in one atomic operation. Redis
does not carry media packets and does not replace PostgreSQL authorization.
*/
package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"general-project/call-signaling/internal/domain"
	"github.com/redis/go-redis/v9"
)

const sessionPrefix = "calls:session:"

// RedisStore is the adapter around the calls:session:<id> hash namespace.
type RedisStore struct {
	client *redis.Client
	ttl    int
}

// NewRedisStore creates a store with a TTL expressed in seconds.
func NewRedisStore(client *redis.Client, ttlSeconds int) *RedisStore {
	return &RedisStore{client: client, ttl: ttlSeconds}
}

// Save writes a session without changing its expiry. The signaling handler
// normally uses Create and Transition, which both set/refresh the TTL.
func (s *RedisStore) Save(ctx context.Context, session domain.Session) error {
	key := sessionPrefix + session.CallID
	return s.client.HSet(ctx, key, map[string]any{
		"call_id": session.CallID, "caller_id": session.CallerID, "callee_id": session.CalleeID,
		"call_type": string(session.CallType), "status": string(session.Status),
	}).Err()
}

// Create stores the initial ringing session and assigns its expiration.
func (s *RedisStore) Create(ctx context.Context, session domain.Session) error {
	key := sessionPrefix + session.CallID
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"call_id": session.CallID, "caller_id": session.CallerID, "callee_id": session.CalleeID,
		"call_type": string(session.CallType), "status": string(session.Status),
	})
	pipe.Expire(ctx, key, timeDurationSeconds(s.ttl))
	_, err := pipe.Exec(ctx)
	return err
}

// Get loads a session; redis.Nil means the invitation expired or never existed.
func (s *RedisStore) Get(ctx context.Context, callID string) (domain.Session, error) {
	values, err := s.client.HGetAll(ctx, sessionPrefix+callID).Result()
	if err != nil {
		return domain.Session{}, err
	}
	if len(values) == 0 {
		return domain.Session{}, redis.Nil
	}
	callerID, err := strconv.ParseInt(values["caller_id"], 10, 64)
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid caller_id: %w", err)
	}
	calleeID, err := strconv.ParseInt(values["callee_id"], 10, 64)
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid callee_id: %w", err)
	}
	return domain.Session{CallID: values["call_id"], CallerID: callerID, CalleeID: calleeID, CallType: domain.CallType(values["call_type"]), Status: domain.CallStatus(values["status"])}, nil
}

// Refresh keeps an active call addressable while both browser sockets are alive.
func (s *RedisStore) Refresh(ctx context.Context, callID string) error {
	return s.client.Expire(ctx, sessionPrefix+callID, timeDurationSeconds(s.ttl)).Err()
}

// Transition runs the actor check and state transition in Redis atomically.
// This matters when two replicas receive accept/end at nearly the same time:
// only one command may legally move ringing to active or active to ended.
func (s *RedisStore) Transition(ctx context.Context, callID string, actorID int64, action string) (domain.Session, error) {
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
	result, err := s.client.Eval(ctx, script, []string{sessionPrefix + callID}, actorID, action, s.ttl).Result()
	if err != nil {
		return domain.Session{}, err
	}
	items, ok := result.([]any)
	if !ok || len(items) != 2 {
		return domain.Session{}, errors.New("invalid transition response")
	}
	if fmt.Sprint(items[0]) != "1" {
		return domain.Session{}, fmt.Errorf("%s", fmt.Sprint(items[1]))
	}
	return s.Get(ctx, callID)
}

func timeDurationSeconds(seconds int) time.Duration { return time.Duration(seconds) * time.Second }
