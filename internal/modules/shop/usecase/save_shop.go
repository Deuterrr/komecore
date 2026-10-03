package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	authorDomain "komecore/internal/modules/auth/domain"
	"komecore/internal/modules/shop/domain"
	"komecore/internal/modules/shop/repository"
	appclock "komecore/pkg/clock"
	slug "komecore/pkg/slug"

	"github.com/google/uuid"
)

type SaveShopUsecase struct {
	shopRepo repository.ShopRepository
	slugGen  slug.Generator
	executor transaction.Executor
}

func NewSaveShopUsecase(
	shopRepo repository.ShopRepository,
	slugGen slug.Generator,
	executor transaction.Executor,
) *SaveShopUsecase {
	return &SaveShopUsecase{
		shopRepo: shopRepo,
		slugGen:  slugGen,
		executor: executor,
	}
}

type SaveShopInput struct {
	ID             *uuid.UUID
	Name           string
	Description    *string
	IsActive       *bool
	ApprovalStatus *string
}

func (u *SaveShopUsecase) Execute(
	ctx context.Context,
	actor authorDomain.Actor,
	input SaveShopInput,
) error {
	isAdmin := false
	for _, actorRole := range actor.Roles {
		if actorRole.Code == authorDomain.RoleStaffAdmin {
			isAdmin = true
			break
		}
	}

	var shop domain.Shop
	now := appclock.Now()

	if input.ID == nil {
		// Creating new shop
		shopID := uuid.New()
		shop = domain.Shop{
			ID:          shopID,
			Name:        input.Name,
			Slug:        u.slugGen.Generate(input.Name),
			Description: input.Description,
			CreatedAt:   now,
		}

		if isAdmin {
			if input.IsActive != nil {
				shop.IsActive = *input.IsActive
			}
			if input.ApprovalStatus != nil && *input.ApprovalStatus != "" {
				shop.ApprovalStatus = domain.ShopApprovalStatus(*input.ApprovalStatus)
			} else {
				shop.ApprovalStatus = domain.ShopApprovalStatusPending
			}
		} else {
			// Regular staff creates shop in pending approval & inactive state
			shop.IsActive = false
			shop.ApprovalStatus = domain.ShopApprovalStatusPending
		}
	} else {
		// Updating existing shop
		existing, err := u.shopRepo.GetByID(ctx, u.executor, *input.ID)
		if err != nil {
			return fmt.Errorf("failed to retrieve existing shop: %w", err)
		}
		if existing == nil {
			return apperrors.NewNotFound("shop not found")
		}

		shop = *existing
		shop.Name = input.Name
		shop.Slug = u.slugGen.Generate(input.Name)
		shop.Description = input.Description
		shop.UpdatedAt = &now

		if isAdmin {
			if input.IsActive != nil {
				shop.IsActive = *input.IsActive
			}
			if input.ApprovalStatus != nil && *input.ApprovalStatus != "" {
				shop.ApprovalStatus = domain.ShopApprovalStatus(*input.ApprovalStatus)
			}
		}
	}

	err := u.shopRepo.Save(ctx, u.executor, shop)
	if err != nil {
		return fmt.Errorf("failed to save shop: %w", err)
	}

	return nil
}
