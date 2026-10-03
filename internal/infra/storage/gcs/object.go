package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	gcsSDK "cloud.google.com/go/storage"
	"golang.org/x/sync/errgroup"

	"komecore/internal/infra/storage"
)

func (p *GCSProvider) Upload(
	input storage.UploadInput,
) (*storage.ObjectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := p.normalizeObjectKey(input.Key)
	if key == "" {
		return nil, fmt.Errorf("storage key is required")
	}

	bucket := input.Bucket
	if bucket == "" {
		bucket = p.GCSConfig.Bucket
	}
	if bucket == "" {
		bucket = "public-assets"
	}

	obj := p.Client.Bucket(bucket).Object(key)
	w := obj.NewWriter(ctx)
	if input.ContentType != "" {
		w.ContentType = input.ContentType
	}

	if _, err := io.Copy(w, input.File); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("upload to gcs failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close gcs upload failed: %w", err)
	}

	return &storage.ObjectResponse{
		Key:         key,
		ContentType: input.ContentType,
	}, nil
}

func (p *GCSProvider) UploadMany(
	inputs []storage.UploadInput,
) ([]*storage.ObjectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	if len(inputs) == 0 {
		return nil, nil
	}

	results := make([]*storage.ObjectResponse, len(inputs))
	var mu sync.Mutex

	g, _ := errgroup.WithContext(ctx)

	for i, input := range inputs {
		g.Go(func() error {
			resp, err := p.Upload(input)
			if err != nil {
				return err
			}

			mu.Lock()
			results[i] = resp
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func (p *GCSProvider) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	key = p.normalizeObjectKey(key)
	if key == "" {
		return fmt.Errorf("storage key is required")
	}

	bucket := p.GCSConfig.Bucket
	if bucket == "" {
		bucket = "public-assets"
	}

	err := p.Client.Bucket(bucket).Object(key).Delete(ctx)
	if err != nil && !errors.Is(err, gcsSDK.ErrObjectNotExist) {
		return fmt.Errorf("delete from gcs failed: %w", err)
	}

	return nil
}

func (p *GCSProvider) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	key = p.normalizeObjectKey(key)
	if key == "" {
		return false, fmt.Errorf("storage key is required")
	}

	bucket := p.GCSConfig.Bucket
	if bucket == "" {
		bucket = "public-assets"
	}

	_, err := p.Client.Bucket(bucket).Object(key).Attrs(ctx)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gcsSDK.ErrObjectNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("check object in gcs failed: %w", err)
}
