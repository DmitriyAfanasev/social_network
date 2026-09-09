package events

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/segmentio/kafka-go"

	"general-project/media/internal/ports"
)

// KafkaConsumer принимает результаты транскодирования от video-worker.
type KafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer создаёт consumer для topic и consumer group.
func NewKafkaConsumer(broker string, topic string, groupID string, maxBytes int) *KafkaConsumer {
	return &KafkaConsumer{reader: kafka.NewReader(kafka.ReaderConfig{Brokers: []string{broker}, Topic: topic, GroupID: groupID, MinBytes: 1, MaxBytes: maxBytes})}
}

// Run читает completion-события и подтверждает их после успешной обработки.
func (c *KafkaConsumer) Run(ctx context.Context, handler func(context.Context, ports.TranscodeCompletion) error) (runErr error) {
	defer func() {
		if err := c.reader.Close(); err != nil && runErr == nil {
			runErr = err
		}
	}()
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		var completion ports.TranscodeCompletion
		if err := json.Unmarshal(message.Value, &completion); err != nil {
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				return err
			}
			continue
		}
		if err := handler(ctx, completion); err != nil {
			return err
		}
		if err := c.reader.CommitMessages(ctx, message); err != nil {
			return err
		}
	}
}
