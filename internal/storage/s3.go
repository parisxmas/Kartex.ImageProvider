package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/kartex/imageprovider/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client *minio.Client
	bucket string
}

func NewS3Storage() (*S3Storage, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")
	useSSL := os.Getenv("S3_USE_SSL") == "true"

	// Initialize minio client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// Create bucket if it doesn't exist
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &S3Storage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *S3Storage) Save(image *models.Image) error {
	ctx := context.Background()
	_, err := s.client.PutObject(ctx, s.bucket, image.ID+".webp", bytes.NewReader(image.Data), int64(len(image.Data)), minio.PutObjectOptions{
		ContentType: "image/webp",
	})
	return err
}

func (s *S3Storage) Get(id string) (*models.Image, error) {
	ctx := context.Background()
	object, err := s.client.GetObject(ctx, s.bucket, id+".webp", minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, err
	}

	return &models.Image{
		ID:     id,
		Data:   data,
		Format: "webp",
	}, nil
}

func (s *S3Storage) Delete(id string) error {
	ctx := context.Background()
	return s.client.RemoveObject(ctx, s.bucket, id+".webp", minio.RemoveObjectOptions{})
}

func (s *S3Storage) List() ([]string, error) {
	ctx := context.Background()
	var ids []string

	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    "",
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		if filepath.Ext(object.Key) == ".webp" {
			ids = append(ids, object.Key[:len(object.Key)-5]) // Remove .webp extension
		}
	}

	return ids, nil
}
