package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (p *SupabaseProvider) EnsureBucket(name string) (bool, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		p.BucketURL,
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("build bucket validation request: %w", err)
	}

	token := p.SupabaseConfig.ServiceRoleKey
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", token)

	resp, err := p.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("validate supabase bucket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return false, p.newResponseError("validate supabase bucket", resp)
	}

	var buckets []struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&buckets); err != nil {
		return false, fmt.Errorf("decode bucket response: %w", err)
	}

	bucketName := name
	for _, b := range buckets {
		if b.ID == bucketName {
			return true, nil
		}
	}

	return false, nil
}

func (p *SupabaseProvider) CreateBucket(name string, public bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload := map[string]any{
		"id":     name,
		"name":   name,
		"public": public,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal bucket payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.BucketURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build create bucket request: %w", err)
	}

	token := p.SupabaseConfig.ServiceRoleKey
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("create bucket failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return p.newResponseError("create supabase bucket", resp)
	}

	return nil
}
