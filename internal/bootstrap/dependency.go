package bootstrap

import (
	"fmt"
	"net/http"
	"strings"

	"komecore/internal/config"
	"komecore/internal/infra/cache"
	database "komecore/internal/infra/db"
	paymentgateway "komecore/internal/infra/payment-gateway"
	midtransGateway "komecore/internal/infra/payment-gateway/midtrans"
	"komecore/internal/infra/shipping"
	manualShipProvider "komecore/internal/infra/shipping/manual"
	"komecore/internal/infra/storage"
	gcsStorage "komecore/internal/infra/storage/gcs"
	supabaseStorage "komecore/internal/infra/storage/supabase"
	"komecore/internal/infra/transactor"
)

type Dependency struct {
	DB                  *database.Connection
	StorageProvider     storage.Provider
	TransactionProvider transactor.Transactor
	TransactionExecutor transactor.Executor
	PaymentGateway      paymentgateway.Provider
	ShippingProvider    shipping.ShippingProvider
	LogisticsProvider   shipping.LogisticsProvider
	Cache               cache.Cache
}

func NewDependency(cfg Config) (*Dependency, error) {
	storageProvider, err := ResolveStorageProvider(
		cfg.Storage,
		&http.Client{},
	)
	if err != nil {
		return nil, err
	}

	gateway, err := ResolvePaymentGateway(cfg.PaymentGateway)
	if err != nil {
		return nil, err
	}

	db, err := ResolveDatabaseConnection(cfg.DB)
	if err != nil {
		return nil, err
	}

	shippingEstimator := shipping.NewSimpleProvider()

	logistics := manualShipProvider.NewManualShippingProvider()

	cacheStore := ResolveCache(cfg.Redis)

	return &Dependency{
		DB:                  db,
		StorageProvider:     storageProvider,
		TransactionProvider: db.Transactor,
		TransactionExecutor: db.Executor,
		PaymentGateway:      gateway,
		ShippingProvider:    shippingEstimator,
		LogisticsProvider:   logistics,
		Cache:               cacheStore,
	}, nil
}

func (i *Dependency) Close() {
	if i == nil {
		return
	}

	if i.Cache != nil {
		_ = i.Cache.Close()
	}

	if i.DB != nil {
		i.DB.Close()
	}
}

// ResolveCache initializes Redis cache or falls back to MemoryCache / Noop.
func ResolveCache(cfg config.RedisConfig) cache.Cache {
	if !cfg.Enabled || strings.TrimSpace(cfg.Host) == "" {
		return cache.NewNoopCache()
	}

	redisCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		// Fallback to thread-safe in-memory cache for offline/test environments
		return cache.NewMemoryCache()
	}
	return redisCache
}

// ResolveStorageProvider selects and initializes the storage provider based on configuration.
// It supports "supabase", "gcs" (or "google"), and falls back to "noop".
func ResolveStorageProvider(storageCfg config.StorageConfig, httpClient *http.Client) (storage.Provider, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	switch strings.ToLower(strings.TrimSpace(storageCfg.Provider)) {
	case "gcs", "google", "google-storage":
		gcsCfg := config.LoadGCSConfig()
		return gcsStorage.NewGCSProvider(storageCfg, gcsCfg)

	case "supabase":
		supaCfg := config.LoadSupabaseConfig()
		if strings.TrimSpace(supaCfg.ProjectURL) == "" || strings.TrimSpace(supaCfg.ServiceRoleKey) == "" {
			return storage.NewNoopProvider(), nil
		}
		return supabaseStorage.NewSupabaseProvider(storageCfg, supaCfg, httpClient)

	case "noop":
		return storage.NewNoopProvider(), nil

	default:
		// Backward compatibility: credential presence detection
		supaCfg := config.LoadSupabaseConfig()
		if strings.TrimSpace(supaCfg.ProjectURL) != "" && strings.TrimSpace(supaCfg.ServiceRoleKey) != "" {
			return supabaseStorage.NewSupabaseProvider(storageCfg, supaCfg, httpClient)
		}
		gcsCfg := config.LoadGCSConfig()
		if strings.TrimSpace(gcsCfg.ProjectID) != "" || strings.TrimSpace(gcsCfg.CredentialsFile) != "" || strings.TrimSpace(gcsCfg.CredentialsJSON) != "" {
			return gcsStorage.NewGCSProvider(storageCfg, gcsCfg)
		}
		return storage.NewNoopProvider(), nil
	}
}

// ResolvePaymentGateway selects and initializes the payment gateway provider based on configuration.
// It supports "midtrans" and falls back to "noop".
func ResolvePaymentGateway(cfg config.PaymentGatewayConfig) (paymentgateway.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "midtrans":
		midtransCfg := config.LoadMidTransConfig()
		return midtransGateway.NewMidtransAPIProvider(midtransCfg)

	case "noop":
		return paymentgateway.NewNoopProvider(), nil

	default:
		midtransCfg := config.LoadMidTransConfig()
		return midtransGateway.NewMidtransAPIProvider(midtransCfg)
	}
}

// ResolveDatabaseConnection initializes the database connection based on dialect.
func ResolveDatabaseConnection(cfg config.DatabaseConfig) (*database.Connection, error) {
	if cfg.DSN == nil || *cfg.DSN == "" {
		return nil, fmt.Errorf("database: DSN is not configured")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Dialect)) {
	case "sqlserver":
		return database.NewSQLServerConnection(*cfg.DSN)

	case "postgres", "":
		return database.NewPostgresConnection(*cfg.DSN)

	default:
		return nil, fmt.Errorf("database: unsupported dialect %q", cfg.Dialect)
	}
}
