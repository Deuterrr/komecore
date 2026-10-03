package config

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func LoadSMTPConfig() SMTPConfig {
	return SMTPConfig{
		Host:     GetEnv("SMTP_HOST"),
		Port:     GetEnv("SMTP_PORT"),
		Username: GetEnv("SMTP_USERNAME"),
		Password: GetEnv("SMTP_PASSWORD"),
		From:     GetEnv("SMTP_FROM"),
	}
}
