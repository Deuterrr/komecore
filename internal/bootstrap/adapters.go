package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/payment/paymentusecase"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/modules/shop/shoprepo"
	"komecore/internal/modules/shop/shopusecase"
	"komecore/internal/modules/staff/staffrepo"
	"komecore/internal/modules/staff/staffusecase"
	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userusecase"
	appclock "komecore/pkg/clock"
)

// orderPaymentAdapter implements paymentusecase.OrderPaymentManager.
type orderPaymentAdapter struct {
	orderRepo     orderrepo.OrderRepository
	orderItemRepo orderrepo.OrderItemRepository
}

func newOrderPaymentAdapter(o orderrepo.OrderRepository, oi orderrepo.OrderItemRepository) *orderPaymentAdapter {
	return &orderPaymentAdapter{
		orderRepo:     o,
		orderItemRepo: oi,
	}
}

func (a *orderPaymentAdapter) GetOrderForPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*paymentusecase.OrderInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
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
		`, order.CustomerID).Scan(&customerName, &customerEmail)
	}

	return &paymentusecase.OrderInfo{
		ID:            order.ID,
		CustomerID:    order.CustomerID,
		Number:        order.Number,
		Total:         order.Total,
		CustomerEmail: customerEmail,
		CustomerName:  customerName,
	}, nil
}

func (a *orderPaymentAdapter) GetInvoiceNumber(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (string, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return "", err
	}
	if order == nil {
		return "", nil
	}
	return order.Number, nil
}

func (a *orderPaymentAdapter) ConfirmOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID, confirmedAt time.Time, handlingWindow time.Duration) ([]paymentusecase.OrderItemInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if order != nil {
		if err := order.Confirm(confirmedAt, handlingWindow); err != nil {
			return nil, fmt.Errorf("failed to confirm order: %w", err)
		}
		if err := a.orderRepo.UpdateStatusWithSLA(ctx, exec, order.ID, order.Status, order.ConfirmedAt, order.HandlingExpiresAt); err != nil {
			return nil, fmt.Errorf("failed to update order: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]paymentusecase.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = paymentusecase.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

func (a *orderPaymentAdapter) ExpireOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]paymentusecase.OrderItemInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if order != nil {
		if err := order.UpdateStatus(orderdomain.OrderStatusExpired); err != nil {
			return nil, fmt.Errorf("failed to expire order: %w", err)
		}
		if err := a.orderRepo.UpdateStatus(ctx, exec, order.ID, orderdomain.OrderStatusExpired); err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]paymentusecase.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = paymentusecase.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

func (a *orderPaymentAdapter) CancelOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]paymentusecase.OrderItemInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if order != nil {
		if err := order.UpdateStatus(orderdomain.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("failed to cancel order: %w", err)
		}
		if err := a.orderRepo.UpdateStatus(ctx, exec, order.ID, orderdomain.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]paymentusecase.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = paymentusecase.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

// orderDeliveryAdapter implements shipmentusecase.OrderDeliveryUpdater.
type orderDeliveryAdapter struct {
	orderRepo orderrepo.OrderRepository
}

func newOrderDeliveryAdapter(o orderrepo.OrderRepository) *orderDeliveryAdapter {
	return &orderDeliveryAdapter{orderRepo: o}
}

func (a *orderDeliveryAdapter) MarkOrderDelivered(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) error {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return apperror.NewNotFound("order not found")
	}
	if err := order.UpdateStatus(orderdomain.OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if err := a.orderRepo.UpdateStatus(ctx, exec, order.ID, orderdomain.OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// inventoryProductCheckerAdapter implements inventory usecase ProductChecker.
type inventoryProductCheckerAdapter struct {
	productRepo productrepo.ProductRepository
}

func newInventoryProductCheckerAdapter(p productrepo.ProductRepository) *inventoryProductCheckerAdapter {
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
	shopRepo shoprepo.ShopRepository
}

func newInventoryShopCheckerAdapter(s shoprepo.ShopRepository) *inventoryShopCheckerAdapter {
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
	stockHistoryRepo productrepo.ProductStockHistoryRepository
}

func newInventoryStockHistoryAdapter(r productrepo.ProductStockHistoryRepository) *inventoryStockHistoryAdapter {
	return &inventoryStockHistoryAdapter{stockHistoryRepo: r}
}

func (a *inventoryStockHistoryAdapter) RecordStockEvent(ctx context.Context, exec transaction.Executor, productID, shopID uuid.UUID, available int) error {
	return a.stockHistoryRepo.RecordStockEvent(ctx, exec, productdomain.ProductStockEvent{
		ProductID:  productID,
		ShopID:     shopID,
		Available:  available,
		RecordedAt: appclock.Now(),
	})
}

// shopProductAdapter implements shopusecase.ShopProductProvider.
type shopProductAdapter struct {
	inventoryRepo inventoryrepo.InventoryRepository
	productRepo   productrepo.ProductRepository
}

func newShopProductAdapter(i inventoryrepo.InventoryRepository, p productrepo.ProductRepository) *shopProductAdapter {
	return &shopProductAdapter{
		inventoryRepo: i,
		productRepo:   p,
	}
}

func (a *shopProductAdapter) GetShopProducts(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]shopusecase.ShopProductResult, error) {
	inventories, err := a.inventoryRepo.ListByShopID(ctx, exec, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop inventories: %w", err)
	}
	if len(inventories) == 0 {
		return []shopusecase.ShopProductResult{}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(inventories))
	for _, inv := range inventories {
		productIDs = append(productIDs, inv.ProductID)
	}
	products, err := a.productRepo.FindByIDs(ctx, exec, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products for shop: %w", err)
	}
	productMap := make(map[uuid.UUID]productdomain.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	results := make([]shopusecase.ShopProductResult, 0, len(inventories))
	for _, inv := range inventories {
		p, ok := productMap[inv.ProductID]
		if !ok {
			continue
		}
		results = append(results, shopusecase.ShopProductResult{
			Product: shopusecase.ShopProductInfo{
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
			Inventory: shopusecase.ShopProductInventoryInfo{
				TotalStock:    inv.TotalStock,
				ReservedStock: inv.ReservedStock,
			},
		})
	}

	return results, nil
}

// staffAccountAdapter implements staffusecase.AccountManager.
type staffAccountAdapter struct {
	accountRepo authrepo.AccountRepository
	sessionRepo authrepo.SessionRepository
}

func newStaffAccountAdapter(a authrepo.AccountRepository, s authrepo.SessionRepository) *staffAccountAdapter {
	return &staffAccountAdapter{
		accountRepo: a,
		sessionRepo: s,
	}
}

func (a *staffAccountAdapter) GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*staffusecase.AccountInfo, error) {
	acc, err := a.accountRepo.GetByEmail(ctx, exec, email)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staffusecase.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*staffusecase.AccountInfo, error) {
	acc, err := a.accountRepo.GetByUserID(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staffusecase.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffusecase.AccountInfo, error) {
	acc, err := a.accountRepo.GetByID(ctx, exec, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staffusecase.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) CreateStaffAccount(ctx context.Context, exec transaction.Executor, input staffusecase.CreateAccountInput) error {
	acc := authdomain.Account{
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

// userAccountAdapter implements userusecase.AccountReader.
type userAccountAdapter struct {
	accountRepo authrepo.AccountRepository
}

func newUserAccountAdapter(a authrepo.AccountRepository) *userAccountAdapter {
	return &userAccountAdapter{accountRepo: a}
}

func (a *userAccountAdapter) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*userusecase.UserAccount, error) {
	acc, err := a.accountRepo.GetByUserID(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &userusecase.UserAccount{
		Type: acc.Type,
	}, nil
}

// userSessionAdapter implements userusecase.SessionReader.
type userSessionAdapter struct {
	sessionRepo authrepo.SessionRepository
}

func newUserSessionAdapter(s authrepo.SessionRepository) *userSessionAdapter {
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

// userStaffProfileAdapter implements userusecase.StaffProfileProvider.
type userStaffProfileAdapter struct {
	staffRepo staffrepo.StaffRepository
}

func newUserStaffProfileAdapter(s staffrepo.StaffRepository) *userStaffProfileAdapter {
	return &userStaffProfileAdapter{staffRepo: s}
}

func (a *userStaffProfileAdapter) GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*userdomain.StaffProfile, error) {
	return a.staffRepo.GetProfileByUserID(ctx, exec, userID)
}
