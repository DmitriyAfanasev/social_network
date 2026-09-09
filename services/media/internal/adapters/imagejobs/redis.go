// Package imagejobs содержит Redis Streams адаптер очереди оптимизации изображений.
package imagejobs

import (
	"context"
	"errors"
	"strings"
	"time"

	"uuid"

	"github.com/redis/go-redis/v9"

	"general-project/media/internal/ports"
)

const mediaIDField = "media_id"

// RedisStream реализует надёжную очередь заданий через Redis Streams и consumer group.
type RedisStream struct {
	client    *redis.Client
	stream    string
	group     string
	consumer  string
	maxLength int64
	retryIdle time.Duration
}

// NewRedisStream создаёт очередь с ограниченной длиной stream.
func NewRedisStream(client *redis.Client, stream string, group string, consumer string, maxLength int64, retryIdle time.Duration) *RedisStream {
	return &RedisStream{client: client, stream: stream, group: group, consumer: consumer, maxLength: maxLength, retryIdle: retryIdle}
}

// Publish добавляет запрос в Redis Stream.
func (q *RedisStream) Publish(ctx context.Context, job ports.ImageOptimizationJob) error {
	_, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		MaxLen: q.maxLength,
		Approx: true,
		Values: map[string]any{mediaIDField: job.MediaID.String()},
	}).Result()
	return err
}

// Run читает задания, включая неподтверждённые сообщения после перезапуска worker.
func (q *RedisStream) Run(ctx context.Context, handler func(context.Context, ports.ImageOptimizationJob) error) error {
	if err := q.ensureGroup(ctx); err != nil {
		return err
	}
	for {
		claimed, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   q.stream,
			Group:    q.group,
			Consumer: q.consumer,
			MinIdle:  q.retryIdle,
			Start:    "0-0",
			Count:    1,
		}).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}
		if len(claimed) > 0 {
			if err := q.handle(ctx, claimed[0], handler); err != nil {
				return err
			}
			continue
		}

		streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    q.group,
			Consumer: q.consumer,
			Streams:  []string{q.stream, ">"},
			Count:    1,
			Block:    time.Second,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := q.handle(ctx, message, handler); err != nil {
					return err
				}
			}
		}
	}
}

func (q *RedisStream) ensureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0").Err()
	if err != nil && !isBusyGroup(err) {
		return err
	}
	return nil
}

func (q *RedisStream) handle(ctx context.Context, message redis.XMessage, handler func(context.Context, ports.ImageOptimizationJob) error) error {
	mediaIDValue, ok := message.Values[mediaIDField]
	mediaIDText, isString := mediaIDValue.(string)
	if !ok || !isString {
		return q.ack(ctx, message.ID)
	}
	mediaID, err := uuid.Parse(mediaIDText)
	if err != nil {
		return q.ack(ctx, message.ID)
	}
	if err := handler(ctx, ports.ImageOptimizationJob{MediaID: mediaID}); err != nil {
		return err
	}
	return q.ack(ctx, message.ID)
}

func (q *RedisStream) ack(ctx context.Context, messageID string) error {
	_, err := q.client.XAck(ctx, q.stream, q.group, messageID).Result()
	return err
}

func isBusyGroup(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}
