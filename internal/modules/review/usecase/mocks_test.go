package usecase_test

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/review/domain"
	"komecore/internal/modules/review/repository"

	"github.com/google/uuid"
)

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(nil)
}

type mockReviewRepo struct {
	reviews        map[uuid.UUID]*domain.Review
	deletedReviews map[uuid.UUID]bool
	createErr      error
	getErr         error
	deleteErr      error
	listErr        error
	summaryErr     error
}

func newMockReviewRepo() *mockReviewRepo {
	return &mockReviewRepo{
		reviews:        make(map[uuid.UUID]*domain.Review),
		deletedReviews: make(map[uuid.UUID]bool),
	}
}

func (m *mockReviewRepo) Create(_ context.Context, _ transaction.Executor, review *domain.Review) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.reviews[review.ID] = review
	return nil
}

func (m *mockReviewRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*domain.Review, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.deletedReviews[id] {
		return nil, nil
	}
	return m.reviews[id], nil
}

func (m *mockReviewRepo) Delete(_ context.Context, _ transaction.Executor, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedReviews[id] = true
	return nil
}

func (m *mockReviewRepo) ListByProductID(_ context.Context, _ transaction.Executor, params repository.ListReviewsParams) ([]domain.ReviewWithCustomer, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var res []domain.ReviewWithCustomer
	for _, r := range m.reviews {
		if r.ProductID == params.ProductID && !m.deletedReviews[r.ID] {
			res = append(res, domain.ReviewWithCustomer{
				Review:       *r,
				CustomerName: "Test Customer",
			})
		}
	}
	return res, len(res), nil
}

func (m *mockReviewRepo) GetRatingSummary(_ context.Context, _ transaction.Executor, productID uuid.UUID) (*domain.ProductRatingSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	count := 0
	sum := 0.0
	for _, r := range m.reviews {
		if r.ProductID == productID && !m.deletedReviews[r.ID] {
			count++
			sum += float64(r.Rating)
		}
	}
	rawAvg := 0.0
	if count > 0 {
		rawAvg = sum / float64(count)
	}
	summary := domain.NewProductRatingSummary(rawAvg, count)
	return &summary, nil
}

func (m *mockReviewRepo) HasReviewedOrder(_ context.Context, _ transaction.Executor, customerID, productID, orderID uuid.UUID) (bool, error) {
	for _, r := range m.reviews {
		if r.CustomerID == customerID && r.ProductID == productID && r.OrderID == orderID && !m.deletedReviews[r.ID] {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockReviewRepo) GetReviewedOrderIDs(_ context.Context, _ transaction.Executor, customerID, productID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, r := range m.reviews {
		if r.CustomerID == customerID && r.ProductID == productID && !m.deletedReviews[r.ID] {
			ids = append(ids, r.OrderID)
		}
	}
	return ids, nil
}

type mockProductRepo struct {
	products           map[uuid.UUID]*productDomain.Product
	updatedRatings     map[uuid.UUID]float64
	updatedReviewCount map[uuid.UUID]int
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products:           make(map[uuid.UUID]*productDomain.Product),
		updatedRatings:     make(map[uuid.UUID]float64),
		updatedReviewCount: make(map[uuid.UUID]int),
	}
}

func (m *mockProductRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*productDomain.Product, error) {
	return m.products[id], nil
}

func (m *mockProductRepo) UpdateRating(_ context.Context, _ transaction.Executor, id uuid.UUID, avg float64, count int) error {
	m.updatedRatings[id] = avg
	m.updatedReviewCount[id] = count
	if p, ok := m.products[id]; ok {
		p.AverageRating = avg
		p.ReviewCount = count
	}
	return nil
}

type mockOrderRepo struct {
	orders map[uuid.UUID]*orderDomain.Order
}

func (m *mockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*orderDomain.Order, error) {
	return m.orders[id], nil
}

func (m *mockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, params orderRepo.FindOrderParams) ([]orderDomain.Order, int, error) {
	var res []orderDomain.Order
	for _, o := range m.orders {
		if params.CustomerID != nil && o.CustomerID != *params.CustomerID {
			continue
		}
		if len(params.Statuses) > 0 {
			matched := false
			for _, st := range params.Statuses {
				if string(o.Status) == st {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		res = append(res, *o)
	}
	return res, len(res), nil
}

type mockOrderItemRepo struct {
	items map[uuid.UUID][]orderDomain.OrderItem
}

func (m *mockOrderItemRepo) ListByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]orderDomain.OrderItem, error) {
	return m.items[orderID], nil
}

func (m *mockOrderItemRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, orderIDs []uuid.UUID) ([]orderDomain.OrderItem, error) {
	var res []orderDomain.OrderItem
	for _, oID := range orderIDs {
		res = append(res, m.items[oID]...)
	}
	return res, nil
}
