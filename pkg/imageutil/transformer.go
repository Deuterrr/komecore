package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type imageTransformer struct{}

func NewImageTransformer() ImageTransformer {
	return &imageTransformer{}
}

func (iT *imageTransformer) Decode(r io.Reader, contentType string) (image.Image, error) {
	// Attempt to decode using standard image decoder which handles registered formats
	// Since we import _ "golang.org/x/image/webp", it should support webp decoding.
	// image/jpeg and image/png are already imported.
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

func (iT *imageTransformer) Encode(img image.Image, contentType string) ([]byte, error) {
	var buf bytes.Buffer
	var err error

	switch contentType {
	case "image/png":
		err = png.Encode(&buf, img)
	case "image/jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	case "image/webp":
		// Standard lib doesn't have webp encoder, fallback to jpeg.
		// Alternatively, we could save as png. Using JPEG for size.
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	default:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}
	return buf.Bytes(), nil
}

func (iT *imageTransformer) Resize(img image.Image, targetWidth int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= targetWidth {
		return img
	}

	ratio := float64(targetWidth) / float64(width)
	targetHeight := int(float64(height) * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}
