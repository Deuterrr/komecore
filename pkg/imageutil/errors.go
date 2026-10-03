package imageutil

import "errors"

var (
	ErrInvalidImageMime     = errors.New("unsupported image mime type")
	ErrInvalidImageFileName = errors.New("malformed image filename")
	ErrInvalidImageSize     = errors.New("image exceeds max upload size")
)
