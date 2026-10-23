package wishlist_test

import (
	"context"
	"sync"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/wishlist"

	"github.com/google/uuid"
)

type mockWishlistRepo struct {
	mu                      sync.RWMutex
	items                   map[string]wishlist.WishlistItem
	addErr                  error
	removeErr               error
	listErr                 error
	existsErr               error
	simulateUniqueViolation bool
}

func newMockWishlistRepo() *mockWishlistRepo {
	return &mockWishlistRepo{
		items: make(map[string]wishlist.WishlistItem),
	}
}

func (m *mockWishlistRepo) key(customerID, productID uuid.UUID) string {
	return customerID.String() + ":" + productID.String()
}

func (m *mockWishlistRepo) Add(_ context.Context, _ transaction.Executor, item wishlist.WishlistItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.addErr != nil {
		return m.addErr
	}
	k := m.key(item.CustomerID, item.ProductID)
	if m.simulateUniqueViolation {
		if _, ok := m.items[k]; ok {
			return wishlist.ErrWishlistItemAlreadyExists
		}
	}
	m.items[k] = item
	return nil
}

func (m *mockWishlistRepo) Remove(_ context.Context, _ transaction.Executor, customerID, productID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.items, m.key(customerID, productID))
	return nil
}

func (m *mockWishlistRepo) ListByCustomerID(_ context.Context, _ transaction.Executor, customerID uuid.UUID) ([]wishlist.WishlistItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []wishlist.WishlistItem
	for _, item := range m.items {
		if item.CustomerID == customerID {
			res = append(res, item)
		}
	}
	return res, nil
}

func (m *mockWishlistRepo) Exists(_ context.Context, _ transaction.Executor, customerID, productID uuid.UUID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.existsErr != nil {
		return false, m.existsErr
	}
	_, ok := m.items[m.key(customerID, productID)]
	return ok, nil
}

type mockProductRepo struct {
	products map[uuid.UUID]*product.Product
	getErr   error
	findErr  error
}

func (m *mockProductRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*product.Product, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.products[id], nil
}

func (m *mockProductRepo) FindByIDs(_ context.Context, _ transaction.Executor, ids []uuid.UUID) ([]product.Product, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var res []product.Product
	for _, id := range ids {
		if p, ok := m.products[id]; ok {
			res = append(res, *p)
		}
	}
	return res, nil
}

type mockInventoryRepo struct {
	inventories map[uuid.UUID][]inventory.Inventory
}

func (m *mockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, ids []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error) {
	return m.inventories, nil
}
