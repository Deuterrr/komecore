package productpersistence

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	image "komecore/pkg/imageutil"

	"github.com/google/uuid"
)

type ProductImageRepository struct{}

func NewProductImageRepository() *ProductImageRepository {
	return &ProductImageRepository{}
}

func (r *ProductImageRepository) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]productdomain.ProductImage, error) {
	result := make(map[uuid.UUID][]productdomain.ProductImage)
	if len(productIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT
			id,
			product_id,
			thumbnail_url,
			preview_url,
			detail_url,
			thumbnail_key,
			preview_key,
			detail_key,
			is_primary,
			display_order,
			created_at
		FROM product_images
		WHERE product_id = ANY($1::uuid[]) AND deleted_at IS NULL
		ORDER BY display_order ASC
	`

	productIDStrings := make([]string, len(productIDs))
	for i, id := range productIDs {
		productIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, productIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query product images failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r productImageRow

		err := rows.Scan(
			&r.ID,
			&r.ProductID,
			&r.ThumbURL,
			&r.PreviewURL,
			&r.DetailURL,
			&r.ThumbKey,
			&r.PreviewKey,
			&r.DetailKey,
			&r.IsPrimary,
			&r.DisplayOrder,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}

		img := productdomain.ProductImage{
			ID:        r.ID,
			ProductID: r.ProductID,

			Variants: map[image.ResolutionType]productdomain.ImageVariant{
				productdomain.ResolutionThumbnail: {
					Type: productdomain.ResolutionThumbnail,
					Key:  r.ThumbKey,
				},
				productdomain.ResolutionPreview: {
					Type: productdomain.ResolutionPreview,
					Key:  r.PreviewKey,
				},
				productdomain.ResolutionDetail: {
					Type: productdomain.ResolutionDetail,
					Key:  r.DetailKey,
				},
			},

			IsPrimary:    r.IsPrimary,
			DisplayOrder: r.DisplayOrder,

			Metadata:  productdomain.ProductImageMetadata{},
			CreatedAt: r.CreatedAt,
		}

		result[r.ProductID] = append(result[r.ProductID], img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product image failed: %w", err)
	}

	return result, nil
}

func (r *ProductImageRepository) ListByProductID(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
) ([]productdomain.ProductImage, error) {
	query := `
		SELECT
			id,
			product_id,
			thumbnail_url,
			preview_url,
			detail_url,
			thumbnail_key,
			preview_key,
			detail_key,
			is_primary,
			display_order,
			created_at
		FROM product_images
		WHERE product_id = $1 AND deleted_at IS NULL
		ORDER BY display_order ASC
	`

	rows, err := exec.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("query product images failed: %w", err)
	}
	defer rows.Close()

	var rowsData []productImageRow
	for rows.Next() {
		var r productImageRow

		err := rows.Scan(
			&r.ID,
			&r.ProductID,
			&r.ThumbURL,
			&r.ThumbKey,
			&r.PreviewURL,
			&r.PreviewKey,
			&r.DetailURL,
			&r.DetailKey,
			&r.IsPrimary,
			&r.DisplayOrder,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}

		rowsData = append(rowsData, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product image failed: %w", err)
	}

	images := make([]productdomain.ProductImage, 0, len(rowsData))

	for _, r := range rowsData {
		img := productdomain.ProductImage{
			ID:        r.ID,
			ProductID: r.ProductID,

			Variants: map[image.ResolutionType]productdomain.ImageVariant{
				productdomain.ResolutionThumbnail: {
					Type: productdomain.ResolutionThumbnail,
					Key:  r.ThumbKey,
				},
				productdomain.ResolutionPreview: {
					Type: productdomain.ResolutionPreview,
					Key:  r.PreviewKey,
				},
				productdomain.ResolutionDetail: {
					Type: productdomain.ResolutionDetail,
					Key:  r.DetailKey,
				},
			},

			IsPrimary:    r.IsPrimary,
			DisplayOrder: r.DisplayOrder,

			Metadata:  productdomain.ProductImageMetadata{},
			CreatedAt: r.CreatedAt,
		}

		images = append(images, img)
	}

	return images, nil
}

func (r *ProductImageRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	images []productdomain.ProductImage,
) error {
	query := `
		INSERT INTO product_images (
			id,
			product_id,
			thumbnail_url,
			preview_url,
			detail_url,
			thumbnail_key,
			preview_key,
			detail_key,
			is_primary,
			display_order,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	for _, image := range images {
		_, err := exec.Exec(ctx, query,
			image.ID,
			image.ProductID,
			image.Variants[productdomain.ResolutionThumbnail].Key,
			image.Variants[productdomain.ResolutionPreview].Key,
			image.Variants[productdomain.ResolutionDetail].Key,
			image.Variants[productdomain.ResolutionThumbnail].Key,
			image.Variants[productdomain.ResolutionPreview].Key,
			image.Variants[productdomain.ResolutionDetail].Key,
			image.IsPrimary,
			image.DisplayOrder,
			image.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert product image failed: %w", err)
		}
	}

	return nil
}

func (r *ProductImageRepository) SoftDeleteByProductID(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
) error {
	query := `
		UPDATE
			product_images
		SET
			deleted_at = NOW()
		WHERE
			product_id = $1
	`

	_, err := exec.Exec(ctx, query, productID)
	if err != nil {
		return fmt.Errorf("delete product images failed: %w", err)
	}

	return nil
}
