package usecase

import (
	"context"
	"errors"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"
	appclock "komecore/pkg/clock"
	"komecore/pkg/slug"

	"github.com/google/uuid"
)

type SaveProductUsecase struct {
	transactor  transaction.Transactor
	productRepo repository.ProductRepository
	slugGen     slug.Generator
	perfRepo    repository.ProductPerformanceRepository
}

func NewSaveProductUsecase(
	transactor transaction.Transactor,
	productRepo repository.ProductRepository,
	slugGen slug.Generator,
	perfRepo repository.ProductPerformanceRepository,
) *SaveProductUsecase {
	return &SaveProductUsecase{
		transactor:  transactor,
		productRepo: productRepo,
		slugGen:     slugGen,
		perfRepo:    perfRepo,
	}
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

	prodStatus := domain.ProductStatus(input.Status)
	if prodStatus == "" {
		prodStatus = domain.ProductStatusActive
	}

	product := &domain.Product{
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
		if errors.Is(err, domain.ErrInvalidProductName) ||
			errors.Is(err, domain.ErrInvalidProductPrice) {
			return apperrors.NewInvalidInput(err.Error())
		}

		return err
	}

	perf := domain.ProductPerformance{
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

	return nil
}
