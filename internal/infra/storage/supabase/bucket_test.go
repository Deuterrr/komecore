package supabase

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"komecore/internal/config"
)

func TestEnsureBucket_ErrorHandling(t *testing.T) {
	// Mock server returning HTTP 400 Bad Request with an error JSON object
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"statusCode":"400","error":"Bad Request","message":"Invalid API key"}`))
	}))
	defer server.Close()

	provider := &SupabaseProvider{
		BucketURL: server.URL,
		SupabaseConfig: config.SupabaseConfig{
			ServiceRoleKey: "test-key",
		},
		Client: server.Client(),
	}

	exists, err := provider.EnsureBucket("public-assets")
	if err == nil {
		t.Fatal("expected error from EnsureBucket on HTTP 400, got nil")
	}
	if exists {
		t.Error("expected exists to be false on error")
	}

	// Verify the error is NOT a JSON unmarshal crash
	if strings.Contains(err.Error(), "cannot unmarshal object into Go value") {
		t.Errorf("error contains JSON unmarshal crash: %v", err)
	}

	// Verify error captures the response
	if !strings.Contains(err.Error(), "validate supabase bucket") {
		t.Errorf("expected validate supabase bucket prefix in error, got: %v", err)
	}
}

func TestEnsureBucket_Success(t *testing.T) {
	// Mock server returning HTTP 200 with bucket array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"public-assets"},{"id":"private-assets"}]`))
	}))
	defer server.Close()

	provider := &SupabaseProvider{
		BucketURL: server.URL,
		SupabaseConfig: config.SupabaseConfig{
			ServiceRoleKey: "test-key",
		},
		Client: server.Client(),
	}

	exists, err := provider.EnsureBucket("public-assets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected bucket public-assets to exist")
	}

	notExists, err := provider.EnsureBucket("non-existent-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notExists {
		t.Error("expected bucket non-existent-bucket to not exist")
	}
}

func TestCreateBucket_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"statusCode":"401","error":"Unauthorized","message":"Unauthorized"}`))
	}))
	defer server.Close()

	provider := &SupabaseProvider{
		BucketURL: server.URL,
		SupabaseConfig: config.SupabaseConfig{
			ServiceRoleKey: "test-key",
		},
		Client: server.Client(),
	}

	err := provider.CreateBucket("public-assets", true)
	if err == nil {
		t.Fatal("expected error from CreateBucket on HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "create supabase bucket") {
		t.Errorf("expected create supabase bucket in error, got: %v", err)
	}
}
