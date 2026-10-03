package supabase

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"komecore/internal/config"
)

type SupabaseProvider struct {
	StorageConfig  config.StorageConfig
	SupabaseConfig config.SupabaseConfig
	Client         *http.Client

	ObjectURL          string
	BucketURL          string
	BucketAnalyticsURL string
	S3URL              string
	TransformationURL  string
	ResumableURL       string
	CDNURL             string
	HealthURL          string
	IcebergURL         string
	VectorURL          string
}

func NewSupabaseProvider(
	StorageConfig config.StorageConfig,
	SupabaseConfig config.SupabaseConfig,
	Client *http.Client,
) (*SupabaseProvider, error) {
	if strings.TrimSpace(SupabaseConfig.ProjectURL) == "" {
		return nil, fmt.Errorf("supabase project url is required")
	}
	if strings.TrimSpace(SupabaseConfig.ServiceRoleKey) == "" {
		return nil, fmt.Errorf("supabase service role key is required")
	}
	if Client == nil {
		Client = http.DefaultClient
	}

	baseURL, err := url.Parse(strings.TrimRight(SupabaseConfig.ProjectURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse supabase project url: %w", err)
	}

	basePath := path.Join(baseURL.Path, "storage", "v1")

	objectURL := *baseURL
	objectURL.Path = path.Join(basePath, "object")

	bucketURL := *baseURL
	bucketURL.Path = path.Join(basePath, "bucket")

	provider := &SupabaseProvider{
		StorageConfig:  StorageConfig,
		SupabaseConfig: SupabaseConfig,
		Client:         Client,

		ObjectURL: objectURL.String(),
		BucketURL: bucketURL.String(),
	}

	return provider, nil
}

type supabaseSignedURLResponse struct {
	SignedURL string `json:"signedURL"`
}

func (p *SupabaseProvider) newResponseError(action string, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = "status " + strconv.Itoa(resp.StatusCode)
	}

	return fmt.Errorf("%s: %s", action, message)
}
