package repository

import (
	"ImageProcessor/internal/model"
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ImageStore interface {
	Upload(ctx context.Context, file io.Reader, objectName string, size int64) error
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectName string) error
	GetManyURL(ctx context.Context, images []model.ImageInRepo, expiry time.Duration) ([]string, error)
	GetURL(ctx context.Context, image model.ImageInRepo) (string, error)
}

type ImageStorage struct {
	Client     *minio.Client
	BucketName string
}

func NewImageStorage(endpoint, user, password, bucketName string, sslMode bool) (*ImageStorage, error) {
	ctx := context.Background()

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(user, password, ""),
		Secure: sslMode,
	})

	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	if !exists {
		err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	policy := fmt.Sprintf(`{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": {
                    "AWS": ["*"]
                },
                "Action": ["s3:GetObject"],
                "Resource": ["arn:aws:s3:::%s/*"]
            }
        ]
    }`, bucketName)

	err = client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		return nil, err
	}

	return &ImageStorage{Client: client, BucketName: bucketName}, nil
}

func (i *ImageStorage) Upload(ctx context.Context, file io.Reader, objectName string, size int64) error {
	contentType := "application/octet-stream"
	ext := filepath.Ext(objectName)
	if ext != "" {
		ct := mime.TypeByExtension(ext)
		if ct != "" {
			contentType = ct
		}
	}

	_, err := i.Client.PutObject(ctx, i.BucketName, objectName, file, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return err
	}
	return nil
}

func (i *ImageStorage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	file, err := i.Client.GetObject(ctx, i.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (i *ImageStorage) Delete(ctx context.Context, objectName string) error {
	err := i.Client.RemoveObject(ctx, i.BucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (i *ImageStorage) GetManyURL(ctx context.Context, images []model.ImageInRepo, expiry time.Duration) ([]string, error) {
	urls := make([]string, len(images))
	errCh := make(chan error, len(images))

	var wg sync.WaitGroup
	_, cancel := context.WithCancel(ctx)
	defer cancel()

	for idx, img := range images {
		wg.Add(1)
		go func(index int, processedPath, uploadsPath string) {
			defer wg.Done()
			var objectName string
			if processedPath == "" {
				objectName = uploadsPath
			} else {
				objectName = processedPath
			}

			url, err := i.Client.PresignedGetObject(ctx, i.BucketName, objectName, expiry, nil)
			if err != nil {
				errCh <- err
				cancel()
				return
			}
			urls[idx] = url.String()
		}(idx, img.ProcessedPath, img.UploadsPath)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	errs := make([]error, 0)

	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("get many url error: %v", errs)
	}

	return urls, nil

}

func (i *ImageStorage) GetURL(ctx context.Context, image model.ImageInRepo) (string, error) {
	var objectName string
	if image.ProcessedPath == "" {
		objectName = image.UploadsPath
	} else {
		objectName = image.ProcessedPath
	}
	url, err := i.Client.PresignedGetObject(ctx, i.BucketName, objectName, 7*time.Hour, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
