package multipart

import (
	"mime/multipart"
)

type File struct {
	File        multipart.File
	Filename    string
	Size        int64
	ContentType string
}

type Image struct {
	File
	Width  int
	Height int
}

type ValidationConfig struct {
	AllowedMIMEs []string
	MaxFileSize  int64
}
