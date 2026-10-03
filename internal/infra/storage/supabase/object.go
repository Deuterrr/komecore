package supabase

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"komecore/internal/infra/storage"

	"golang.org/x/sync/errgroup"
)

func (p *SupabaseProvider) Upload(
	input storage.UploadInput,
) (*storage.ObjectResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/%s",
		p.ObjectURL,
		strings.TrimRight(input.Bucket, "/"),
		strings.TrimRight(input.Key, "/"),
	)
	req, err := http.NewRequest(
		http.MethodPost,
		endpoint,
		input.File,
	)
	if err != nil {
		return nil, fmt.Errorf("build upload request: %w", err)
	}

	token := p.SupabaseConfig.ServiceRoleKey
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", token)
	req.Header.Set("x-upsert", "true")
	if input.ContentType != "" {
		req.Header.Set("Content-Type", input.ContentType)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload object to supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, p.newResponseError("upload object to supabase", resp)
	}

	key := p.normalizeObjectKey(input.Key)
	if key == "" {
		return nil, fmt.Errorf("storage key is required")
	}

	return &storage.ObjectResponse{
		Key:         key,
		ContentType: input.ContentType,
	}, nil
}

func (p *SupabaseProvider) UploadMany(
	inputs []storage.UploadInput,
) ([]*storage.ObjectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	if len(inputs) == 0 {
		return nil, nil
	}

	results := make([]*storage.ObjectResponse, len(inputs))
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

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

func (p *SupabaseProvider) Delete(key string) error {
	key = p.normalizeObjectKey(key)
	if key == "" {
		return fmt.Errorf("storage key is required")
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		p.ObjectURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf("build delete request: %w", err)
	}

	token := p.SupabaseConfig.ServiceRoleKey
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", token)

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("delete object from supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return p.newResponseError("delete object from supabase", resp)
	}

	return nil
}

func (p *SupabaseProvider) Exists(key string) (bool, error) {
	key = p.normalizeObjectKey(key)
	if key == "" {
		return false, fmt.Errorf("storage key is required")
	}

	req, err := http.NewRequest(
		http.MethodGet,
		p.ObjectURL,
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("build exists request: %w", err)
	}

	token := p.SupabaseConfig.ServiceRoleKey
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", token)

	resp, err := p.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("check object in supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return false, p.newResponseError("check object in supabase", resp)
	}

	return true, nil
}

func (p *SupabaseProvider) normalizeObjectKey(key string) string {
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
