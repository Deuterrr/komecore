package bootstrap

import (
	"time"

	applogger "komecore/pkg/logger"
	applimiter "komecore/pkg/ratelimit"

	appconfig "komecore/internal/config"
	transaction "komecore/internal/infra/transactor"
	appclock "komecore/pkg/clock"
	imgSvc "komecore/pkg/imageutil"
	mailerSvc "komecore/pkg/mailer"
	otpSvc "komecore/pkg/otp"
	sGen "komecore/pkg/slug"

	"komecore/internal/modules/address/addresspersistence"
	"komecore/internal/modules/auth/authpersistence"
	"komecore/internal/modules/cart/cartpersistence"
	"komecore/internal/modules/courier/courierpersistence"
	"komecore/internal/modules/discount/discountpersistence"
	"komecore/internal/modules/inventory/inventorypersistence"
	"komecore/internal/modules/order/orderpersistence"
	"komecore/internal/modules/payment/paymentpersistence"
	"komecore/internal/modules/product/productpersistence"
	"komecore/internal/modules/review/reviewpersistence"
	"komecore/internal/modules/shipment/shipmentpersistence"
	"komecore/internal/modules/shop/shoppersistence"
	"komecore/internal/modules/staff/staffpersistence"
	"komecore/internal/modules/user/userpersistence"
	"komecore/internal/modules/wishlist/wishlistpersistence"

	"komecore/internal/modules/auth/authrepo"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/order/ordersvc"

	"komecore/internal/modules/address/addressrepo"
	"komecore/internal/modules/cart/cartrepo"
	"komecore/internal/modules/courier/courierrepo"
	"komecore/internal/modules/discount/discountrepo"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/payment/paymentrepo"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/modules/review/reviewrepo"
	"komecore/internal/modules/shipment/shipmentrepo"
	"komecore/internal/modules/shop/shoprepo"
	"komecore/internal/modules/staff/staffrepo"
	"komecore/internal/modules/user/userrepo"
	"komecore/internal/modules/wishlist/wishlistrepo"

	"komecore/internal/modules/address/addressusecase"
	"komecore/internal/modules/auth/authusecase"
	"komecore/internal/modules/cart/cartusecase"
	"komecore/internal/modules/courier/courierusecase"
	"komecore/internal/modules/discount/discountusecase"
	"komecore/internal/modules/inventory/inventoryusecase"
	"komecore/internal/modules/order/orderusecase"
	"komecore/internal/modules/payment/paymentusecase"
	"komecore/internal/modules/product/productusecase"
	"komecore/internal/modules/review/reviewusecase"
	"komecore/internal/modules/shipment/shipmentusecase"
	"komecore/internal/modules/shop/shopusecase"
	"komecore/internal/modules/staff/staffusecase"
	"komecore/internal/modules/user/userusecase"
	"komecore/internal/modules/wishlist/wishlistusecase"

	appmiddleware "komecore/internal/common/middleware"
	"komecore/internal/infra/cache"
	paymentgateway "komecore/internal/infra/payment-gateway"
)

type Container struct {
	Logger             applogger.Logger
	AuditLogger        applogger.AuditLogger
	CORSAllowedOrigins []string
	Authenticator      authrepo.Authenticator
	Authorizer         authrepo.Authorizer
	DBExecutor         transaction.Executor
	DBTransactor       transaction.Transactor
	GoogleOAuth        appconfig.GoogleOAuthConfig
	Cache              cache.Cache
	Idempotency        *appmiddleware.IdempotencyMiddleware
	paymentMethodRepo  paymentrepo.PaymentMethodRepository
	paymentGateway     paymentgateway.Provider

	FindProducts     productusecase.FindProductsUsecase
	GetProduct       productusecase.GetProductUsecase
	SaveProduct      productusecase.SaveProductUsecase
	DeleteProduct    productusecase.DeleteProductUsecase
	AddProductImages productusecase.AddProductImagesUsecase
	GetProductStats  productusecase.GetProductStatsUsecase
	Inventory inventoryusecase.InventoryService

	Me                   authusecase.MeUsecase
	LoginCustomer        authusecase.LoginCustomerUsecase
	LoginStaff           authusecase.LoginStaffUsecase
	RegisterCustomer     authusecase.RegisterCustomerUsecase
	VerifyAccount        authusecase.VerifyAccountUsecase
	GetAccount           authusecase.GetAccountUsecase
	Logout               authusecase.LogoutUsecase
	AuthenticateOAuth    authusecase.AuthenticateOAuthUsecase
	RequestPasswordReset authusecase.RequestPasswordResetUsecase
	VerifyPasswordReset  authusecase.VerifyPasswordResetUsecase
	ResetPassword        authusecase.ResetPasswordUsecase
	RefreshToken         authusecase.RefreshTokenUsecase
	DeleteAccount        authusecase.DeleteAccountUsecase

	Staff staffusecase.StaffService

	Cart     cartusecase.CartService
	Checkout orderusecase.CheckoutUsecase

	User userusecase.UserService

	Address addressusecase.AddressService

	Shop shopusecase.ShopService

	SavePaymentMethod      paymentusecase.SavePaymentMethodUsecase
	ListPaymentMethod      paymentusecase.ListPaymentMethodUsecase
	ProcessPaymentWebhook  paymentusecase.ProcessPaymentWebhookUsecase
	SavePaymentInstruction paymentusecase.SavePaymentInstructionUsecase
	GetPaymentDetail       paymentusecase.GetPaymentDetailUsecase
	CheckPaymentStatus     paymentusecase.CheckPaymentStatusUsecase
	SyncPendingPayments    paymentusecase.SyncPendingPaymentsUsecase
	ExpirePastDuePayments  paymentusecase.ExpirePastDuePaymentsUsecase
	SyncPaymentMethods     paymentusecase.SyncPaymentMethodsUsecase
	ProcessOrderRefund     paymentusecase.ProcessOrderRefundUsecase

	Courier courierusecase.CourierService

	EstimateShippingOptions shipmentusecase.EstimateShippingOptionsUsecase
	UpdateShipmentStatus    shipmentusecase.UpdateShipmentStatusUsecase
	UpdateShipment          shipmentusecase.UpdateShipmentUsecase

	CreateOrder             orderusecase.CreateOrderUsecase
	FindOrders              orderusecase.FindOrdersUsecase
	GetOrder                orderusecase.GetOrderUsecase
	UpdateOrderStatus       orderusecase.UpdateOrderStatusUsecase
	DispatchShopShipment    orderusecase.DispatchShopShipmentUsecase
	GetOrderTracking        orderusecase.GetOrderTrackingUsecase
	ExpireUnfulfilledOrders orderusecase.ExpireUnfulfilledOrdersUsecase

	Wishlist wishlistusecase.WishlistService
	Review   reviewusecase.ReviewService
	Discount discountusecase.DiscountService

	Limiter applimiter.Limiter
}

func NewContainer(cfg Config, infra *Dependency) *Container {
	log := applogger.NewZapLogger(cfg.App.Env)
	auditLogger := applogger.NewLogAuditLogger(log)

	repos := initRepositories()
	svcs := initSharedServices(cfg, infra, repos)

	return buildContainer(cfg, infra, repos, svcs, log, auditLogger)
}

type repositories struct {
	product             productrepo.ProductRepository
	productImage        productrepo.ProductImageRepository
	productPerformance  productrepo.ProductPerformanceRepository
	productStockHistory productrepo.ProductStockHistoryRepository
	inventory           inventoryrepo.InventoryRepository
	account             authrepo.AccountRepository
	challenge           authrepo.VerificationChallengeRepository
	oauth               authrepo.OAuthConnectionRepository
	session             authrepo.SessionRepository
	refreshToken        authrepo.RefreshTokenRepository
	cart                cartrepo.CartRepository
	user                userrepo.UserRepository
	address             addressrepo.CustomerAddressRepository
	addressShop         addressrepo.ShopAddressRepository
	payment             paymentrepo.PaymentRepository
	paymentMethod       paymentrepo.PaymentMethodRepository
	paymentEvent        paymentrepo.PaymentEventRepository
	paymentInstruction  paymentrepo.PaymentInstructionRepository
	paymentChannelData  paymentrepo.PaymentChannelDataRepository
	paymentWebhookEvent paymentrepo.PaymentWebhookEventRepository
	shop                shoprepo.ShopRepository
	courier             courierrepo.CourierRepository
	staff               staffrepo.StaffRepository
	customer            authrepo.CustomerRepository
	membership          staffrepo.StaffMembershipRepository
	role                staffrepo.RoleRepository
	order               orderrepo.OrderRepository
	orderItem           orderrepo.OrderItemRepository
	invoice             orderrepo.InvoiceRepository
	invoiceItem         orderrepo.InvoiceItemRepository
	shipment            shipmentrepo.ShipmentRepository
	wishlist            wishlistrepo.WishlistRepository
	review              reviewrepo.ReviewRepository
	coupon              discountrepo.CouponRepository
}

func initRepositories() *repositories {
	return &repositories{
		product:             productpersistence.NewProductRepository(),
		productImage:        productpersistence.NewProductImageRepository(),
		productPerformance:  productpersistence.NewProductPerformanceRepository(),
		productStockHistory: productpersistence.NewProductStockHistoryRepository(),
		inventory:           inventorypersistence.NewInventoryRepository(),
		account:             authpersistence.NewAccountRepository(),
		challenge:           authpersistence.NewChallengeRepository(),
		oauth:               authpersistence.NewOAuthConnectionRepository(),
		session:             authpersistence.NewSessionRepository(),
		refreshToken:        authpersistence.NewRefreshTokenRepository(),
		cart:                cartpersistence.NewCartRepository(),
		user:                userpersistence.NewUserRepository(),
		address:             addresspersistence.NewCustomerAddressRepository(),
		addressShop:         addresspersistence.NewShopAddressRepository(),
		payment:             paymentpersistence.NewPaymentRepository(),
		paymentMethod:       paymentpersistence.NewPaymentMethodRepository(),
		paymentEvent:        paymentpersistence.NewPaymentEventRepository(),
		paymentInstruction:  paymentpersistence.NewPaymentInstructionRepository(),
		paymentChannelData:  paymentpersistence.NewPaymentChannelDataRepository(),
		paymentWebhookEvent: paymentpersistence.NewPaymentWebhookEventRepository(),
		shop:                shoppersistence.NewShopRepository(),
		courier:             courierpersistence.NewCourierRepository(),
		staff:               staffpersistence.NewStaffRepository(),
		customer:            authpersistence.NewCustomerRepository(),
		membership:          staffpersistence.NewStaffMembershipRepository(),
		role:                staffpersistence.NewRoleRepository(),
		order:               orderpersistence.NewOrderRepository(),
		orderItem:           orderpersistence.NewOrderItemRepository(),
		invoice:             orderpersistence.NewInvoiceRepository(),
		invoiceItem:         orderpersistence.NewInvoiceItemRepository(),
		shipment:            shipmentpersistence.NewShipmentRepository(),
		wishlist:            wishlistpersistence.NewWishlistRepository(),
		review:              reviewpersistence.NewReviewRepository(),
		coupon:              discountpersistence.NewCouponRepository(),
	}
}

type sharedServices struct {
	tokenSvc             authrepo.TokenService
	pwHasher             authrepo.PasswordHasher
	tokenHasher          authrepo.TokenHasher
	authMidd             authrepo.Authenticator
	userDeletionSvc      authrepo.UserDeletionService
	authorMdwr           authrepo.Authorizer
	slugGen              sGen.Generator
	mailSender           mailerSvc.Sender
	otpGen               otpSvc.Generator
	imageTransformer     imgSvc.ImageTransformer
	imageVariantProvider imgSvc.VariantCreator
	pricingService       orderrepo.PricingService
	discountService      *discountusecase.DiscountService
}

func initSharedServices(cfg Config, infra *Dependency, repos *repositories) *sharedServices {
	tokenSvc := authsvc.NewJWTService(cfg.JWT.Secret)
	pwHasher := authsvc.NewBcryptHasher()
	tokenHasher := authsvc.NewSHATokenHasher()
	authMidd := authsvc.NewJWTAuthenticator(
		tokenSvc,
		repos.session,
		tokenHasher,
		repos.refreshToken,
	)

	userDeletionSvc := authsvc.NewUserDeletionService(
		repos.account,
		repos.oauth,
		repos.session,
		repos.user,
	)
	authorMdwr := authsvc.NewAuthorizer()

	slugGen := sGen.NewGenerator()

	mailSender := mailerSvc.NewSMTPSender(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
	)

	otpGen := otpSvc.NewNumericGenerator(6)

	imageTransformer := imgSvc.NewImageTransformer()
	imageVariantProvider := imgSvc.NewResolutionGenerator(imageTransformer)

	discountService := discountusecase.NewDiscountService(
		repos.coupon,
		infra.TransactionExecutor,
		appclock.RealClock{},
	)

	pricingService := ordersvc.NewPricingService(
		repos.address,
		repos.cart,
		repos.courier,
		repos.inventory,
		repos.paymentMethod,
		repos.product,
		infra.ShippingProvider,
		repos.addressShop,
		repos.shop,
	).WithCouponService(discountService)

	return &sharedServices{
		tokenSvc:             tokenSvc,
		pwHasher:             pwHasher,
		tokenHasher:          tokenHasher,
		authMidd:             authMidd,
		userDeletionSvc:      userDeletionSvc,
		authorMdwr:           authorMdwr,
		slugGen:              slugGen,
		mailSender:           mailSender,
		otpGen:               otpGen,
		imageTransformer:     imageTransformer,
		imageVariantProvider: imageVariantProvider,
		pricingService:       pricingService,
		discountService:      discountService,
	}
}

func buildContainer(
	cfg Config,
	infra *Dependency,
	repos *repositories,
	svcs *sharedServices,
	log applogger.Logger,
	auditLogger applogger.AuditLogger,
) *Container {
	var (
		productRepo             = repos.product
		productImageRepo        = repos.productImage
		productPerformanceRepo  = repos.productPerformance
		productStockHistoryRepo = repos.productStockHistory
		inventoryRepo           = repos.inventory
		accountRepo             = repos.account
		challengeRepo           = repos.challenge
		oauthRepo               = repos.oauth
		sessionRepo             = repos.session
		refreshTokenRepo        = repos.refreshToken
		cartRepo                = repos.cart
		userRepo                = repos.user
		addressRepo             = repos.address
		addressShopRepo         = repos.addressShop
		paymentRepo             = repos.payment
		paymentMethodRepo       = repos.paymentMethod
		paymentEventRepo        = repos.paymentEvent
		paymentInstructionRepo  = repos.paymentInstruction
		paymentChannelDataRepo  = repos.paymentChannelData
		paymentWebhookEventRepo = repos.paymentWebhookEvent
		shopRepo                = repos.shop
		courierRepo             = repos.courier
		staffRepo               = repos.staff
		customerRepo            = repos.customer
		membershipRepo          = repos.membership
		roleRepo                = repos.role
		orderRepo               = repos.order
		orderItemRepo           = repos.orderItem
		invoiceRepo             = repos.invoice
		invoiceItemRepo         = repos.invoiceItem
		shipmentRepo            = repos.shipment
		wishlistRepo            = repos.wishlist
		reviewRepo              = repos.review
	)

	var (
		tokenSvc             = svcs.tokenSvc
		pwHasher             = svcs.pwHasher
		tokenHasher          = svcs.tokenHasher
		authMidd             = svcs.authMidd
		userDeletionSvc      = svcs.userDeletionSvc
		authorMdwr           = svcs.authorMdwr
		slugGen              = svcs.slugGen
		mailSender           = svcs.mailSender
		otpGen               = svcs.otpGen
		imageVariantProvider = svcs.imageVariantProvider
		pricingService       = svcs.pricingService
		discountService      = svcs.discountService
	)

	orderPaymentAdapter := newOrderPaymentAdapter(orderRepo, orderItemRepo)
	orderDeliveryAdapter := newOrderDeliveryAdapter(orderRepo)
	inventoryProductChecker := newInventoryProductCheckerAdapter(productRepo)
	inventoryShopChecker := newInventoryShopCheckerAdapter(shopRepo)
	inventoryStockHistory := newInventoryStockHistoryAdapter(productStockHistoryRepo)
	shopProductAdapter := newShopProductAdapter(inventoryRepo, productRepo)
	staffAccountAdapter := newStaffAccountAdapter(accountRepo, sessionRepo)
	userAccountAdapter := newUserAccountAdapter(accountRepo)
	userSessionAdapter := newUserSessionAdapter(sessionRepo)
	userStaffProfileAdapter := newUserStaffProfileAdapter(staffRepo)

	processPaymentWebhook := *paymentusecase.NewProcessPaymentWebhookUsecase(
		paymentRepo,
		paymentEventRepo,
		paymentWebhookEventRepo,
		orderPaymentAdapter,
		inventoryRepo,
		infra.PaymentGateway,
		auditLogger,
		infra.TransactionProvider,
		infra.TransactionExecutor,
	)

	c := &Container{
		Logger:             log,
		AuditLogger:        auditLogger,
		CORSAllowedOrigins: cfg.App.CORSAllowedOrigins,
		Authenticator:      authMidd,
		Authorizer:         authorMdwr,
		DBExecutor:         infra.TransactionExecutor,
		DBTransactor:       infra.TransactionProvider,
		GoogleOAuth:        cfg.GoogleOAuth,
		Cache:              infra.Cache,
		Idempotency:        appmiddleware.NewIdempotencyMiddleware(infra.Cache),
		paymentMethodRepo:  paymentMethodRepo,
		paymentGateway:     infra.PaymentGateway,

		FindProducts: *productusecase.NewFindProductsUsecase(
			productRepo,
			inventoryRepo,
			productImageRepo,
			shopRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		).WithCache(infra.Cache),
		GetProduct: *productusecase.NewGetProductUsecase(
			infra.TransactionExecutor,
			infra.StorageProvider,
			productRepo,
			inventoryRepo,
			productImageRepo,
			shopRepo,
			productPerformanceRepo,
		).WithCache(infra.Cache),
		SaveProduct: *productusecase.NewSaveProductUsecase(
			infra.TransactionProvider,
			productRepo,
			slugGen,
			productPerformanceRepo,
		).WithCache(infra.Cache),
		GetProductStats: *productusecase.NewGetProductStatsUsecase(
			productPerformanceRepo,
			productImageRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		),
		DeleteProduct: *productusecase.NewDeleteProductUsecase(
			productRepo,
			infra.TransactionExecutor,
		).WithCache(infra.Cache),
		AddProductImages: *productusecase.NewAddProductImagesUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			productRepo,
			productImageRepo,
			slugGen,
			imageVariantProvider,
			infra.StorageProvider,
		),
		Inventory: *inventoryusecase.NewInventoryService(
			inventoryRepo,
			inventoryProductChecker,
			inventoryShopChecker,
			infra.TransactionExecutor,
			inventoryStockHistory,
		),

		Me: *authusecase.NewMeUsecase(
			infra.TransactionExecutor,
			accountRepo,
			userRepo,
			oauthRepo,
		),
		Logout: *authusecase.NewLogoutUsecase(
			infra.TransactionProvider,
			refreshTokenRepo,
			sessionRepo,
			auditLogger,
		),
		LoginCustomer: *authusecase.NewLoginCustomerUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			pwHasher,
			tokenHasher,
			tokenSvc,
			sessionRepo,
			refreshTokenRepo,
			customerRepo,
			auditLogger,
		),
		LoginStaff: *authusecase.NewLoginStaffUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			pwHasher,
			tokenHasher,
			tokenSvc,
			sessionRepo,
			refreshTokenRepo,
			staffRepo,
			membershipRepo,
			auditLogger,
		),

		Staff: *staffusecase.NewStaffService(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			staffRepo,
			membershipRepo,
			roleRepo,
			userRepo,
			staffAccountAdapter,
			pwHasher,
			userDeletionSvc,
			auditLogger,
		),

		RegisterCustomer: *authusecase.NewRegisterCustomerUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			pwHasher,
			userRepo,
			customerRepo,
			challengeRepo,
			otpGen,
			mailSender,
			auditLogger,
		),
		VerifyAccount: *authusecase.NewVerifyAccountUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			pwHasher,
			tokenHasher,
			userRepo,
			customerRepo,
			membershipRepo,
			challengeRepo,
			tokenSvc,
			sessionRepo,
			refreshTokenRepo,
			auditLogger,
		),
		GetAccount: *authusecase.NewGetAccountUsecase(
			accountRepo,
			infra.TransactionExecutor,
		),
		AuthenticateOAuth: *authusecase.NewAuthenticateOAuthUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			oauthRepo,
			userRepo,
			customerRepo,
			tokenHasher,
			tokenSvc,
			sessionRepo,
			refreshTokenRepo,
			auditLogger,
		),
		RequestPasswordReset: *authusecase.NewRequestPasswordResetUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			challengeRepo,
			pwHasher,
			otpGen,
			mailSender,
			auditLogger,
		),
		VerifyPasswordReset: *authusecase.NewVerifyPasswordResetUsecase(
			infra.TransactionExecutor,
			challengeRepo,
			pwHasher,
			auditLogger,
		),
		ResetPassword: *authusecase.NewResetPasswordUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			sessionRepo,
			challengeRepo,
			pwHasher,
			auditLogger,
		),
		RefreshToken: *authusecase.NewRefreshTokenUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			tokenSvc,
			tokenHasher,
			sessionRepo,
			refreshTokenRepo,
		),
		DeleteAccount: *authusecase.NewDeleteAccountUsecase(
			infra.TransactionProvider,
			userDeletionSvc,
			customerRepo,
			auditLogger,
		),

		Cart: *cartusecase.NewCartService(
			cartRepo,
			inventoryRepo,
			productRepo,
			productImageRepo,
			shopRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),
		Checkout: *orderusecase.NewCheckoutUsecase(
			infra.TransactionExecutor,
			pricingService,
		),

		User: *userusecase.NewUserService(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			userAccountAdapter,
			userStaffProfileAdapter,
			userSessionAdapter,
			userRepo,
		),

		Address: *addressusecase.NewAddressService(
			addressRepo,
			addressShopRepo,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),

		Shop: *shopusecase.NewShopService(
			shopRepo,
			addressShopRepo,
			shopProductAdapter,
			slugGen,
			infra.TransactionExecutor,
		),

		SavePaymentMethod: *paymentusecase.NewSavePaymentMethodUsecase(
			paymentMethodRepo,
			infra.TransactionExecutor,
		),
		ListPaymentMethod: *paymentusecase.NewListPaymentMethodUsecase(
			paymentMethodRepo,
			infra.TransactionExecutor,
		),
		ProcessPaymentWebhook: processPaymentWebhook,
		SavePaymentInstruction: *paymentusecase.NewSavePaymentInstructionUsecase(
			paymentMethodRepo,
			paymentInstructionRepo,
			infra.TransactionExecutor,
		),
		GetPaymentDetail: *paymentusecase.NewGetPaymentDetailUsecase(
			infra.TransactionExecutor,
			orderPaymentAdapter,
			paymentRepo,
			paymentMethodRepo,
			paymentInstructionRepo,
			paymentChannelDataRepo,
		),
		CheckPaymentStatus: *paymentusecase.NewCheckPaymentStatusUsecase(
			orderPaymentAdapter,
			paymentRepo,
			infra.PaymentGateway,
			&processPaymentWebhook,
			infra.TransactionExecutor,
		),
		SyncPendingPayments: *paymentusecase.NewSyncPendingPaymentsUsecase(
			paymentRepo,
			infra.PaymentGateway,
			&processPaymentWebhook,
			infra.TransactionExecutor,
			log,
			time.Duration(cfg.PaymentSync.LookbackHours)*time.Hour,
			infra.TransactionProvider,
			orderPaymentAdapter,
			inventoryRepo,
		),
		ExpirePastDuePayments: *paymentusecase.NewExpirePastDuePaymentsUsecase(
			paymentRepo,
			infra.PaymentGateway,
			infra.TransactionExecutor,
			infra.TransactionProvider,
			orderPaymentAdapter,
			inventoryRepo,
			log,
			cfg.PaymentExpiry.BatchSize,
			cfg.PaymentExpiry.Concurrency,
		),
		SyncPaymentMethods: *paymentusecase.NewSyncPaymentMethodsUsecase(
			paymentMethodRepo,
			infra.TransactionExecutor,
			infra.PaymentGateway,
		),
		ProcessOrderRefund: *paymentusecase.NewProcessOrderRefundUsecase(
			paymentRepo,
			infra.PaymentGateway,
			infra.TransactionExecutor,
			infra.TransactionProvider,
			log,
		),

		Courier: *courierusecase.NewCourierService(
			infra.TransactionExecutor,
			courierRepo,
		),

		EstimateShippingOptions: *shipmentusecase.NewEstimateShippingOptionsUsecase(
			infra.ShippingProvider,
			infra.TransactionExecutor,
			courierRepo,
		),
		UpdateShipmentStatus: *shipmentusecase.NewUpdateShipmentStatusUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			shipmentRepo,
			orderDeliveryAdapter,
		),
		UpdateShipment: *shipmentusecase.NewUpdateShipmentUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			shipmentRepo,
		),

		CreateOrder: *orderusecase.NewCreateOrderUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			orderRepo,
			orderItemRepo,
			invoiceRepo,
			invoiceItemRepo,
			paymentRepo,
			paymentMethodRepo,
			paymentEventRepo,
			paymentInstructionRepo,
			paymentChannelDataRepo,
			inventoryRepo,
			cartRepo,
			userRepo,
			infra.PaymentGateway,
			pricingService,
		).WithCouponService(discountService),
		FindOrders: *orderusecase.NewFindOrdersUsecase(
			infra.TransactionExecutor,
			orderRepo,
			orderItemRepo,
			paymentRepo,
			paymentChannelDataRepo,
			shipmentRepo,
			addressRepo,
		),
		GetOrder: *orderusecase.NewGetOrderUsecase(
			infra.TransactionExecutor,
			orderRepo,
			orderItemRepo,
			paymentRepo,
			paymentChannelDataRepo,
			shipmentRepo,
		),
		UpdateOrderStatus: *orderusecase.NewUpdateOrderStatusUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			orderRepo,
			orderItemRepo,
			inventoryRepo,
			paymentRepo,
			productRepo,
			shipmentRepo,
			addressRepo,
			addressShopRepo,
			infra.LogisticsProvider,
			auditLogger,
		),
		DispatchShopShipment: *orderusecase.NewDispatchShopShipmentUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			orderRepo,
			orderItemRepo,
			productRepo,
			shipmentRepo,
			addressRepo,
			addressShopRepo,
			infra.LogisticsProvider,
			auditLogger,
		),
		GetOrderTracking: *orderusecase.NewGetOrderTrackingUsecase(
			infra.TransactionExecutor,
			orderRepo,
			shipmentRepo,
		),
		ExpireUnfulfilledOrders: *orderusecase.NewExpireUnfulfilledOrdersUsecase(
			orderRepo,
			orderItemRepo,
			inventoryRepo,
			paymentusecase.NewProcessOrderRefundUsecase(
				paymentRepo,
				infra.PaymentGateway,
				infra.TransactionExecutor,
				infra.TransactionProvider,
				log,
			),
			infra.TransactionExecutor,
			infra.TransactionProvider,
			log,
			auditLogger,
			100,
			5,
		),

		Wishlist: *wishlistusecase.NewWishlistService(
			wishlistRepo,
			productRepo,
			inventoryRepo,
			productImageRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		),

		Review: *reviewusecase.NewReviewService(
			reviewRepo,
			productRepo,
			orderRepo,
			orderItemRepo,
			infra.Cache,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),

		Discount: *discountService,

		Limiter: applimiter.NewInMemorySlidingWindowLimiter(10*time.Second, 30),
	}

	return c
}
