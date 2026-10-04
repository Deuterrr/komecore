package config

import (
	"fmt"
	"strconv"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Enabled  bool
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func LoadRedisConfig() RedisConfig {
	host := GetEnv("REDIS_HOST", "127.0.0.1")
	port := GetEnv("REDIS_PORT", "6379")
	password := GetEnv("REDIS_PASSWORD", "")
	dbStr := GetEnv("REDIS_DB", "0")
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		db = 0
	}
	enabledStr := GetEnv("REDIS_ENABLED", "true")
	enabled := enabledStr == "true" || enabledStr == "1"

	return RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
		Enabled:  enabled,
	}
}
