package imageutil

type (
	MIME           string
	ResolutionType string
)

const (
	MIMEJPEG MIME = "image/jpeg"
	MIMEPNG  MIME = "image/png"
	MIMEWEBP MIME = "image/webp"
)

func (m MIME) IsAllowed() bool {
	switch m {
	case MIMEJPEG, MIMEPNG, MIMEWEBP:
		return true
	default:
		return false
	}
}

func (m MIME) Extension() (string, error) {
	switch m {
	case MIMEJPEG:
		return "jpg", nil
	case MIMEPNG:
		return "png", nil
	case MIMEWEBP:
		return "webp", nil
	default:
		return "", ErrInvalidImageMime
	}
}
