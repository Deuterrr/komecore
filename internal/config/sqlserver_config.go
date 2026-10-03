package config

type SQLServerConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	DSN      string
}

func LoadSQLServerConfig() SQLServerConfig {
	host := GetEnv("SQLSERVER_DB_HOST", "")
	port := GetEnv("SQLSERVER_DB_PORT", "1433")
	name := GetEnv("SQLSERVER_DB_NAME", "")
	user := GetEnv("SQLSERVER_DB_USER", "")
	password := GetEnv("SQLSERVER_DB_PASSWORD", "")

	dsn := "sqlserver://" + user + ":" + password +
		"@" + host + ":" + port + "?database=" + name

	return SQLServerConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Name:     name,
		DSN:      dsn,
	}
}
