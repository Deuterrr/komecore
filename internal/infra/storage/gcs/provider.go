package gcs

import (
	"context"
	"fmt"
	"path"
	"strings"

	gcsSDK "cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"komecore/internal/config"
	"komecore/internal/infra/storage"
)

type GCSProvider struct {
	StorageConfig config.StorageConfig
	GCSConfig     config.GCSConfig
	Client        *gcsSDK.Client
}

var _ storage.Provider = (*GCSProvider)(nil)

func NewGCSProvider(
	storageCfg config.StorageConfig,
	gcsCfg config.GCSConfig,
) (*GCSProvider, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if strings.TrimSpace(gcsCfg.CredentialsJSON) != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(gcsCfg.CredentialsJSON)))
	} else if strings.TrimSpace(gcsCfg.CredentialsFile) != "" {
		opts = append(opts, option.WithCredentialsFile(gcsCfg.CredentialsFile))
	}

	client, err := gcsSDK.NewClient(ctx, opts...)
	if err != nil {
		// Fallback to unauthenticated client for local development / testing / public access
		var fallbackErr error
		client, fallbackErr = gcsSDK.NewClient(ctx, append(opts, option.WithoutAuthentication())...)
		if fallbackErr != nil {
			return nil, fmt.Errorf("failed to initialize gcs client: %w", err)
		}
	}

	return &GCSProvider{
		StorageConfig: storageCfg,
		GCSConfig:     gcsCfg,
		Client:        client,
	}, nil
}

func (p *GCSProvider) Close() error {
	if p.Client != nil {
		return p.Client.Close()
	}
	return nil
}

func (p *GCSProvider) normalizeObjectKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")

	cleaned := path.Clean(key)
	if cleaned == "." || cleaned == "" {
		return ""
	}

	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return ""
	}

	return cleaned
}
