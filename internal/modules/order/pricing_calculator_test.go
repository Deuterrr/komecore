package order

import (
	"context"
	"testing"

	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type pcMockProductRepo struct {
	products []product.Product
}

func (m *pcMockProductRepo) FindByIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]product.Product, error) {
	return m.products, nil
}
func (m *pcMockProductRepo) Save(_ context.Context, _ transaction.Executor, _ *product.Product) error {
	return nil
}
func (m *pcMockProductRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *pcMockProductRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*product.Product, error) {
	return nil, nil
}
func (m *pcMockProductRepo) GetBySlug(_ context.Context, _ transaction.Executor, _ string) (*product.Product, error) {
	return nil, nil
}
func (m *pcMockProductRepo) FindProducts(_ context.Context, _ transaction.Executor, _ product.FindProductParams) ([]product.Product, int, error) {
	return nil, 0, nil
}
func (m *pcMockProductRepo) FindProductsWithInventory(_ context.Context, _ transaction.Executor, _ product.FindProductParams) ([]product.ProductWithInventory, int, error) {
	return nil, 0, nil
}
func (m *pcMockProductRepo) UpdateRating(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ float64, _ int) error {
	return nil
}

type pcMockShopRepo struct {
	shops []shop.Shop
}

func (m *pcMockShopRepo) FindByIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shop.Shop, error) {
	return m.shops, nil
}
func (m *pcMockShopRepo) Save(_ context.Context, _ transaction.Executor, _ shop.Shop) error {
	return nil
}
func (m *pcMockShopRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *pcMockShopRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shop.Shop, error) {
	return nil, nil
}
func (m *pcMockShopRepo) GetBySlug(_ context.Context, _ transaction.Executor, _ string) (*shop.Shop, error) {
	return nil, nil
}
func (m *pcMockShopRepo) FindByParams(_ context.Context, _ transaction.Executor, _ shop.FindShopsParams) ([]shop.Shop, int, error) {
	return nil, 0, nil
}

type pcMockInventoryRepo struct {
	invMap map[uuid.UUID][]inventory.Inventory
}

func (m *pcMockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error) {
	return m.invMap, nil
}
func (m *pcMockInventoryRepo) GetByProductIDAndShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) (*inventory.Inventory, error) {
	return nil, nil
}
func (m *pcMockInventoryRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return nil, nil
}
func (m *pcMockInventoryRepo) ListByShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return nil, nil
}
func (m *pcMockInventoryRepo) Create(_ context.Context, _ transaction.Executor, _ *inventory.Inventory) error {
	return nil
}
func (m *pcMockInventoryRepo) Reserve(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *pcMockInventoryRepo) Release(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *pcMockInventoryRepo) Commit(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *pcMockInventoryRepo) Restock(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *pcMockInventoryRepo) Update(_ context.Context, _ transaction.Executor, _ *inventory.Inventory) error {
	return nil
}
func (m *pcMockInventoryRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type pcMockAddressRepo struct {
	addr *address.CustomerAddress
}

func (m *pcMockAddressRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*address.CustomerAddress, error) {
	return m.addr, nil
}
func (m *pcMockAddressRepo) GetDefaultByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*address.CustomerAddress, error) {
	return m.addr, nil
}
func (m *pcMockAddressRepo) ListByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]address.CustomerAddress, error) {
	return nil, nil
}
func (m *pcMockAddressRepo) ListByIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]address.CustomerAddress, error) {
	return nil, nil
}
func (m *pcMockAddressRepo) CountByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*int, error) {
	zero := 0
	return &zero, nil
}
func (m *pcMockAddressRepo) UnsetDefaultByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *pcMockAddressRepo) Save(_ context.Context, _ transaction.Executor, _ address.CustomerAddress) error {
	return nil
}
func (m *pcMockAddressRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *pcMockAddressRepo) DeleteByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

type pcMockPaymentMethodRepo struct {
	methods []payment.PaymentMethod
}

func (m *pcMockPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ pagination.Sorts) ([]payment.PaymentMethod, error) {
	return m.methods, nil
}
func (m *pcMockPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*payment.PaymentMethod, error) {
	for _, pm := range m.methods {
		if pm.ID == id {
			return &pm, nil
		}
	}
	return nil, nil
}
func (m *pcMockPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*payment.PaymentMethod, error) {
	return nil, nil
}
func (m *pcMockPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, _ payment.PaymentMethod) error {
	return nil
}
func (m *pcMockPaymentMethodRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

type pcMockShippingProvider struct{}

func (m *pcMockShippingProvider) CalculateShippingFee(_ float64, _ string) (int64, error) {
	return 10000, nil
}
func (m *pcMockShippingProvider) CalculateRates(_ context.Context, _ shipping.CalculateRatesInput) ([]shipping.RateOption, error) {
	return []shipping.RateOption{
		{Code: "jne", Service: "reg", Cost: 10000, Name: "JNE Regular"},
	}, nil
}

func TestCalculate_MultipleVariantQuantitiesExceedStock(t *testing.T) {
	productID := uuid.New()
	shopID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()

	prod := product.Product{
		ID:     productID,
		Name:   "Polo Shirt",
		Price:  100000,
		Status: product.ProductStatusActive,
	}

	sh := shop.Shop{
		ID:             shopID,
		Name:           "Fashion Store",
		IsActive:       true,
		ApprovalStatus: shop.ShopApprovalStatusApproved,
	}

	inv := inventory.Inventory{
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    5,
		ReservedStock: 0,
	}

	svc := NewPricingService(
		&pcMockAddressRepo{addr: &address.CustomerAddress{ID: addressID}},
		nil,
		nil,
		&pcMockInventoryRepo{invMap: map[uuid.UUID][]inventory.Inventory{productID: {inv}}},
		&pcMockPaymentMethodRepo{methods: []payment.PaymentMethod{{ID: uuid.New(), Name: "BCA", Type: "bank_transfer"}}},
		&pcMockProductRepo{products: []product.Product{prod}},
		&pcMockShippingProvider{},
		nil,
		&pcMockShopRepo{shops: []shop.Shop{sh}},
	)

	// Item 1 requires 3, Item 2 requires 3. Total = 6. Available = 5.
	input := PricingInput{
		CustomerID: customerID,
		AddressID:  &addressID,
		Shops: []PricingShopInput{
			{
				ShopID: shopID,
				Items: []PricingItemInput{
					{ProductID: productID, Quantity: 3, ItemOptions: cart.ItemOptions{"size": "M"}},
					{ProductID: productID, Quantity: 3, ItemOptions: cart.ItemOptions{"size": "L"}},
				},
			},
		},
	}

	_, err := svc.Calculate(context.Background(), nil, input)
	if err == nil {
		t.Fatal("expected conflict error when accumulated line item quantities exceed stock, got nil")
	}
}

func TestCalculate_MultipleVariantQuantitiesWithinStock_Succeeds(t *testing.T) {
	productID := uuid.New()
	shopID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()

	prod := product.Product{
		ID:     productID,
		Name:   "Polo Shirt",
		Price:  100000,
		Status: product.ProductStatusActive,
	}

	sh := shop.Shop{
		ID:             shopID,
		Name:           "Fashion Store",
		IsActive:       true,
		ApprovalStatus: shop.ShopApprovalStatusApproved,
	}

	inv := inventory.Inventory{
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    5,
		ReservedStock: 0,
	}

	svc := NewPricingService(
		&pcMockAddressRepo{addr: &address.CustomerAddress{ID: addressID}},
		nil,
		nil,
		&pcMockInventoryRepo{invMap: map[uuid.UUID][]inventory.Inventory{productID: {inv}}},
		&pcMockPaymentMethodRepo{methods: []payment.PaymentMethod{{ID: uuid.New(), Name: "BCA", Type: "bank_transfer"}}},
		&pcMockProductRepo{products: []product.Product{prod}},
		&pcMockShippingProvider{},
		nil,
		&pcMockShopRepo{shops: []shop.Shop{sh}},
	)

	// Item 1 requires 2, Item 2 requires 3. Total = 5 <= 5. Should succeed.
	input := PricingInput{
		CustomerID: customerID,
		AddressID:  &addressID,
		Shops: []PricingShopInput{
			{
				ShopID: shopID,
				Items: []PricingItemInput{
					{ProductID: productID, Quantity: 2, ItemOptions: cart.ItemOptions{"size": "M"}},
					{ProductID: productID, Quantity: 3, ItemOptions: cart.ItemOptions{"size": "L"}},
				},
			},
		},
	}

	res, err := svc.Calculate(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("expected calculate to succeed, got error: %v", err)
	}
	if res.Subtotal != 500000 {
		t.Errorf("expected subtotal 500000, got %d", res.Subtotal)
	}
}
