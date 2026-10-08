package productdomain

import (
	"fmt"
	"path"
	"strings"
	"time"

	image "komecore/pkg/imageutil"

	"github.com/google/uuid"
)

const (
	DirPrefix   = "products"
	KeyPattern  = "products/{product_id}/{image_id}.{ext}"
	MaxFileSize = 5 << 20
	MaxFileMB   = MaxFileSize / (1 << 20)
)

const (
	ResolutionThumbnail image.ResolutionType = "thumbnail"
	ResolutionPreview   image.ResolutionType = "preview"
	ResolutionDetail    image.ResolutionType = "detail"
)

type ImageVariant struct {
	Type image.ResolutionType
	Key  string
}

type ProductImage struct {
	ID        uuid.UUID
	ProductID uuid.UUID

	Variants map[image.ResolutionType]ImageVariant

	IsPrimary    bool
	DisplayOrder int

	Metadata ProductImageMetadata

	CreatedAt time.Time
}

type ProductImageMetadata struct {
	OriginalName string
	MIMEType     image.MIME
	SizeBytes    int64
}

func IsValidProductImageFileName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}

	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}

	return path.Base(name) == name
}

func (pI *ProductImage) BuildObjectKey() (string, error) {
	ext, err := pI.Metadata.MIMEType.Extension()
	if err != nil {
		return "", err
	}

	return path.Join(
		DirPrefix,
		pI.ProductID.String(),
		fmt.Sprintf("%s.%s",
			pI.ID.String(),
			ext,
		),
	), nil
}
