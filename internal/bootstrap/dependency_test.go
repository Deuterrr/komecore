package bootstrap_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"komecore/internal/bootstrap"
	"komecore/internal/config"
	"komecore/internal/infra/paymentgateway"
	"komecore/internal/infra/storage"
	"komecore/internal/infra/storage/gcs"
	"komecore/internal/infra/storage/supabase"
)

// --- Database Resolver Tests ---

func TestResolveDatabaseConnection_Validation(t *testing.T) {
	t.Run("nil DSN returns error", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			Dialect: "postgres",
			DSN:     nil,
		}
		conn, err := bootstrap.ResolveDatabaseConnection(cfg)
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "DSN is not configured")
	})

	t.Run("empty DSN returns error", func(t *testing.T) {
		empty := ""
		cfg := config.DatabaseConfig{
			Dialect: "postgres",
			DSN:     &empty,
		}
		conn, err := bootstrap.ResolveDatabaseConnection(cfg)
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "DSN is not configured")
	})

	t.Run("unsupported dialect returns error", func(t *testing.T) {
		dsn := "some-dsn"
		cfg := config.DatabaseConfig{
			Dialect: "mysql",
			DSN:     &dsn,
		}
		conn, err := bootstrap.ResolveDatabaseConnection(cfg)
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "unsupported dialect")
	})
}

// --- Payment Gateway Resolver Tests ---

func TestResolvePaymentGateway_Noop(t *testing.T) {
	cfg := config.PaymentGatewayConfig{
		Provider: "noop",
	}

	provider, err := bootstrap.ResolvePaymentGateway(cfg)
	require.NoError(t, err)
	assert.IsType(t, &paymentgateway.NoopProvider{}, provider)
	assert.Equal(t, "noop", provider.Name())
}

func TestResolvePaymentGateway_Midtrans(t *testing.T) {
	t.Setenv("MIDTRANS_IS_PRODUCTION", "false")
	t.Setenv("MIDTRANS_SERVER_KEY", "SB-Mid-server-test123")

	cfg := config.PaymentGatewayConfig{
		Provider: "midtrans",
	}

	provider, err := bootstrap.ResolvePaymentGateway(cfg)
	require.NoError(t, err)
	assert.Equal(t, "midtrans", provider.Name())
}

// --- Storage Provider Resolver Tests ---

func TestResolveStorageProvider_ExplicitNoop(t *testing.T) {
	storageCfg := config.StorageConfig{
		Provider:        "noop",
		SignedURLExpiry: 15 * time.Minute,
	}

	provider, err := bootstrap.ResolveStorageProvider(storageCfg, http.DefaultClient)
	require.NoError(t, err)
	assert.IsType(t, &storage.NoopProvider{}, provider)
}

func TestResolveStorageProvider_Supabase(t *testing.T) {
	t.Setenv("SUPABASE_PROJECT_URL", "https://testproject.supabase.co")
	t.Setenv("SUPABASE_SUPA_KEY", "secret-key")

	storageCfg := config.StorageConfig{
		Provider:        "supabase",
		SignedURLExpiry: 15 * time.Minute,
	}

	provider, err := bootstrap.ResolveStorageProvider(storageCfg, http.DefaultClient)
	require.NoError(t, err)
	assert.IsType(t, &supabase.SupabaseProvider{}, provider)
}

func TestResolveStorageProvider_Supabase_MissingCredentials_FallsBackToNoop(t *testing.T) {
	t.Setenv("SUPABASE_PROJECT_URL", "")
	t.Setenv("SUPABASE_SUPA_KEY", "")

	storageCfg := config.StorageConfig{
		Provider:        "supabase",
		SignedURLExpiry: 15 * time.Minute,
	}

	provider, err := bootstrap.ResolveStorageProvider(storageCfg, http.DefaultClient)
	require.NoError(t, err)
	assert.IsType(t, &storage.NoopProvider{}, provider)
}

func TestResolveStorageProvider_GCS(t *testing.T) {
	t.Setenv("GCS_PROJECT_ID", "test-gcp-project")
	t.Setenv("GCS_BUCKET", "public-assets")

	storageCfg := config.StorageConfig{
		Provider:        "gcs",
		SignedURLExpiry: 15 * time.Minute,
	}

	provider, err := bootstrap.ResolveStorageProvider(storageCfg, http.DefaultClient)
	require.NoError(t, err)
	assert.IsType(t, &gcs.GCSProvider{}, provider)
}

func TestResolveStorageProvider_AutoDetect_Supabase(t *testing.T) {
	t.Setenv("SUPABASE_PROJECT_URL", "https://testproject.supabase.co")
	t.Setenv("SUPABASE_SUPA_KEY", "secret-key")

	storageCfg := config.StorageConfig{
		Provider: "", // unconfigured, should auto-detect
	}

	provider, err := bootstrap.ResolveStorageProvider(storageCfg, http.DefaultClient)
	require.NoError(t, err)
	assert.IsType(t, &supabase.SupabaseProvider{}, provider)
}

func TestResolveStorageProvider_AutoDetect_NoCredentials(t *testing.T) {
	t.Setenv("SUPABASE_PROJECT_URL", "")
	t.Setenv("SUPABASE_SUPA_KEY", "")
	t.Setenv("GCS_PROJECT_ID", "")
	t.Setenv("GCS_CREDENTIALS_FILE", "")
	t.Setenv("GCS_CREDENTIALS_JSON", "")

	storageCfg := config.StorageConfig{
		Provider: "",
	}

	provider, err := bootstrap.ResolveStorageProvider(storageCfg, http.DefaultClient)
	require.NoError(t, err)
	assert.IsType(t, &storage.NoopProvider{}, provider)
}
