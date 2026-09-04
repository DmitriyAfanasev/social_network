// Package events содержит Kafka-адаптеры media-сервиса.
package events

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"general-project/media/internal/ports"
)

// KafkaPublisher публикует команды video-worker в Kafka.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher создаёт publisher для заданных брокера и topic.
func NewKafkaPublisher(broker string, topic string) *KafkaPublisher {
	return &KafkaPublisher{writer: &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}}}
}

// Publish публикует команду на транскодирование видео.
func (p *KafkaPublisher) Publish(ctx context.Context, request ports.TranscodeRequest) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Key: []byte(request.VideoID.String()), Value: payload})
}

// Close закрывает writer и освобождает сетевые ресурсы.
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
