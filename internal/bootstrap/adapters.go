package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"komecore/internal/common/authctx"
	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	authenDomain "komecore/internal/modules/auth/domain"
	authenRepo "komecore/internal/modules/auth/repository"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	paymentUsecase "komecore/internal/modules/payment/usecase"
	productDomain "komecore/internal/modules/product/domain"
	productRepo "komecore/internal/modules/product/repository"
	shopRepo "komecore/internal/modules/shop/repository"
	shopUsecase "komecore/internal/modules/shop/usecase"
	staffRepo "komecore/internal/modules/staff/repository"
	staffUsecase "komecore/internal/modules/staff/usecase"
	userDomain "komecore/internal/modules/user/domain"
	userUsecase "komecore/internal/modules/user/usecase"
	appclock "komecore/pkg/clock"
)

// orderPaymentAdapter implements paymentUsecase.OrderPaymentManager.
type orderPaymentAdapter struct {
	orderRepo     orderRepo.OrderRepository
	orderItemRepo orderRepo.OrderItemRepository
}

func newOrderPaymentAdapter(o orderRepo.OrderRepository, oi orderRepo.OrderItemRepository) *orderPaymentAdapter {
	return &orderPaymentAdapter{
		orderRepo:     o,
		orderItemRepo: oi,
	}
}

func (a *orderPaymentAdapter) GetOrderForPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*paymentUsecase.OrderInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return nil, nil
	}
	return &paymentUsecase.OrderInfo{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Number:     order.Number,
		Total:      order.Total,
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

func (a *orderPaymentAdapter) ConfirmOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID, confirmedAt time.Time, handlingWindow time.Duration) ([]paymentUsecase.OrderItemInfo, error) {
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

	result := make([]paymentUsecase.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = paymentUsecase.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

func (a *orderPaymentAdapter) ExpireOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]paymentUsecase.OrderItemInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if order != nil {
		if err := order.UpdateStatus(orderDomain.OrderStatusExpired); err != nil {
			return nil, fmt.Errorf("failed to expire order: %w", err)
		}
		if err := a.orderRepo.UpdateStatus(ctx, exec, order.ID, orderDomain.OrderStatusExpired); err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]paymentUsecase.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = paymentUsecase.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

func (a *orderPaymentAdapter) CancelOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]paymentUsecase.OrderItemInfo, error) {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}
	if order != nil {
		if err := order.UpdateStatus(orderDomain.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("failed to cancel order: %w", err)
		}
		if err := a.orderRepo.UpdateStatus(ctx, exec, order.ID, orderDomain.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	items, err := a.orderItemRepo.ListByOrderID(ctx, exec, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}

	result := make([]paymentUsecase.OrderItemInfo, len(items))
	for i, item := range items {
		result[i] = paymentUsecase.OrderItemInfo{
			ProductID: item.ProductID,
			ShopID:    item.ShopID,
			Quantity:  item.Quantity,
		}
	}
	return result, nil
}

// orderDeliveryAdapter implements shipmentUsecase.OrderDeliveryUpdater.
type orderDeliveryAdapter struct {
	orderRepo orderRepo.OrderRepository
}

func newOrderDeliveryAdapter(o orderRepo.OrderRepository) *orderDeliveryAdapter {
	return &orderDeliveryAdapter{orderRepo: o}
}

func (a *orderDeliveryAdapter) MarkOrderDelivered(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) error {
	order, err := a.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return apperrors.NewNotFound("order not found")
	}
	if err := order.UpdateStatus(orderDomain.OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if err := a.orderRepo.UpdateStatus(ctx, exec, order.ID, orderDomain.OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// inventoryProductCheckerAdapter implements inventory usecase ProductChecker.
type inventoryProductCheckerAdapter struct {
	productRepo productRepo.ProductRepository
}

func newInventoryProductCheckerAdapter(p productRepo.ProductRepository) *inventoryProductCheckerAdapter {
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
	shopRepo shopRepo.ShopRepository
}

func newInventoryShopCheckerAdapter(s shopRepo.ShopRepository) *inventoryShopCheckerAdapter {
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
	stockHistoryRepo productRepo.ProductStockHistoryRepository
}

func newInventoryStockHistoryAdapter(r productRepo.ProductStockHistoryRepository) *inventoryStockHistoryAdapter {
	return &inventoryStockHistoryAdapter{stockHistoryRepo: r}
}

func (a *inventoryStockHistoryAdapter) RecordStockEvent(ctx context.Context, exec transaction.Executor, productID, shopID uuid.UUID, available int) error {
	return a.stockHistoryRepo.RecordStockEvent(ctx, exec, productDomain.ProductStockEvent{
		ProductID:  productID,
		ShopID:     shopID,
		Available:  available,
		RecordedAt: appclock.Now(),
	})
}

// shopProductAdapter implements shopUsecase.ShopProductProvider.
type shopProductAdapter struct {
	inventoryRepo inventoryRepo.InventoryRepository
	productRepo   productRepo.ProductRepository
}

func newShopProductAdapter(i inventoryRepo.InventoryRepository, p productRepo.ProductRepository) *shopProductAdapter {
	return &shopProductAdapter{
		inventoryRepo: i,
		productRepo:   p,
	}
}

func (a *shopProductAdapter) GetShopProducts(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]shopUsecase.ShopProductResult, error) {
	inventories, err := a.inventoryRepo.ListByShopID(ctx, exec, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop inventories: %w", err)
	}
	if len(inventories) == 0 {
		return []shopUsecase.ShopProductResult{}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(inventories))
	for _, inv := range inventories {
		productIDs = append(productIDs, inv.ProductID)
	}
	products, err := a.productRepo.FindByIDs(ctx, exec, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products for shop: %w", err)
	}
	productMap := make(map[uuid.UUID]productDomain.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	results := make([]shopUsecase.ShopProductResult, 0, len(inventories))
	for _, inv := range inventories {
		p, ok := productMap[inv.ProductID]
		if !ok {
			continue
		}
		results = append(results, shopUsecase.ShopProductResult{
			Product: shopUsecase.ShopProductInfo{
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
			Inventory: shopUsecase.ShopProductInventoryInfo{
				TotalStock:    inv.TotalStock,
				ReservedStock: inv.ReservedStock,
			},
		})
	}

	return results, nil
}

// staffAccountAdapter implements staffUsecase.AccountManager.
type staffAccountAdapter struct {
	accountRepo authenRepo.AccountRepository
	sessionRepo authenRepo.SessionRepository
}

func newStaffAccountAdapter(a authenRepo.AccountRepository, s authenRepo.SessionRepository) *staffAccountAdapter {
	return &staffAccountAdapter{
		accountRepo: a,
		sessionRepo: s,
	}
}

func (a *staffAccountAdapter) GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*staffUsecase.AccountInfo, error) {
	acc, err := a.accountRepo.GetByEmail(ctx, exec, email)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staffUsecase.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*staffUsecase.AccountInfo, error) {
	acc, err := a.accountRepo.GetByUserID(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staffUsecase.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffUsecase.AccountInfo, error) {
	acc, err := a.accountRepo.GetByID(ctx, exec, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &staffUsecase.AccountInfo{
		ID:     acc.ID,
		UserID: acc.UserID,
		Email:  acc.Email,
	}, nil
}

func (a *staffAccountAdapter) CreateStaffAccount(ctx context.Context, exec transaction.Executor, input staffUsecase.CreateAccountInput) error {
	acc := authenDomain.Account{
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

// userAccountAdapter implements userUsecase.AccountReader.
type userAccountAdapter struct {
	accountRepo authenRepo.AccountRepository
}

func newUserAccountAdapter(a authenRepo.AccountRepository) *userAccountAdapter {
	return &userAccountAdapter{accountRepo: a}
}

func (a *userAccountAdapter) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*userUsecase.UserAccount, error) {
	acc, err := a.accountRepo.GetByUserID(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	return &userUsecase.UserAccount{
		Type: acc.Type,
	}, nil
}

// userSessionAdapter implements userUsecase.SessionReader.
type userSessionAdapter struct {
	sessionRepo authenRepo.SessionRepository
}

func newUserSessionAdapter(s authenRepo.SessionRepository) *userSessionAdapter {
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

// userStaffProfileAdapter implements userUsecase.StaffProfileProvider.
type userStaffProfileAdapter struct {
	staffRepo staffRepo.StaffRepository
}

func newUserStaffProfileAdapter(s staffRepo.StaffRepository) *userStaffProfileAdapter {
	return &userStaffProfileAdapter{staffRepo: s}
}

func (a *userStaffProfileAdapter) GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*userDomain.StaffProfile, error) {
	return a.staffRepo.GetProfileByUserID(ctx, exec, userID)
}
