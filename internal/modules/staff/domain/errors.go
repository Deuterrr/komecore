package domain

import "errors"

var (
	ErrNotFoundStaff    = errors.New("staff not found")
	ErrInvalidEmail     = errors.New("email is required")
	ErrInsufficientRole = errors.New("insufficient role for operation")
)
