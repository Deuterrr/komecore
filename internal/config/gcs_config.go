package config

type GCSConfig struct {
	ProjectID       string
	Bucket          string
	CredentialsFile string
	CredentialsJSON string
}

func LoadGCSConfig() GCSConfig {
	return GCSConfig{
		ProjectID:       GetEnv("GCS_PROJECT_ID", ""),
		Bucket:          GetEnv("GCS_BUCKET", ""),
		CredentialsFile: GetEnv("GCS_CREDENTIALS_FILE", ""),
		CredentialsJSON: GetEnv("GCS_CREDENTIALS_JSON", ""),
	}
}
