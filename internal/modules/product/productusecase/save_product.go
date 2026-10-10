package productusecase

import (
	"context"
	"errors"
	"fmt"

	"komecore/internal/apperror"
	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	appclock "komecore/pkg/clock"
	"komecore/pkg/slug"

	"github.com/google/uuid"
)

type SaveProductUsecase struct {
	transactor  transaction.Transactor
	productRepo productrepo.ProductRepository
	slugGen     slug.Generator
	perfRepo    productrepo.ProductPerformanceRepository
	cache       cache.Cache
}

func NewSaveProductUsecase(
	transactor transaction.Transactor,
	productRepo productrepo.ProductRepository,
	slugGen slug.Generator,
	perfRepo productrepo.ProductPerformanceRepository,
) *SaveProductUsecase {
	return &SaveProductUsecase{
		transactor:  transactor,
		productRepo: productRepo,
		slugGen:     slugGen,
		perfRepo:    perfRepo,
	}
}

func (u *SaveProductUsecase) WithCache(c cache.Cache) *SaveProductUsecase {
	u.cache = c
	return u
}

type SaveProductInput struct {
	ID                   *uuid.UUID
	SKU                  string
	Name                 string
	Description          *string
	Status               string
	Price                int64
	Weight               *float64
	CostPrice            *int64
	SupplierLeadTimeDays *int
}

func (u *SaveProductUsecase) Execute(
	ctx context.Context,
	input SaveProductInput,
) error {
	now := appclock.Now()

	var productID uuid.UUID
	if input.ID == nil {
		productID = uuid.New()
	} else {
		productID = *input.ID
	}

	prodStatus := productdomain.ProductStatus(input.Status)
	if prodStatus == "" {
		prodStatus = productdomain.ProductStatusActive
	}

	product := &productdomain.Product{
		ID:          productID,
		SKU:         input.SKU,
		Name:        input.Name,
		Slug:        u.slugGen.Generate(input.Name),
		Description: input.Description,
		Status:      prodStatus,
		Price:       input.Price,
		Weight:      input.Weight,
		CreatedAt:   now,
	}
	if err := product.Validate(); err != nil {
		if errors.Is(err, productdomain.ErrInvalidProductName) ||
			errors.Is(err, productdomain.ErrInvalidProductPrice) {
			return apperror.NewInvalidInput(err.Error())
		}

		return err
	}

	perf := productdomain.ProductPerformance{
		ProductID:            productID,
		CostPrice:            input.CostPrice,
		SupplierLeadTimeDays: input.SupplierLeadTimeDays,
	}

	if err := u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.productRepo.Save(ctx, exec,
			product,
		); err != nil {
			return fmt.Errorf("failed to save product: %w", err)
		}

		if err := u.perfRepo.UpsertPerformance(ctx, exec,
			perf,
			product.Price,
		); err != nil {
			return fmt.Errorf("failed to save product performance: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	if u.cache != nil {
		_ = u.cache.DeletePattern(ctx, "cache:product*")
	}

	return nil
}
