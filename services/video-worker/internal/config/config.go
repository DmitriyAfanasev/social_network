package config

import (
	"log/slog"
	"os"
	"strconv"
)

const (
	VideoTranscodeRequestedTopic = "video.transcode.requested"
	VideoTranscodeCompletedTopic = "video.transcode.completed"
)

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

func Load() Config {
	return Config{
		KafkaBrokers:  env("KAFKA_BROKERS", "localhost:9092"),
		KafkaGroupID:  env("KAFKA_GROUP_ID", "video-worker"),
		KafkaMaxBytes: positiveInt(env("KAFKA_MAX_BYTES", "10000000"), 10000000),
		S3Endpoint:    env("S3_ENDPOINT", "localhost:9000"),
		S3Bucket:      env("S3_BUCKET", "general-project"),
		S3AccessKey:   env("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:   env("S3_SECRET_KEY", "minioadmin"),
		S3Secure:      env("S3_SECURE", "false") == "true",
		ParallelJobs:  positiveInt(env("VIDEO_PARALLEL_JOBS", "3"), 3),
		LogLevel:      parseLevel(env("LOG_LEVEL", "INFO")),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
func parseLevel(value string) slog.Level {
	switch value {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
