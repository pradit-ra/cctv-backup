package cctv

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
)

type BackupStorage interface {
	Write(location string, data io.ReadCloser) error
}

func NewGCSBackupStorage(bucket string, pathPrefix string) (BackupStorage, error) {
	c, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("create GCS client, %w", err)
	}
	return &gcsBackupStorage{
		bucketH:    c.Bucket(bucket),
		pathPrefix: pathPrefix,
	}, nil
}

func NewFileBackupStorage(pathPrefix string) BackupStorage {
	return &fileBuckupStorage{
		pathPrefix: pathPrefix,
	}
}

type fileBuckupStorage struct {
	pathPrefix string
}

type gcsBackupStorage struct {
	bucketH    *storage.BucketHandle
	pathPrefix string
}

func (g *gcsBackupStorage) Write(location string, data io.ReadCloser) error {
	storage := g.bucketH.Object(fmt.Sprintf("%s/%s", g.pathPrefix, location)).NewWriter(context.Background())
	defer storage.Close()
	// copy data into storage
	return copy(storage, data)
}

func (f *fileBuckupStorage) Write(location string, data io.ReadCloser) error {
	storage, _ := os.Create(fmt.Sprintf("%s/%s", f.pathPrefix, location))
	defer storage.Close()
	// copy data into storage
	return copy(storage, data)
}

func copy(writer io.WriteCloser, reader io.ReadCloser) error {
	size, err := io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("copy video to Backup Storage, %w", err)
	}
	logger.Info("Write Backup video", "size(Mb)", (size / (1024 * 1024)))
	return nil
}
