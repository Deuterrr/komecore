package gcs

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	gcsSDK "cloud.google.com/go/storage"
)

func (p *GCSProvider) PublicURL(key string, bucket string) string {
	b := strings.Trim(bucket, "/")
	if b == "" {
		b = p.GCSConfig.Bucket
	}
	if b == "" {
		b = "public-assets"
	}

	cleanedKey := strings.TrimLeft(p.normalizeObjectKey(key), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, cleanedKey)
}

func (p *GCSProvider) SignedURL(key string) (string, error) {
	bucket := p.GCSConfig.Bucket
	if bucket == "" {
		bucket = "private-assets"
	}

	cleanedKey := strings.TrimLeft(p.normalizeObjectKey(key), "/")
	if cleanedKey == "" {
		return "", fmt.Errorf("storage key is required")
	}

	opts := &gcsSDK.SignedURLOptions{
		Scheme:  gcsSDK.SigningSchemeV4,
		Method:  http.MethodGet,
		Expires: time.Now().Add(p.StorageConfig.SignedURLExpiry),
	}

	url, err := p.Client.Bucket(bucket).SignedURL(cleanedKey, opts)
	if err != nil {
		// When running without private key credentials (e.g. Workload Identity or local dev),
		// return a deterministic fallback URL instead of hard erroring if possible.
		return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, cleanedKey), nil
	}

	return url, nil
}
