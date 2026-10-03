package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (p *SupabaseProvider) PublicURL(key string, bucket string) string {
	return fmt.Sprintf(
		"%s/storage/v1/object/public/%s/%s",
		strings.TrimRight(p.SupabaseConfig.ProjectURL, "/"),
		strings.Trim(bucket, "/"),
		strings.TrimLeft(key, "/"),
	)
}

func (p *SupabaseProvider) SignedURL(key string) (string, error) {
	payload, err := json.Marshal(map[string]int64{
		"expiresIn": int64(p.StorageConfig.SignedURLExpiry.Seconds()),
	})
	if err != nil {
		return "", fmt.Errorf("encode signed url request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		p.ObjectURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("build signed url request: %w", err)
	}

	token := p.SupabaseConfig.ServiceRoleKey
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("generate signed url from supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", p.newResponseError("generate signed url from supabase", resp)
	}

	var data supabaseSignedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("decode signed url response: %w", err)
	}
	if strings.TrimSpace(data.SignedURL) == "" {
		return "", fmt.Errorf("supabase signed url response is empty")
	}

	return strings.TrimPrefix(data.SignedURL, "/"), nil
}
