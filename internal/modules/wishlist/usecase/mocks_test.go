package usecase_test

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/wishlist/domain"

	"github.com/google/uuid"
)

type mockWishlistRepo struct {
	items     map[string]domain.WishlistItem
	addErr    error
	removeErr error
	listErr   error
	existsErr error
}

func newMockWishlistRepo() *mockWishlistRepo {
	return &mockWishlistRepo{
		items: make(map[string]domain.WishlistItem),
	}
}

func (m *mockWishlistRepo) key(customerID, productID uuid.UUID) string {
	return customerID.String() + ":" + productID.String()
}

func (m *mockWishlistRepo) Add(_ context.Context, _ transaction.Executor, item domain.WishlistItem) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.items[m.key(item.CustomerID, item.ProductID)] = item
	return nil
}

func (m *mockWishlistRepo) Remove(_ context.Context, _ transaction.Executor, customerID, productID uuid.UUID) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.items, m.key(customerID, productID))
	return nil
}

func (m *mockWishlistRepo) ListByCustomerID(_ context.Context, _ transaction.Executor, customerID uuid.UUID) ([]domain.WishlistItem, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []domain.WishlistItem
	for _, item := range m.items {
		if item.CustomerID == customerID {
			res = append(res, item)
		}
	}
	return res, nil
}

func (m *mockWishlistRepo) Exists(_ context.Context, _ transaction.Executor, customerID, productID uuid.UUID) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	_, ok := m.items[m.key(customerID, productID)]
	return ok, nil
}

type mockProductRepo struct {
	products map[uuid.UUID]*productDomain.Product
	getErr   error
	findErr  error
}

func (m *mockProductRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*productDomain.Product, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.products[id], nil
}

func (m *mockProductRepo) FindByIDs(_ context.Context, _ transaction.Executor, ids []uuid.UUID) ([]productDomain.Product, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var res []productDomain.Product
	for _, id := range ids {
		if p, ok := m.products[id]; ok {
			res = append(res, *p)
		}
	}
	return res, nil
}

type mockInventoryRepo struct {
	inventories map[uuid.UUID][]inventoryDomain.Inventory
}

func (m *mockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, ids []uuid.UUID) (map[uuid.UUID][]inventoryDomain.Inventory, error) {
	return m.inventories, nil
}
