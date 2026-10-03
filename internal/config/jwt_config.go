package config

import "time"

type JWTConfig struct {
	Secret string
	Exp    time.Duration
}

func LoadJWTConfig() JWTConfig {
	return JWTConfig{
		Secret: GetEnv("JWT_SECRET"),
		Exp:    24 * time.Minute,
	}
}
