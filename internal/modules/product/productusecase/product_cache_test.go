package productusecase_test

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/modules/product/productusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCacheProductRepo struct {
	productrepo.ProductRepository
	product         *productdomain.Product
	getBySlugCalls  int
	getByIDCalls    int
	deleteCalls     int
	lastDeletedUUID uuid.UUID
}

func (m *mockCacheProductRepo) GetBySlug(ctx context.Context, exec transaction.Executor, slug string) (*productdomain.Product, error) {
	m.getBySlugCalls++
	return m.product, nil
}

func (m *mockCacheProductRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*productdomain.Product, error) {
	m.getByIDCalls++
	return m.product, nil
}

func (m *mockCacheProductRepo) Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error {
	m.deleteCalls++
	m.lastDeletedUUID = id
	return nil
}

func TestProductCacheAside_GetProduct(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache()

	testProdID := uuid.New()
	prod := &productdomain.Product{
		ID:    testProdID,
		Name:  "Mechanical Keyboard",
		Slug:  "mechanical-keyboard",
		Price: 1200000,
	}

	repo := &mockCacheProductRepo{
		product: prod,
	}

	uc := productusecase.NewGetProductUsecase(
		nil,
		nil,
		repo,
		nil,
		nil,
		nil,
		nil,
	).WithCache(c)

	// Call 1: Cache Miss -> Fetches from repo and populates cache
	res1, err := uc.Execute(ctx, "mechanical-keyboard")
	require.NoError(t, err)
	require.NotNil(t, res1)
	assert.Equal(t, prod.Name, res1.Product.Name)
	assert.Equal(t, 1, repo.getBySlugCalls, "repo must be called on cache miss")

	// Call 2: Cache Hit -> Served directly from MemoryCache, no additional repo call
	res2, err := uc.Execute(ctx, "mechanical-keyboard")
	require.NoError(t, err)
	require.NotNil(t, res2)
	assert.Equal(t, prod.Name, res2.Product.Name)
	assert.Equal(t, 1, repo.getBySlugCalls, "repo must NOT be called on cache hit")
}

func TestProductCacheInvalidation_DeleteProduct(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache()

	// Seed cache with product keys
	_ = c.Set(ctx, "cache:product:slug:test-item", "cached-detail", 15*time.Minute)
	_ = c.Set(ctx, "cache:products:list:all", "cached-list", 5*time.Minute)
	_ = c.Set(ctx, "cache:user:profile:1", "user-data", 10*time.Minute)

	testID := uuid.New()
	prod := &productdomain.Product{ID: testID, Name: "Test", Slug: "test-item"}

	repo := &mockCacheProductRepo{
		product: prod,
	}

	deleteUc := productusecase.NewDeleteProductUsecase(repo, nil).WithCache(c)

	err := deleteUc.Execute(ctx, testID)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.deleteCalls)
	assert.Equal(t, testID, repo.lastDeletedUUID)

	// Confirm product caches were purged while unrelated caches remain intact
	var dest string
	assert.ErrorIs(t, c.Get(ctx, "cache:product:slug:test-item", &dest), cache.ErrCacheMiss)
	assert.ErrorIs(t, c.Get(ctx, "cache:products:list:all", &dest), cache.ErrCacheMiss)
	assert.NoError(t, c.Get(ctx, "cache:user:profile:1", &dest))
}
