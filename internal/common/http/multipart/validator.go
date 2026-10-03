package multipart

import "errors"

func ValidateFile(
	file *File,
	cfg ValidationConfig,
) error {
	if file.Size > cfg.MaxFileSize {
		return errors.New("file size exceeded")
	}

	allowed := false
	for _, mime := range cfg.AllowedMIMEs {
		if mime == file.ContentType {
			allowed = true
			break
		}
	}

	if !allowed {
		return errors.New("invalid mime type")
	}

	return nil
}
