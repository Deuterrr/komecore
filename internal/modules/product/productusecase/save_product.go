package productusecase

import (
	"context"
	"errors"
	"fmt"

	"komecore/internal/apperror"
	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product"
	appclock "komecore/pkg/clock"
	"komecore/pkg/slug"

	"github.com/google/uuid"
)

type SaveProductUsecase struct {
	transactor  transaction.Transactor
	productRepo product.Repository
	slugGen     slug.Generator
	perfRepo    product.PerformanceRepository
	cache       cache.Cache
}

func NewSaveProductUsecase(
	transactor transaction.Transactor,
	productRepo product.Repository,
	slugGen slug.Generator,
	perfRepo product.PerformanceRepository,
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

	prodStatus := product.ProductStatus(input.Status)
	if prodStatus == "" {
		prodStatus = product.ProductStatusActive
	}

	prod := &product.Product{
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
	if err := prod.Validate(); err != nil {
		if errors.Is(err, product.ErrInvalidProductName) ||
			errors.Is(err, product.ErrInvalidProductPrice) {
			return apperror.NewInvalidInput(err.Error())
		}

		return err
	}

	perf := product.ProductPerformance{
		ProductID:            productID,
		CostPrice:            input.CostPrice,
		SupplierLeadTimeDays: input.SupplierLeadTimeDays,
	}

	if err := u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.productRepo.Save(ctx, exec,
			prod,
		); err != nil {
			return fmt.Errorf("failed to save product: %w", err)
		}

		if err := u.perfRepo.UpsertPerformance(ctx, exec,
			perf,
			prod.Price,
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
