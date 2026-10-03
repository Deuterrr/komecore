package imageutil

import (
	"image"
	"io"
)

type VariantSpec struct {
	Type  ResolutionType
	Width int
}

type GeneratedVariant struct {
	Type      ResolutionType
	Data      []byte
	SizeBytes int64
	MIMEType  MIME
}

type VariantCreator interface {
	GenerateVariants(data []byte, mime MIME, specs []VariantSpec) ([]GeneratedVariant, error)
}

type ImageTransformer interface {
	Resize(img image.Image, targetWidth int) image.Image

	Decode(r io.Reader, contentType string) (image.Image, error)

	Encode(img image.Image, contentType string) ([]byte, error)
}
