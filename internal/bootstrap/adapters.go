package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/order"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"
	"komecore/internal/modules/staff"
	"komecore/internal/modules/user"
	appclock "komecore/pkg/clock"
)

// orderPaymentAdapter implements payment.OrderPaymentManager.
type orderPaymentAdapter struct {
	orderRepo     order.Repository
	orderItemRepo order.ItemRepository
}

func newOrderPaymentAdapter(o order.Repository, oi order.ItemRepository) *orderPaymentAdapter {
	return &orderPaymentAdapter{
		orderRepo:     o,
		orderItemRepo: oi,
	}
}

func (a *orderPaymentAdapter) GetOrderForPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*payment.OrderInfo, error) {
	ord, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	if ord == nil {
		return nil, nil
	}

	var customerEmail, customerName string
	if exec != nil {
		_ = exec.QueryRow(ctx, `
			SELECT COALESCE(u.name, ''), COALESCE(a.email, '')
			FROM customers c
			JOIN users u ON u.id = c.user_id
			JOIN accounts a ON a.user_id = u.id
			WHERE c.id = $1
		`, ord.CustomerID).Scan(&customerName, &customerEmail)
	}

	return &payment.OrderInfo{
		ID:            ord.ID,
		CustomerID:    ord.CustomerID,
		Number:        ord.Number,
		Total:         ord.Total,
		CustomerEmail: customerEmail,
		CustomerName:  customerName,
	}, nil
}

func (a *orderPaymentAdapter) GetInvoiceNumber(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (string, error) {
	ord, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return "", err
	}
	if ord == nil {
		return "", nil
	}
	return ord.Number, nil
}

func (a *orderPaymentAdapter) ConfirmOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID, confirmedAt time.Time, handlingWindow time.Duration) ([]payment.OrderItemInfo, error) {
	ord, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if ord != nil {
		if err := ord.Confirm(confirmedAt, handlingWindow); err != nil {
			return nil, fmt.Errorf("failed to confirm order: %w", err)
		}
		if err := a.orderRepo.UpdateStatusWithSLA(ctx, exec, ord.ID, ord.Status, ord.ConfirmedAt, ord.HandlingExpiresAt); err != nil {
			return nil, fmt.Errorf("failed to update order: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]payment.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = payment.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

func (a *orderPaymentAdapter) ExpireOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]payment.OrderItemInfo, error) {
	ord, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if ord != nil {
		if err := ord.UpdateStatus(order.OrderStatusExpired); err != nil {
			return nil, fmt.Errorf("failed to expire order: %w", err)
		}
		if err := a.orderRepo.UpdateStatus(ctx, exec, ord.ID, order.OrderStatusExpired); err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]payment.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = payment.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

func (a *orderPaymentAdapter) CancelOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]payment.OrderItemInfo, error) {
	ord, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if ord != nil {
		if err := ord.UpdateStatus(order.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("failed to cancel order: %w", err)
		}
		if err := a.orderRepo.UpdateStatus(ctx, exec, ord.ID, order.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]payment.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = payment.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

// orderDeliveryAdapter implements shipment.OrderDeliveryUpdater.
type orderDeliveryAdapter struct {
	orderRepo order.Repository
}

func newOrderDeliveryAdapter(o order.Repository) *orderDeliveryAdapter {
	return &orderDeliveryAdapter{orderRepo: o}
}

func (a *orderDeliveryAdapter) MarkOrderDelivered(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) error {
	ord, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if ord == nil {
		return apperror.NewNotFound("order not found")
	}
	if err := ord.UpdateStatus(order.OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if err := a.orderRepo.UpdateStatus(ctx, exec, ord.ID, order.OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// inventoryProductCheckerAdapter implements inventory usecase ProductChecker.
type inventoryProductCheckerAdapter struct {
	productRepo product.Repository
}

func newInventoryProductCheckerAdapter(p product.Repository) *inventoryProductCheckerAdapter {
	return &inventoryProductCheckerAdapter{productRepo: p}
}

func (a *inventoryProductCheckerAdapter) ProductExists(ctx context.Context, exec transaction.Executor, productID uuid.UUID) (bool, error) {
	p, err := a.productRepo.GetByID(ctx, exec, productID)
	if err != nil {
		return false, err
	}
	return p != nil, nil
}

// inventoryShopCheckerAdapter implements inventory usecase ShopChecker.
type inventoryShopCheckerAdapter struct {
	shopRepo shop.Repository
}

func newInventoryShopCheckerAdapter(s shop.Repository) *inventoryShopCheckerAdapter {
	return &inventoryShopCheckerAdapter{shopRepo: s}
}

func (a *inventoryShopCheckerAdapter) ShopExists(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) (bool, error) {
	s, err := a.shopRepo.GetByID(ctx, exec, shopID)
	if err != nil {
		return false, err
	}
	return s != nil, nil
}

// inventoryStockHistoryAdapter implements inventory usecase StockHistoryRecorder.
type inventoryStockHistoryAdapter struct {
	stockHistoryRepo product.StockHistoryRepository
}

func newInventoryStockHistoryAdapter(r product.StockHistoryRepository) *inventoryStockHistoryAdapter {
	return &inventoryStockHistoryAdapter{stockHistoryRepo: r}
}

func (a *inventoryStockHistoryAdapter) RecordStockEvent(ctx context.Context, exec transaction.Executor, productID, shopID uuid.UUID, available int) error {
	return a.stockHistoryRepo.RecordStockEvent(ctx, exec, product.ProductStockEvent{
		ProductID:  productID,
		ShopID:     shopID,
		Available:  available,
		RecordedAt: appclock.Now(),
	})
}

// shopProductAdapter implements shop.ShopProductProvider.
type shopProductAdapter struct {
	inventoryRepo inventory.Repository
	productRepo   product.Repository
}

func newShopProductAdapter(i inventory.Repository, p product.Repository) *shopProductAdapter {
	return &shopProductAdapter{
		inventoryRepo: i,
		productRepo:   p,
	}
}

func (a *shopProductAdapter) GetShopProducts(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]shop.ShopProductResult, error) {
	inventories, err := a.inventoryRepo.ListByShopID(ctx, exec, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop inventories: %w", err)
	}
	if len(inventories) == 0 {
		return []shop.ShopProductResult{}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(inventories))
	for _, inv := range inventories {
		productIDs = append(productIDs, inv.ProductID)
	}
	products, err := a.productRepo.FindByIDs(ctx, exec, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products for shop: %w", err)
	}
	productMap := make(map[uuid.UUID]product.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	results := make([]shop.ShopProductResult, 0, len(inventories))
	for _, inv := range inventories {
		p, ok := productMap[inv.ProductID]
		if !ok {
			continue
		}
		results = append(results, shop.ShopProductResult{
			Product: shop.ShopProductInfo{
				ID:          p.ID,
				SKU:         p.SKU,
				Name:        p.Name,
				Slug:        p.Slug,
				Description: p.Description,
				Status:      string(p.Status),
				Price:       p.Price,
				Weight:      p.Weight,
				CreatedAt:   p.CreatedAt,
				UpdatedAt:   p.UpdatedAt,
			},
			Inventory: shop.ShopProductInventoryInfo{
				TotalStock:    inv.TotalStock,
				ReservedStock: inv.ReservedStock,
			},
		})
	}

	return results, nil
}

// staffAccountAdapter implements staff.AccountManager.
type staffAccountAdapter struct {
	accountRepo auth.AccountRepository
	sessionRepo auth.SessionRepository
}

func newStaffAccountAdapter(a auth.AccountRepository, s auth.SessionRepository) *staffAccountAdapter {
	return &staffAccountAdapter{
		accountRepo: a,
		sessionRepo: s,
	}
}

func (a *staffAccountAdapter) GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*staff.AccountInfo, error) {
	acc, err := a.accountRepo.GetByEmail(ctx, exec, email)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staff.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*staff.AccountInfo, error) {
	acc, err := a.accountRepo.GetByUserID(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staff.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staff.AccountInfo, error) {
	acc, err := a.accountRepo.GetByID(ctx, exec, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staff.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) CreateStaffAccount(ctx context.Context, exec transaction.Executor, input staff.CreateAccountInput) error {
	acc := auth.Account{
		ID:        input.ID,
		UserID:    input.UserID,
		Email:     input.Email,
		Password:  input.Password,
		Type:      authctx.AccountTypeStaff,
		CreatedAt: input.CreatedAt,
	}
	return a.accountRepo.Create(ctx, exec, acc)
}

func (a *staffAccountAdapter) DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	return a.accountRepo.DeleteByUserID(ctx, exec, userID)
}

func (a *staffAccountAdapter) RevokeSessionsByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	return a.sessionRepo.RevokeAllByUserID(ctx, exec, userID)
}

// userAccountAdapter implements user.AccountReader.
type userAccountAdapter struct {
	accountRepo auth.AccountRepository
}

func newUserAccountAdapter(a auth.AccountRepository) *userAccountAdapter {
	return &userAccountAdapter{accountRepo: a}
}

func (a *userAccountAdapter) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*user.UserAccount, error) {
	acc, err := a.accountRepo.GetByUserID(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &user.UserAccount{
		Type: acc.Type,
	}, nil
}

// userSessionAdapter implements user.SessionReader.
type userSessionAdapter struct {
	sessionRepo auth.SessionRepository
}

func newUserSessionAdapter(s auth.SessionRepository) *userSessionAdapter {
	return &userSessionAdapter{sessionRepo: s}
}

func (a *userSessionAdapter) GetLastActivity(ctx context.Context, exec transaction.Executor, sessionID uuid.UUID) (*time.Time, error) {
	sess, err := a.sessionRepo.GetByID(ctx, exec, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	return sess.LastActivityAt, nil
}

// userStaffProfileAdapter implements user.StaffProfileProvider.
type userStaffProfileAdapter struct {
	staffRepo staff.Repository
}

func newUserStaffProfileAdapter(s staff.Repository) *userStaffProfileAdapter {
	return &userStaffProfileAdapter{staffRepo: s}
}

func (a *userStaffProfileAdapter) GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*user.StaffProfile, error) {
	return a.staffRepo.GetProfileByUserID(ctx, exec, userID)
}
