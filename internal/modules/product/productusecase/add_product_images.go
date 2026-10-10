package productusecase

import (
	"bytes"
	"context"
	"fmt"

	"komecore/internal/apperror"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	appclock "komecore/pkg/clock"
	image "komecore/pkg/imageutil"
	slug "komecore/pkg/slug"

	"github.com/google/uuid"
)

var specs = []image.VariantSpec{
	{Type: productdomain.ResolutionThumbnail, Width: 150},
	{Type: productdomain.ResolutionPreview, Width: 600},
	{Type: productdomain.ResolutionDetail, Width: 1200},
}

type AddProductImagesUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	productRepo    productrepo.ProductRepository
	productImgRepo productrepo.ProductImageRepository
	slugGen        slug.Generator
	resolutionGen  image.VariantCreator
	fileStore      storage.Provider
}

func NewAddProductImagesUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	productRepo productrepo.ProductRepository,
	productImgRepo productrepo.ProductImageRepository,
	slugGen slug.Generator,
	resolutionGen image.VariantCreator,
	fileStore storage.Provider,
) *AddProductImagesUsecase {
	return &AddProductImagesUsecase{
		executor:       executor,
		transactor:     transactor,
		productRepo:    productRepo,
		productImgRepo: productImgRepo,
		slugGen:        slugGen,
		resolutionGen:  resolutionGen,
		fileStore:      fileStore,
	}
}

type ProductImageInput struct {
	Data         []byte
	OriginalName string
	MIMEType     string
	SizeBytes    int64
	IsPrimary    bool
	DisplayOrder int
}

type AddProductImageInput struct {
	ProductID uuid.UUID
	Images    []ProductImageInput
}

type productToVariants struct {
	index       int
	variantType image.ResolutionType
}

func (u *AddProductImagesUsecase) Execute(
	ctx context.Context,
	input AddProductImageInput,
) error {
	product, err := u.productRepo.GetByID(ctx, u.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	if product == nil {
		return apperror.NewNotFound("product not found")
	}

	var (
		// Input payloads prepared for storage
		// upload operations.
		uploadImages []storage.UploadInput

		// Domain-level product image entities
		// before persistence.
		productImages []productdomain.ProductImage

		// Index mapping between uploaded results
		// and product images.Used to re-associate
		// generated variant outputs with their
		// source image.
		mappings []productToVariants

		now = appclock.Now()
	)

	for _, img := range input.Images {
		productImage := productdomain.ProductImage{
			ID:           uuid.New(),
			ProductID:    product.ID,
			Variants:     make(map[image.ResolutionType]productdomain.ImageVariant),
			IsPrimary:    img.IsPrimary,
			DisplayOrder: img.DisplayOrder,
			Metadata: productdomain.ProductImageMetadata{
				OriginalName: img.OriginalName,
				MIMEType:     image.MIME(img.MIMEType),
				SizeBytes:    img.SizeBytes,
			},
			CreatedAt: now,
		}

		// UploadMany returns responses in the same order as uploadImages,
		// but each ObjectResponse only contains the stored result data
		// (such as the Key) and does not include domain context such as
		// the variant type or the ProductImage it belongs to.
		//
		// It is necessary to capture the index of the incoming ProductImage
		// before append-ing it so that the location of each upload response
		// can be reconstructed.
		productImageIndex := len(productImages)
		productImages = append(productImages, productImage)

		variants, err := u.resolutionGen.GenerateVariants(
			img.Data,
			image.MIME(img.MIMEType),
			specs,
		)
		if err != nil {
			return fmt.Errorf("failed to create variants of %s: %w", img.OriginalName, err)
		}

		for _, variant := range variants {
			key, err := productImage.BuildObjectKey()
			if err != nil {
				return fmt.Errorf("failed to build %s key: %w", variant.Type, err)
			}

			uploadImages = append(uploadImages, storage.UploadInput{
				Bucket:      "public-assets",
				Key:         key,
				File:        bytes.NewReader(variant.Data),
				ContentType: string(variant.MIMEType),
			})

			// Record positional metadata for this upload.
			//
			//   - uploadImages[n] <-> mappings[n]
			//
			// This makes it possible to restore the domain context after
			// UploadMany returns a value, because ObjectResponse doesn't
			// tell which variant/image it has.
			mappings = append(mappings, productToVariants{
				index:       productImageIndex,
				variantType: variant.Type,
			})
		}
	}

	responses, err := u.fileStore.UploadMany(uploadImages)
	if err != nil {
		return fmt.Errorf("failed to upload images: %w", err)
	}

	// Responses maintain the order of UploadMany input.
	// So, responses[i] corresponds to uploadImages[i], and mappings[i]
	// tells exactly which ProductImage and variant slot should receive
	// the uploaded key.
	for i, resp := range responses {
		m := mappings[i]

		productImages[m.index].Variants[m.variantType] =
			productdomain.ImageVariant{
				Type: m.variantType,
				Key:  resp.Key,
			}
	}

	err = u.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := u.productImgRepo.Create(ctx, exec, productImages); err != nil {
				return fmt.Errorf("failed to save product image: %w", err)
			}

			return nil
		},
	)

	return nil
}
