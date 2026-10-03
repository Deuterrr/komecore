package domain

import "errors"

var (
	ErrInvalidProductPrice = errors.New("price must be greater than 0")
	ErrInvalidProductName  = errors.New("name is required")
	ErrInvalidStock        = errors.New("stock cannot be negative")
	ErrInvalidSlug         = errors.New("slug is required")
	ErrProductNotFound     = errors.New("product not found")
)

var (
	ErrInvalidProductImageMime     = errors.New("unsupported product image mime type")
	ErrInvalidProductImageFileName = errors.New("malformed product image filename")
	ErrInvalidProductImageSize     = errors.New("product image exceeds max upload size")
)
