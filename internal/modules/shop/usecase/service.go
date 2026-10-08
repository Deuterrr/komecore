package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/common/authctx"
	transaction "komecore/internal/infra/transactor"
	addressDomain "komecore/internal/modules/address/domain"
	addressRepo "komecore/internal/modules/address/repository"
	"komecore/internal/modules/shop/domain"
	"komecore/internal/modules/shop/repository"
	query "komecore/internal/shared/query"
	appclock "komecore/pkg/clock"
	slug "komecore/pkg/slug"

	"github.com/google/uuid"
)

// ShopProductInfo contains catalog product attributes needed by shop consumers.
type ShopProductInfo struct {
	ID          uuid.UUID
	SKU         string
	Name        string
	Slug        string
	Description *string
	Status      string
	Price       int64
	Weight      *float64
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// ShopProductInventoryInfo contains stock quantities for a shop product.
type ShopProductInventoryInfo struct {
	TotalStock    int
	ReservedStock int
}

func (s ShopProductInventoryInfo) Available() int {
	return s.TotalStock - s.ReservedStock
}

// ShopProductResult pairs a product with its inventory record for a given shop.
type ShopProductResult struct {
	Product   ShopProductInfo
	Inventory ShopProductInventoryInfo
}

// ShopProductProvider decouples the shop module from direct product and inventory repository imports.
type ShopProductProvider interface {
	GetShopProducts(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]ShopProductResult, error)
}

type FindShopsInput struct {
	Page           int
	Limit          int
	ID             *string
	ShopIDs        []uuid.UUID
	Name           *string
	Slug           *string
	IsActive       *bool
	ApprovalStatus *string
	Sort           string
}

type SaveShopInput struct {
	ID             *uuid.UUID
	Name           string
	Description    *string
	IsActive       *bool
	ApprovalStatus *string
}

type ShopService struct {
	shopRepo    repository.ShopRepository
	addressRepo addressRepo.ShopAddressRepository
	provider    ShopProductProvider
	slugGen     slug.Generator
	executor    transaction.Executor
}

func NewShopService(
	shopRepo repository.ShopRepository,
	addressRepo addressRepo.ShopAddressRepository,
	provider ShopProductProvider,
	slugGen slug.Generator,
	executor transaction.Executor,
) *ShopService {
	return &ShopService{
		shopRepo:    shopRepo,
		addressRepo: addressRepo,
		provider:    provider,
		slugGen:     slugGen,
		executor:    executor,
	}
}

// FindShops queries shops with filtering, sorting, and pagination.
func (s *ShopService) FindShops(ctx context.Context, input FindShopsInput) ([]domain.Shop, int, error) {
	var shopSortKeys = map[string]query.SortKey{
		"name":     repository.ShopSortName,
		"active":   repository.ShopSortActive,
		"date":     repository.ShopSortLatest,
		"modified": repository.ShopSortModify,
	}

	var sorts query.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir query.SortDirection = query.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = query.SortAsc
				}
			}

			sortKey, exists := shopSortKeys[key]
			if exists {
				sorts = append(sorts, query.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = query.Sorts{
			{
				By:        repository.ShopSortLatest,
				Direction: query.SortDesc,
			},
		}
	}

	approvalStatus := input.ApprovalStatus
	if approvalStatus == nil && input.IsActive != nil && *input.IsActive {
		approvedStr := string(domain.ShopApprovalStatusApproved)
		approvalStatus = &approvedStr
	}

	params := repository.FindShopsParams{
		ID:             input.ID,
		ShopIDs:        input.ShopIDs,
		Name:           input.Name,
		Slug:           input.Slug,
		IsActive:       input.IsActive,
		ApprovalStatus: approvalStatus,
		Pagination: query.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	shops, total, err := s.shopRepo.FindByParams(ctx, s.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load shops: %w", err)
	}
	if len(shops) == 0 {
		return []domain.Shop{}, 0, nil
	}

	return shops, total, nil
}

// GetByID finds a shop by its unique ID.
func (s *ShopService) GetByID(ctx context.Context, shopID uuid.UUID) (*domain.Shop, error) {
	shop, err := s.shopRepo.GetByID(ctx, s.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop: %w", err)
	}
	return shop, nil
}

// GetBySlug finds a shop by its unique slug.
func (s *ShopService) GetBySlug(ctx context.Context, slug string) (*domain.Shop, error) {
	shop, err := s.shopRepo.GetBySlug(ctx, s.executor, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop by slug: %w", err)
	}
	return shop, nil
}

// SaveShop creates or updates a shop record according to actor permissions.
func (s *ShopService) SaveShop(ctx context.Context, actor authctx.Actor, input SaveShopInput) error {
	isAdmin := false
	for _, actorRole := range actor.Roles {
		if actorRole.Code == authctx.RoleStaffAdmin {
			isAdmin = true
			break
		}
	}

	var shop domain.Shop
	now := appclock.Now()

	if input.ID == nil {
		shopID := uuid.New()
		shop = domain.Shop{
			ID:          shopID,
			Name:        input.Name,
			Slug:        s.slugGen.Generate(input.Name),
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
			shop.IsActive = false
			shop.ApprovalStatus = domain.ShopApprovalStatusPending
		}
	} else {
		existing, err := s.shopRepo.GetByID(ctx, s.executor, *input.ID)
		if err != nil {
			return fmt.Errorf("failed to retrieve existing shop: %w", err)
		}
		if existing == nil {
			return apperrors.NewNotFound("shop not found")
		}

		shop = *existing
		shop.Name = input.Name
		shop.Slug = s.slugGen.Generate(input.Name)
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

	err := s.shopRepo.Save(ctx, s.executor, shop)
	if err != nil {
		return fmt.Errorf("failed to save shop: %w", err)
	}

	return nil
}

// DeleteShop deletes a shop, requiring staff admin authorization.
func (s *ShopService) DeleteShop(ctx context.Context, actor authctx.Actor, shopID uuid.UUID) error {
	isAdmin := false
	for _, role := range actor.Roles {
		if role.Code == authctx.RoleStaffAdmin {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		return apperrors.NewForbidden("insufficient permissions to delete shop")
	}

	shop, err := s.shopRepo.GetByID(ctx, s.executor, shopID)
	if err != nil {
		return fmt.Errorf("failed to retrieve shop: %w", err)
	}
	if shop == nil {
		return apperrors.NewNotFound("shop not found")
	}

	if err := s.shopRepo.Delete(ctx, s.executor, shop.ID); err != nil {
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	return nil
}

// GetShopAddresses retrieves the address list for a shop.
func (s *ShopService) GetShopAddresses(ctx context.Context, shopID uuid.UUID) ([]addressDomain.ShopAddress, error) {
	if s.addressRepo == nil {
		return nil, fmt.Errorf("shop address repository not initialized")
	}
	addresses, err := s.addressRepo.FindByShopID(ctx, s.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop addresses: %w", err)
	}
	return addresses, nil
}

// GetShopProducts retrieves the product catalog and inventories for a shop.
func (s *ShopService) GetShopProducts(ctx context.Context, shopID uuid.UUID) ([]ShopProductResult, error) {
	if s.provider == nil {
		return []ShopProductResult{}, nil
	}
	return s.provider.GetShopProducts(ctx, s.executor, shopID)
}
