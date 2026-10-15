package contract_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/bootstrap"
	"komecore/internal/httpx"
	appcookie "komecore/internal/httpx/cookie"
	appmiddleware "komecore/internal/httpx/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/courier/courierrepo"
	"komecore/internal/modules/courier/courierusecase"
	"komecore/internal/modules/discount/discountdomain"
	"komecore/internal/modules/discount/discountrepo"
	"komecore/internal/modules/discount/discountusecase"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/order/orderusecase"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	"komecore/internal/modules/payment/paymentusecase"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/modules/product/productusecase"
	"komecore/internal/modules/shop/shopdomain"
	"komecore/internal/modules/shop/shoprepo"
	"komecore/internal/modules/staff/staffdomain"
	"komecore/internal/modules/staff/staffrepo"
	"komecore/internal/modules/staff/staffusecase"
	"komecore/internal/pagination"
	"komecore/pkg/clock"
	applogger "komecore/pkg/logger"
	applimiter "komecore/pkg/ratelimit"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// loadOpenAPIRouter loads api/openapi.yaml and builds an openapi3filter router.
func loadOpenAPIRouter(t *testing.T) (*openapi3.T, routers.Router) {
	t.Helper()

	specPath := filepath.Join("..", "..", "api", "openapi.yaml")
	data, err := os.ReadFile(specPath)
	require.NoError(t, err, "failed to read api/openapi.yaml")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromData(data)
	require.NoError(t, err, "failed to parse OpenAPI 3.0 document")

	err = doc.Validate(context.Background())
	require.NoError(t, err, "OpenAPI 3.0 specification validation failed")

	// Set Servers to empty so router matches requests by URL Path without host restrictions
	doc.Servers = openapi3.Servers{}

	router, err := gorillamux.NewRouter(doc)
	require.NoError(t, err, "failed to create openapi3filter gorillamux router")

	return doc, router
}

// validateHTTPResponse validates an HTTP response against the OpenAPI specification.
func validateHTTPResponse(t *testing.T, router routers.Router, req *http.Request, resp *http.Response, bodyBytes []byte) {
	t.Helper()

	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err, "route not found in OpenAPI spec for %s %s", req.Method, req.URL.Path)

	reqValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	respValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqValidationInput,
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
		Body:                   io.NopCloser(bytes.NewReader(bodyBytes)),
	}

	ctx := context.Background()
	err = openapi3filter.ValidateResponse(ctx, respValidationInput)
	require.NoError(t, err, "response for %s %s failed OpenAPI schema validation: %v", req.Method, req.URL.Path, err)
}

// Mock Authenticator & Authorizer for contract tests
type contractAuth struct {
	authCtx *authctx.AuthContext
}

func (m *contractAuth) RequireAuth(_ transaction.Executor, _ transaction.Transactor, _ appcookie.CookieName) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := authctx.WithAuthContext(r.Context(), m.authCtx)
			return next(w, r.WithContext(ctx))
		}
	}
}

func (m *contractAuth) RequireAnyAuth(exec transaction.Executor, tran transaction.Transactor, _ ...appcookie.CookieName) appmiddleware.Middleware {
	return m.RequireAuth(exec, tran, appcookie.CookieCustomer)
}

func (m *contractAuth) RequireMultiAuth(exec transaction.Executor, tran transaction.Transactor, _ ...appcookie.CookieName) appmiddleware.Middleware {
	return m.RequireAuth(exec, tran, appcookie.CookieCustomer)
}

func (m *contractAuth) OptionalAuth(_ transaction.Executor, _ transaction.Transactor, _ ...appcookie.CookieName) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := authctx.WithAuthContext(r.Context(), m.authCtx)
			return next(w, r.WithContext(ctx))
		}
	}
}

func (m *contractAuth) RequireAccountType(_ ...authdomain.AccountType) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *contractAuth) RequireStaffRole(_ ...authdomain.RoleCode) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *contractAuth) RequirePermission(_ string) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *contractAuth) LoadActor(_ transaction.Executor) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *contractAuth) OptionalLoadActor(_ transaction.Executor) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

// Mock Repositories for contract testing using interface embedding

type contractProductRepo struct {
	productrepo.ProductRepository
	products []productdomain.Product
}

func (m *contractProductRepo) FindProducts(_ context.Context, _ transaction.Executor, _ productrepo.FindProductParams) ([]productdomain.Product, int, error) {
	return m.products, len(m.products), nil
}

func (m *contractProductRepo) FindProductsWithInventory(_ context.Context, _ transaction.Executor, _ productrepo.FindProductParams) ([]productdomain.ProductWithInventory, int, error) {
	items := make([]productdomain.ProductWithInventory, 0, len(m.products))
	for _, p := range m.products {
		items = append(items, productdomain.ProductWithInventory{
			Product:       p,
			TotalStock:    100,
			ReservedStock: 0,
		})
	}
	return items, len(items), nil
}

type contractProductImgRepo struct {
	productrepo.ProductImageRepository
}

func (m *contractProductImgRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]productdomain.ProductImage, error) {
	return make(map[uuid.UUID][]productdomain.ProductImage), nil
}

type contractInventoryRepo struct {
	inventoryrepo.InventoryRepository
}

func (m *contractInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventorydomain.Inventory, error) {
	return make(map[uuid.UUID][]inventorydomain.Inventory), nil
}

type contractShopRepo struct {
	shoprepo.ShopRepository
}

func (m *contractShopRepo) FindByIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shopdomain.Shop, error) {
	return []shopdomain.Shop{
		{
			ID:             uuid.MustParse("c0000000-0000-0000-0000-000000000001"),
			Name:           "Kome Central Store",
			Slug:           "kome-central-store",
			IsActive:       true,
			ApprovalStatus: shopdomain.ShopApprovalStatusApproved,
			CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}, nil
}

type contractCourierRepo struct {
	courierrepo.CourierRepository
}

func (m *contractCourierRepo) ListAll(_ context.Context, _ transaction.Executor) ([]string, error) {
	return []string{"jne", "sicepat", "pos"}, nil
}

type contractPaymentMethodRepo struct {
	paymentrepo.PaymentMethodRepository
}

func (m *contractPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ pagination.Sorts) ([]paymentdomain.PaymentMethod, error) {
	return []paymentdomain.PaymentMethod{
		{
			ID:          uuid.MustParse("d0000000-0000-0000-0000-000000000001"),
			Name:        "BCA Virtual Account",
			Code:        "bca_va",
			Provider:    "midtrans",
			Type:        "bank_transfer",
			IsActive:    true,
			Description: "Transfer via BCA Virtual Account",
			FeeType:     "fixed",
			FeeFixed:    4000,
			CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Instruction: &paymentdomain.PaymentInstruction{
				ID:        uuid.MustParse("d0000000-0000-0000-0000-000000000002"),
				Content:   "Transfer to BCA account 1234567890",
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}, nil
}

type contractAnalyticsRepo struct {
	staffrepo.AnalyticsRepository
}

func (m *contractAnalyticsRepo) GetRevenueOverTime(_ context.Context, _ transaction.Executor, _ string, _ time.Time) ([]staffdomain.RevenueBucket, error) {
	return []staffdomain.RevenueBucket{
		{
			Bucket:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Revenue: 12500000,
			Count:   42,
		},
	}, nil
}

func (m *contractAnalyticsRepo) GetOrderPipeline(_ context.Context, _ transaction.Executor) ([]staffdomain.OrderStatusCount, error) {
	return []staffdomain.OrderStatusCount{
		{
			Status:      "confirmed",
			Count:       25,
			TotalAmount: 18500000,
		},
	}, nil
}

func (m *contractAnalyticsRepo) GetTopProducts(_ context.Context, _ transaction.Executor, _ int) ([]staffdomain.TopProduct, error) {
	return []staffdomain.TopProduct{}, nil
}

type contractDiscountRepo struct {
	discountrepo.CouponRepository
}

func (m *contractDiscountRepo) GetByCode(_ context.Context, _ transaction.Executor, code string) (*discountdomain.Coupon, error) {
	if code == "PROMO2026" {
		maxDiscount := int64(50000)
		return &discountdomain.Coupon{
			ID:             uuid.MustParse("c0000000-0000-0000-0000-000000000001"),
			Code:           "PROMO2026",
			Type:           discountdomain.CouponTypePercentage,
			DiscountValue:  10,
			MinSpend:       100000,
			MaxDiscount:    &maxDiscount,
			QuotaTotal:     100,
			QuotaRemaining: 95,
			StartsAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ExpiresAt:      time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
			IsActive:       true,
			CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}, nil
	}
	return nil, apperror.NewNotFound("coupon not found")
}

type contractPricingService struct{}

func (m *contractPricingService) Calculate(_ context.Context, _ transaction.Executor, _ orderrepo.PricingInput) (*orderrepo.PricingResult, error) {
	methodID := uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	return &orderrepo.PricingResult{
		Address: orderrepo.PricingAddressResult{
			ID:            uuid.MustParse("e0000000-0000-0000-0000-000000000001"),
			RecipientName: "John Doe",
			FullAddress:   "Jl. Sudirman No. 1, Jakarta",
		},
		Shops: []orderrepo.PricingShopResult{
			{
				ShopID:   uuid.MustParse("c0000000-0000-0000-0000-000000000001"),
				ShopName: "Kome Central Store",
				ShopSlug: "kome-central-store",
				Subtotal: 150000,
				Total:    160000,
				Items: []orderrepo.PricingItemResult{
					{
						ProductID:   uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
						ProductName: "Organic Japonica Rice 5kg",
						UnitPrice:   75000,
						Quantity:    2,
						Subtotal:    150000,
						WeightGrams: 5000,
					},
				},
				SelectedCourier: orderrepo.SelectedCourierResult{
					Code:    "jne",
					Service: "REG",
					Fee:     10000,
				},
				CourierOptions: []orderrepo.CourierOption{
					{
						Code:    "jne",
						Service: "REG",
						Name:    "JNE Regular",
						ETD:     "2-3 days",
						Fee:     10000,
					},
				},
			},
		},
		Subtotal:         150000,
		TotalShippingFee: 10000,
		GrandTotal:       160000,
		PaymentMethods: []orderrepo.PaymentMethodPricingResult{
			{
				PaymentMethodID: methodID,
				Name:            "BCA Virtual Account",
				Type:            "bank_transfer",
				Description:     "Transfer via BCA Virtual Account",
				Fee:             4000,
				Subtotal:        160000,
				Total:           164000,
			},
		},
		SelectedPaymentMethod: &orderrepo.PaymentMethodPricingResult{
			PaymentMethodID: methodID,
			Name:            "BCA Virtual Account",
			Type:            "bank_transfer",
			Description:     "Transfer via BCA Virtual Account",
			Fee:             4000,
			Subtotal:        160000,
			Total:           164000,
		},
	}, nil
}

// newTestContractServer creates an in-memory test server for contract and snapshot verification.
func newTestContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	now := time.Date(2026, 10, 15, 8, 30, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(now)

	weight := 5000.0
	testProducts := []productdomain.Product{
		{
			ID:        uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
			SKU:       "PRD-001",
			Name:      "Organic Japonica Rice 5kg",
			Slug:      "organic-japonica-rice-5kg",
			Status:    productdomain.ProductStatusActive,
			Price:     75000,
			Weight:    &weight,
			CreatedAt: now,
		},
	}

	prodRepo := &contractProductRepo{products: testProducts}
	prodImgRepo := &contractProductImgRepo{}
	invRepo := &contractInventoryRepo{}
	shopRepo := &contractShopRepo{}
	courierRepo := &contractCourierRepo{}
	paymentMethodRepo := &contractPaymentMethodRepo{}
	analyticsRepo := &contractAnalyticsRepo{}
	discountRepo := &contractDiscountRepo{}
	pricingSvc := &contractPricingService{}

	findProductsUC := productusecase.NewFindProductsUsecase(
		prodRepo,
		invRepo,
		prodImgRepo,
		shopRepo,
		nil,
		nil,
	)

	courierSvc := courierusecase.NewCourierService(nil, courierRepo)
	listPaymentMethodUC := paymentusecase.NewListPaymentMethodUsecase(paymentMethodRepo, nil)
	staffSvc := staffusecase.NewStaffService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).WithAnalyticsRepository(analyticsRepo)
	discountSvc := discountusecase.NewDiscountService(discountRepo, nil, mockClock)
	checkoutUC := orderusecase.NewCheckoutUsecase(nil, pricingSvc)

	staffID := uuid.MustParse("a0000000-0000-0000-0000-000000000003")
	custID := uuid.MustParse("a0000000-0000-0000-0000-000000000004")
	authCtx := &authctx.AuthContext{
		UserID:          uuid.MustParse("a0000000-0000-0000-0000-000000000002"),
		SessionID:       uuid.MustParse("a0000000-0000-0000-0000-000000000001"),
		StaffID:         &staffID,
		CustomerID:      &custID,
		AccountType:     authctx.AccountTypeStaff,
		TokenType:       authctx.TokenTypeAccess,
		IsAuthenticated: true,
		Roles:           []string{string(authctx.RoleStaffAdmin)},
	}

	authMock := &contractAuth{authCtx: authCtx}

	log := applogger.NewSlogLogger("test")
	lim := applimiter.NewInMemorySlidingWindowLimiter(1*time.Second, 1000)

	container := &bootstrap.Container{
		Logger:             log,
		Limiter:            lim,
		CORSAllowedOrigins: []string{"*"},
		Authenticator:      authMock,
		Authorizer:         authMock,
		FindProducts:       *findProductsUC,
		Courier:            *courierSvc,
		ListPaymentMethod:  *listPaymentMethodUC,
		Staff:              *staffSvc,
		Discount:           *discountSvc,
		Checkout:           *checkoutUC,
	}

	router := bootstrap.NewRouter(container)
	return httptest.NewServer(router)
}
