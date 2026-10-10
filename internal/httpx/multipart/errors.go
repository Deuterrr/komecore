package multipart

import "errors"

var (
	ErrInvalidMultipartForm   = errors.New("invalid multipart form")
	ErrNotFoundMultipartFiles = errors.New("multipart files not found")
)
