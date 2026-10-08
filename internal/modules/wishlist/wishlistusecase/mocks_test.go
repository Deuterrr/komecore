package wishlistusecase_test

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/wishlist/wishlistdomain"

	"github.com/google/uuid"
)

type mockWishlistRepo struct {
	items     map[string]wishlistdomain.WishlistItem
	addErr    error
	removeErr error
	listErr   error
	existsErr error
}

func newMockWishlistRepo() *mockWishlistRepo {
	return &mockWishlistRepo{
		items: make(map[string]wishlistdomain.WishlistItem),
	}
}

func (m *mockWishlistRepo) key(customerID, productID uuid.UUID) string {
	return customerID.String() + ":" + productID.String()
}

func (m *mockWishlistRepo) Add(_ context.Context, _ transaction.Executor, item wishlistdomain.WishlistItem) error {
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

func (m *mockWishlistRepo) ListByCustomerID(_ context.Context, _ transaction.Executor, customerID uuid.UUID) ([]wishlistdomain.WishlistItem, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var res []wishlistdomain.WishlistItem
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
	products map[uuid.UUID]*productdomain.Product
	getErr   error
	findErr  error
}

func (m *mockProductRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*productdomain.Product, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.products[id], nil
}

func (m *mockProductRepo) FindByIDs(_ context.Context, _ transaction.Executor, ids []uuid.UUID) ([]productdomain.Product, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var res []productdomain.Product
	for _, id := range ids {
		if p, ok := m.products[id]; ok {
			res = append(res, *p)
		}
	}
	return res, nil
}

type mockInventoryRepo struct {
	inventories map[uuid.UUID][]inventorydomain.Inventory
}

func (m *mockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, ids []uuid.UUID) (map[uuid.UUID][]inventorydomain.Inventory, error) {
	return m.inventories, nil
}
