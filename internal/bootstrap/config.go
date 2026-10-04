package bootstrap

import (
	"komecore/internal/config"
)

type Config struct {
	App            config.AppConfig
	Logistics      config.LogisticsConfig
	JWT            config.JWTConfig
	Storage        config.StorageConfig
	PaymentGateway config.PaymentGatewayConfig
	DB             config.DatabaseConfig
	DBTarget       string
	SMTP           config.SMTPConfig
	GoogleOAuth    config.GoogleOAuthConfig
	PaymentSync    config.PaymentSyncConfig
	PaymentExpiry  config.PaymentExpiryConfig
	Redis          config.RedisConfig
}

func LoadConfig() Config {
	config.LoadDotEnv()

	dbTarget := config.GetEnv("DB_TARGET", "postgres")
	dbConf := config.ResolveDBConfig(dbTarget)

	return Config{
		App:            config.LoadAppConfig(),
		Logistics:      config.LoadLogisticsConfig(),
		JWT:            config.LoadJWTConfig(),
		Storage:        config.LoadStorageConfig(),
		PaymentGateway: config.LoadPaymentGatewayConfig(),
		DB:             dbConf,
		DBTarget:       dbTarget,
		SMTP:           config.LoadSMTPConfig(),
		GoogleOAuth:    config.LoadGoogleOAuthConfig(),
		PaymentSync:    config.LoadPaymentSyncConfig(),
		PaymentExpiry:  config.LoadPaymentExpiryConfig(),
		Redis:          config.LoadRedisConfig(),
	}
}
