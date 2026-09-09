// Package config загружает конфигурацию media-сервиса.
package config

import (
	"time"

	platformconfig "general-project/libs/platform/config"
)

const (
	// VideoTranscodeRequestedTopic — topic команд транскодирования.
	VideoTranscodeRequestedTopic = "video.transcode.requested"
	// VideoTranscodeCompletedTopic — topic результатов транскодирования.
	VideoTranscodeCompletedTopic = "video.transcode.completed"
	// ApplicationEventsTopic — topic интеграционных событий приложения.
	ApplicationEventsTopic = "application.events"
	// ImageOptimizationStream — Redis Stream запросов на оптимизацию изображений.
	ImageOptimizationStream = "media.image-optimization.v1"
	// ImageOptimizationGroup — consumer group обработчиков изображений.
	ImageOptimizationGroup = "media-image-workers"
)

// Config содержит настройки запуска media-сервиса.
type Config struct {
	HTTPAddr                     string
	DatabaseURL                  string
	RedisAddr                    string
	JWTSecret                    string
	ProfilesURL                  string
	SocialURL                    string
	KafkaBrokers                 string
	KafkaGroupID                 string
	KafkaMaxBytes                int
	EventsTopic                  string
	S3Endpoint                   string
	S3Bucket                     string
	S3AccessKey                  string
	S3SecretKey                  string
	S3Secure                     bool
	ImageJobsStream              string
	ImageJobsGroup               string
	ImageJobsMaxLength           int64
	ImageJobsRetryIdle           time.Duration
	ImageWorkerMaxInputBytes     int64
	ImageWorkerMaxPixels         int64
	ImageWorkerMemoryBudgetBytes int64
	ImageWorkerWebPQuality       float32
}

// Load загружает конфигурацию media-сервиса из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:                     platformconfig.Env("MEDIA_HTTP_ADDR", ":8105"),
		DatabaseURL:                  platformconfig.Env("MEDIA_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		RedisAddr:                    platformconfig.Env("MEDIA_REDIS_ADDR", "localhost:6379"),
		JWTSecret:                    platformconfig.Env("MEDIA_JWT_SECRET", "local-identity-secret"),
		ProfilesURL:                  platformconfig.Env("MEDIA_PROFILES_URL", "http://localhost:8102"),
		SocialURL:                    platformconfig.Env("MEDIA_SOCIAL_URL", "http://localhost:8103"),
		KafkaBrokers:                 platformconfig.Env("MEDIA_KAFKA_BROKERS", "localhost:9092"),
		KafkaGroupID:                 platformconfig.Env("MEDIA_KAFKA_GROUP_ID", "media-video-completions"),
		KafkaMaxBytes:                10_000_000,
		EventsTopic:                  platformconfig.Env("MEDIA_EVENTS_TOPIC", ApplicationEventsTopic),
		S3Endpoint:                   platformconfig.Env("MEDIA_S3_ENDPOINT", "localhost:9000"),
		S3Bucket:                     platformconfig.Env("MEDIA_S3_BUCKET", "general-project"),
		S3AccessKey:                  platformconfig.Env("MEDIA_S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:                  platformconfig.Env("MEDIA_S3_SECRET_KEY", "minioadmin"),
		S3Secure:                     platformconfig.Env("MEDIA_S3_SECURE", "false") == "true",
		ImageJobsStream:              platformconfig.Env("MEDIA_IMAGE_JOBS_STREAM", ImageOptimizationStream),
		ImageJobsGroup:               platformconfig.Env("MEDIA_IMAGE_JOBS_GROUP", ImageOptimizationGroup),
		ImageJobsMaxLength:           int64(platformconfig.PositiveInt(platformconfig.Env("MEDIA_IMAGE_JOBS_MAX_LENGTH", "10000"), 10000)),
		ImageJobsRetryIdle:           platformconfig.PositiveDuration(platformconfig.Env("MEDIA_IMAGE_JOBS_RETRY_IDLE", "1m"), time.Minute),
		ImageWorkerMaxInputBytes:     int64(platformconfig.PositiveInt(platformconfig.Env("MEDIA_IMAGE_WORKER_MAX_INPUT_BYTES", "26214400"), 25*1024*1024)),
		ImageWorkerMaxPixels:         int64(platformconfig.PositiveInt(platformconfig.Env("MEDIA_IMAGE_WORKER_MAX_PIXELS", "12000000"), 12_000_000)),
		ImageWorkerMemoryBudgetBytes: int64(platformconfig.PositiveInt(platformconfig.Env("MEDIA_IMAGE_WORKER_MEMORY_BUDGET_BYTES", "268435456"), 256*1024*1024)),
		ImageWorkerWebPQuality:       float32(platformconfig.PositiveInt(platformconfig.Env("MEDIA_IMAGE_WORKER_WEBP_QUALITY", "80"), 80)),
	}
}
