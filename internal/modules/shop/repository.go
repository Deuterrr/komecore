package shop

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	// Orders shops by descending registration timestamp (default directory order).
	ShopSortLatest pagination.SortKey = "latest"
	// Orders shops alphabetically by business name.
	ShopSortName pagination.SortKey = "name"
	// Orders shops by operational activation status.
	ShopSortActive pagination.SortKey = "active"
	// Orders shops by last modification timestamp.
	ShopSortModify pagination.SortKey = "modify"
)

// FindShopsParams encapsulates search filters, approval statuses, and pagination for shop queries.
type FindShopsParams struct {
	ID             *string
	ShopIDs        []uuid.UUID
	Name           *string
	Slug           *string
	IsActive       *bool
	ApprovalStatus *string

	Pagination pagination.Pagination
	Sorts      pagination.Sorts
}

// Repository defines persistence operations for merchant shops, seller profiles,
// and storefront tenant lifecycle within transaction boundaries.
type Repository interface {
	// Queries paginated merchant shops filtered by approval status, active flag, slug, or name.
	FindByParams(ctx context.Context, exec transaction.Executor, params FindShopsParams) ([]Shop, int, error)
	// Batch-loads shop records across multiple IDs to prevent N+1 database queries during basket pricing.
	FindByIDs(ctx context.Context, exec transaction.Executor, IDs []uuid.UUID) ([]Shop, error)

	// Resolves a merchant shop profile by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Shop, error)
	// Looks up an active merchant shop by unique storefront URL slug.
	GetBySlug(ctx context.Context, exec transaction.Executor, slug string) (*Shop, error)

	// Persists a merchant shop, inserting a new profile or updating business name, description, and status.
	Save(ctx context.Context, exec transaction.Executor, shop Shop) error

	// Soft-deletes a merchant shop profile, deactivating seller operations while preserving order history.
	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
}
