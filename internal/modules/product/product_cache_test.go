package product_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCacheProductRepo struct {
	product.Repository
	product         *product.Product
	getBySlugCalls  int
	getByIDCalls    int
	deleteCalls     int
	lastDeletedUUID uuid.UUID
}

func (m *mockCacheProductRepo) GetBySlug(ctx context.Context, exec transaction.Executor, slug string) (*product.Product, error) {
	m.getBySlugCalls++
	return m.product, nil
}

func (m *mockCacheProductRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*product.Product, error) {
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
	prod := &product.Product{
		ID:    testProdID,
		Name:  "Mechanical Keyboard",
		Slug:  "mechanical-keyboard",
		Price: 1200000,
	}

	repo := &mockCacheProductRepo{
		product: prod,
	}

	svc := product.NewService(
		nil,
		nil,
		repo,
		&mockStampedeInventoryRepo{},
		&mockStampedeImgRepo{},
		nil,
		&mockStampedeShopRepo{},
		nil,
		nil,
		nil,
	).WithCache(c)

	// Call 1: Cache Miss -> Fetches from repo and populates cache
	res1, err := svc.GetProduct(ctx, "mechanical-keyboard")
	require.NoError(t, err)
	require.NotNil(t, res1)
	assert.Equal(t, prod.Name, res1.Product.Name)
	assert.Equal(t, 1, repo.getBySlugCalls, "repo must be called on cache miss")

	// Call 2: Cache Hit -> Served directly from MemoryCache, no additional repo call
	res2, err := svc.GetProduct(ctx, "mechanical-keyboard")
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
	prod := &product.Product{ID: testID, Name: "Test", Slug: "test-item"}

	repo := &mockCacheProductRepo{
		product: prod,
	}

	svc := product.NewService(nil, nil, repo, nil, nil, nil, nil, nil, nil, nil).WithCache(c)

	err := svc.DeleteProduct(ctx, testID)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.deleteCalls)
	assert.Equal(t, testID, repo.lastDeletedUUID)

	// Confirm product caches were purged while unrelated caches remain intact
	var dest string
	assert.ErrorIs(t, c.Get(ctx, "cache:product:slug:test-item", &dest), cache.ErrCacheMiss)
	assert.ErrorIs(t, c.Get(ctx, "cache:products:list:all", &dest), cache.ErrCacheMiss)
	assert.NoError(t, c.Get(ctx, "cache:user:profile:1", &dest))
}

type mockStampedeProductRepo struct {
	product.Repository
	product           *product.Product
	getBySlugCalls    atomic.Int64
	findProductsCalls atomic.Int64
	queryDelay        time.Duration
}

func (m *mockStampedeProductRepo) GetBySlug(ctx context.Context, exec transaction.Executor, slug string) (*product.Product, error) {
	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}
	m.getBySlugCalls.Add(1)
	return m.product, nil
}

func (m *mockStampedeProductRepo) FindProductsWithInventory(ctx context.Context, exec transaction.Executor, params product.FindProductParams) ([]product.ProductWithInventory, int, error) {
	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}
	m.findProductsCalls.Add(1)
	if m.product != nil {
		return []product.ProductWithInventory{
			{
				Product: *m.product,
			},
		}, 1, nil
	}
	return []product.ProductWithInventory{}, 0, nil
}

type mockStampedeInventoryRepo struct {
	inventory.Repository
}

func (m *mockStampedeInventoryRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return []inventory.Inventory{}, nil
}

func (m *mockStampedeInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error) {
	return map[uuid.UUID][]inventory.Inventory{}, nil
}

type mockStampedeImgRepo struct {
	product.ImageRepository
}

func (m *mockStampedeImgRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]product.ProductImage, error) {
	return []product.ProductImage{}, nil
}

func (m *mockStampedeImgRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]product.ProductImage, error) {
	return map[uuid.UUID][]product.ProductImage{}, nil
}

type mockStampedeShopRepo struct {
	shop.Repository
}

func (m *mockStampedeShopRepo) FindByIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shop.Shop, error) {
	return []shop.Shop{}, nil
}

func TestProductSingleflight_GetProduct_CoalescesConcurrentMisses(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache() // starts cold/empty

	testProdID := uuid.New()
	prod := &product.Product{
		ID:    testProdID,
		Name:  "Mechanical Keyboard",
		Slug:  "mechanical-keyboard",
		Price: 1200000,
	}

	repo := &mockStampedeProductRepo{
		product:    prod,
		queryDelay: 10 * time.Millisecond, // simulate database query latency
	}

	svc := product.NewService(
		nil,
		nil,
		repo,
		&mockStampedeInventoryRepo{},
		&mockStampedeImgRepo{},
		nil,
		&mockStampedeShopRepo{},
		nil,
		nil,
		nil,
	).WithCache(c)

	const concurrency = 200
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			res, err := svc.GetProduct(ctx, "mechanical-keyboard")
			if err != nil {
				errCh <- fmt.Errorf("worker %d failed: %w", workerID, err)
				return
			}
			if res == nil || res.Product.Name != prod.Name {
				errCh <- fmt.Errorf("worker %d got invalid result: %v", workerID, res)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent GetProduct failed: %v", err)
	}

	// Invariant: Under 200 simultaneous cache misses, singleflight must coalesce
	// calls such that exactly 1 database repository query is executed.
	calls := repo.getBySlugCalls.Load()
	assert.Equal(t, int64(1), calls, "expected exactly 1 repository call for 200 concurrent cache misses")
}

func TestProductSingleflight_FindProducts_CoalescesConcurrentSearch(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache()

	testProdID := uuid.New()
	prod := &product.Product{
		ID:    testProdID,
		Name:  "Wireless Mouse",
		Slug:  "wireless-mouse",
		Price: 350000,
	}

	repo := &mockStampedeProductRepo{
		product:    prod,
		queryDelay: 10 * time.Millisecond,
	}

	svc := product.NewService(
		nil,
		nil,
		repo,
		&mockStampedeInventoryRepo{},
		&mockStampedeImgRepo{},
		nil,
		&mockStampedeShopRepo{},
		nil,
		nil,
		nil,
	).WithCache(c)

	const concurrency = 50
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	searchQuery := "wireless"
	input := product.FindProductsInput{
		SearchQuery: &searchQuery,
		Page:        1,
		Limit:       20,
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			results, total, err := svc.FindProducts(ctx, input)
			if err != nil {
				errCh <- fmt.Errorf("worker %d failed: %w", workerID, err)
				return
			}
			if total != 1 || len(results) != 1 || results[0].Product.Name != prod.Name {
				errCh <- fmt.Errorf("worker %d got unexpected results: total=%d, len=%d", workerID, total, len(results))
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent FindProducts failed: %v", err)
	}

	calls := repo.findProductsCalls.Load()
	assert.Equal(t, int64(1), calls, "expected exactly 1 repository call for 50 concurrent searches")
}
