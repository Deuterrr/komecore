package config

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	DSN      string
}

func LoadPostgresConfig() PostgresConfig {
	var (
		host     = GetEnv("POSTGRES_DB_HOST", "")
		port     = GetEnv("POSTGRES_DB_PORT", "")
		name     = GetEnv("POSTGRES_DB_NAME", "")
		user     = GetEnv("POSTGRES_DB_USER", "")
		password = GetEnv("POSTGRES_DB_PASSWORD", "")
		sslMode  = GetEnv("POSTGRES_DB_SSLMODE", "")
	)

	dsn := "postgresql://" +
		user + ":" +
		password + "@" +
		host + ":" +
		port + "/" +
		name

	if sslMode != "" {
		dsn += "?sslmode=" + sslMode
	}

	return PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Name:     name,
		SSLMode:  sslMode,
		DSN:      dsn,
	}
}
