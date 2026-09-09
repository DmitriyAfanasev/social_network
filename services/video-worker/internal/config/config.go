// Package config содержит конфигурацию video-worker.
package config

import (
	platformconfig "general-project/libs/platform/config"
	"log/slog"
)

const (
	// VideoTranscodeRequestedTopic — Kafka topic входящих заданий.
	VideoTranscodeRequestedTopic = "video.transcode.requested"
	// VideoTranscodeCompletedTopic — Kafka topic результатов транскодирования.
	VideoTranscodeCompletedTopic = "video.transcode.completed"
)

// Config содержит настройки video-worker.
type Config struct {
	KafkaBrokers  string
	KafkaGroupID  string
	KafkaMaxBytes int
	S3Endpoint    string
	S3Bucket      string
	S3AccessKey   string
	S3SecretKey   string
	S3Secure      bool
	ParallelJobs  int
	LogLevel      slog.Level
}

// Load загружает настройки video-worker из окружения.
func Load() Config {
	return Config{
		KafkaBrokers:  platformconfig.Env("KAFKA_BROKERS", "localhost:9092"),
		KafkaGroupID:  platformconfig.Env("KAFKA_GROUP_ID", "video-worker"),
		KafkaMaxBytes: platformconfig.PositiveInt(platformconfig.Env("KAFKA_MAX_BYTES", "10000000"), 10000000),
		S3Endpoint:    platformconfig.Env("S3_ENDPOINT", "localhost:9000"),
		S3Bucket:      platformconfig.Env("S3_BUCKET", "general-project"),
		S3AccessKey:   platformconfig.Env("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:   platformconfig.Env("S3_SECRET_KEY", "minioadmin"),
		S3Secure:      platformconfig.Env("S3_SECURE", "false") == "true",
		ParallelJobs:  platformconfig.PositiveInt(platformconfig.Env("VIDEO_PARALLEL_JOBS", "3"), 3),
		LogLevel:      platformconfig.LogLevel(platformconfig.Env("LOG_LEVEL", "INFO")),
	}
}
