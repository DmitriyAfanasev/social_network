// Package storage содержит адаптеры S3-совместимого object storage.
package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage реализует object storage порт через MinIO или S3.
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIOStorage создаёт клиент object storage с явными параметрами подключения.
func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, secure bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOStorage{client: client, bucket: bucket}, nil
}

// EnsureBucket создаёт bucket, если его ещё нет.
func (s *MinIOStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

// Put загружает содержимое объекта в bucket.
func (s *MinIOStorage) Put(ctx context.Context, objectKey string, contentType string, content io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, content, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// Get открывает бинарное содержимое объекта для потоковой отдачи клиенту.
func (s *MinIOStorage) Get(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
}

// GetRange открывает указанный байтовый диапазон объекта.
func (s *MinIOStorage) GetRange(ctx context.Context, objectKey string, start int64, end int64) (io.ReadCloser, error) {
	options := minio.GetObjectOptions{}
	if err := options.SetRange(start, end); err != nil {
		return nil, err
	}
	return s.client.GetObject(ctx, s.bucket, objectKey, options)
}

// Delete удаляет объект из bucket.
func (s *MinIOStorage) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}
