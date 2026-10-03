package storage

import (
	"io"
)

type Bucket struct {
	Name   string
	Public bool
}

type UploadInput struct {
	Bucket        string
	Key           string
	File          io.Reader
	ContentType   string
	ContentLength int64
}

type ObjectResponse struct {
	Key         string
	ContentType string
}

type ObjectStorage interface {
	Upload(input UploadInput) (*ObjectResponse, error)
	UploadMany(inputs []UploadInput) ([]*ObjectResponse, error)
	Delete(key string) error
	Exists(key string) (bool, error)
}

type URLResolver interface {
	PublicURL(key string, bucket string) string
	SignedURL(key string) (string, error)
}

type BucketManager interface {
	EnsureBucket(name string) (bool, error)
	CreateBucket(name string, public bool) error
}

type Provider interface {
	ObjectStorage
	URLResolver
	BucketManager
}
