package gcs_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"komecore/internal/config"
	"komecore/internal/infra/storage/gcs"
)

func TestGCSProvider_PublicURL(t *testing.T) {
	provider, err := gcs.NewGCSProvider(
		config.StorageConfig{
			SignedURLExpiry: 15 * time.Minute,
		},
		config.GCSConfig{
			ProjectID: "my-project",
			Bucket:    "default-bucket",
		},
	)
	require.NoError(t, err)
	defer provider.Close()

	tests := []struct {
		name     string
		key      string
		bucket   string
		expected string
	}{
		{
			name:     "custom bucket with standard key",
			key:      "products/img_thumb.webp",
			bucket:   "public-assets",
			expected: "https://storage.googleapis.com/public-assets/products/img_thumb.webp",
		},
		{
			name:     "default bucket fallback when empty",
			key:      "avatars/user1.png",
			bucket:   "",
			expected: "https://storage.googleapis.com/default-bucket/avatars/user1.png",
		},
		{
			name:     "handles leading slashes and backslashes in key",
			key:      "\\products/banners/hero.webp",
			bucket:   "public-assets",
			expected: "https://storage.googleapis.com/public-assets/products/banners/hero.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := provider.PublicURL(tt.key, tt.bucket)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestGCSProvider_SignedURL_Fallback(t *testing.T) {
	provider, err := gcs.NewGCSProvider(
		config.StorageConfig{
			SignedURLExpiry: 15 * time.Minute,
		},
		config.GCSConfig{
			ProjectID: "my-project",
			Bucket:    "my-private-bucket",
		},
	)
	require.NoError(t, err)
	defer provider.Close()

	url, err := provider.SignedURL("invoices/inv_123.pdf")
	require.NoError(t, err)
	assert.Contains(t, url, "invoices/inv_123.pdf")
}
