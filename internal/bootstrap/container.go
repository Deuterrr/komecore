package bootstrap

import (
	"time"

	applogger "komecore/pkg/logger"
	applimiter "komecore/pkg/ratelimit"

	appconfig "komecore/internal/config"
	transaction "komecore/internal/infra/transactor"
	imgSvc "komecore/pkg/imageutil"
	mailerSvc "komecore/pkg/mailer"
	otpSvc "komecore/pkg/otp"
	sGen "komecore/pkg/slug"

	addressPersistence "komecore/internal/modules/address/infra/persistence"
	authenPersistence "komecore/internal/modules/auth/infra/persistence"
	cartPersistence "komecore/internal/modules/cart/infra/persistence"
	courierPersistence "komecore/internal/modules/courier/infra/persistence"
	inventoryPersistence "komecore/internal/modules/inventory/infra/persistence"
	orderPersistence "komecore/internal/modules/order/infra/persistence"
	paymentPersistence "komecore/internal/modules/payment/infra/persistence"
	productPersistence "komecore/internal/modules/product/infra/persistence"
	reviewPersistence "komecore/internal/modules/review/infra/persistence"
	shipmentPersistence "komecore/internal/modules/shipment/infra/persistence"
	shopPersistence "komecore/internal/modules/shop/infra/persistence"
	staffPersistence "komecore/internal/modules/staff/infra/persistence"
	userPersistence "komecore/internal/modules/user/infra/persistence"
	wishlistPersistence "komecore/internal/modules/wishlist/infra/persistence"

	authenSvc "komecore/internal/modules/auth/infra/service"
	authenRepo "komecore/internal/modules/auth/repository"
	orderSvc "komecore/internal/modules/order/service"

	addressRepo "komecore/internal/modules/address/repository"
	cartRepo "komecore/internal/modules/cart/repository"
	courierRepo "komecore/internal/modules/courier/repository"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	orderRepo "komecore/internal/modules/order/repository"
	paymentRepo "komecore/internal/modules/payment/repository"
	productRepo "komecore/internal/modules/product/repository"
	reviewRepo "komecore/internal/modules/review/repository"
	shipmentRepo "komecore/internal/modules/shipment/repository"
	shopRepo "komecore/internal/modules/shop/repository"
	staffRepo "komecore/internal/modules/staff/repository"
	userRepo "komecore/internal/modules/user/repository"
	wishlistRepo "komecore/internal/modules/wishlist/repository"

	addressUsecase "komecore/internal/modules/address/usecase"
	authenUsecase "komecore/internal/modules/auth/usecase"
	cartUsecase "komecore/internal/modules/cart/usecase"
	courierUsecase "komecore/internal/modules/courier/usecase"
	inventoryUsecase "komecore/internal/modules/inventory/usecase"
	orderUsecase "komecore/internal/modules/order/usecase"
	paymentUsecase "komecore/internal/modules/payment/usecase"
	productUsecase "komecore/internal/modules/product/usecase"
	reviewUsecase "komecore/internal/modules/review/usecase"
	shipmentUsecase "komecore/internal/modules/shipment/usecase"
	shopUsecase "komecore/internal/modules/shop/usecase"
	staffUsecase "komecore/internal/modules/staff/usecase"
	userUsecase "komecore/internal/modules/user/usecase"
	wishlistUsecase "komecore/internal/modules/wishlist/usecase"

	appmiddleware "komecore/internal/common/middleware"
	"komecore/internal/infra/cache"
	paymentgateway "komecore/internal/infra/payment-gateway"
)

type Container struct {
	Logger             applogger.Logger
	AuditLogger        applogger.AuditLogger
	CORSAllowedOrigins []string
	Authenticator      authenRepo.Authenticator
	Authorizer         authenRepo.Authorizer
	DBExecutor         transaction.Executor
	DBTransactor       transaction.Transactor
	GoogleOAuth        appconfig.GoogleOAuthConfig
	Cache              cache.Cache
	Idempotency        *appmiddleware.IdempotencyMiddleware
	paymentMethodRepo  paymentRepo.PaymentMethodRepository
	paymentGateway     paymentgateway.Provider

	FindProducts     productUsecase.FindProductsUsecase
	GetProduct       productUsecase.GetProductUsecase
	SaveProduct      productUsecase.SaveProductUsecase
	DeleteProduct    productUsecase.DeleteProductUsecase
	AddProductImages productUsecase.AddProductImagesUsecase
	GetProductStats  productUsecase.GetProductStatsUsecase
	Inventory inventoryUsecase.InventoryService

	Me                   authenUsecase.MeUsecase
	LoginCustomer        authenUsecase.LoginCustomerUsecase
	LoginStaff           authenUsecase.LoginStaffUsecase
	RegisterCustomer     authenUsecase.RegisterCustomerUsecase
	VerifyAccount        authenUsecase.VerifyAccountUsecase
	GetAccount           authenUsecase.GetAccountUsecase
	Logout               authenUsecase.LogoutUsecase
	AuthenticateOAuth    authenUsecase.AuthenticateOAuthUsecase
	RequestPasswordReset authenUsecase.RequestPasswordResetUsecase
	VerifyPasswordReset  authenUsecase.VerifyPasswordResetUsecase
	ResetPassword        authenUsecase.ResetPasswordUsecase
	RefreshToken         authenUsecase.RefreshTokenUsecase
	DeleteAccount        authenUsecase.DeleteAccountUsecase

	Staff staffUsecase.StaffService

	Cart     cartUsecase.CartService
	Checkout orderUsecase.CheckoutUsecase

	User userUsecase.UserService

	Address addressUsecase.AddressService

	Shop shopUsecase.ShopService

	SavePaymentMethod      paymentUsecase.SavePaymentMethodUsecase
	ListPaymentMethod      paymentUsecase.ListPaymentMethodUsecase
	ProcessPaymentWebhook  paymentUsecase.ProcessPaymentWebhookUsecase
	SavePaymentInstruction paymentUsecase.SavePaymentInstructionUsecase
	GetPaymentDetail       paymentUsecase.GetPaymentDetailUsecase
	CheckPaymentStatus     paymentUsecase.CheckPaymentStatusUsecase
	SyncPendingPayments    paymentUsecase.SyncPendingPaymentsUsecase
	ExpirePastDuePayments  paymentUsecase.ExpirePastDuePaymentsUsecase
	SyncPaymentMethods     paymentUsecase.SyncPaymentMethodsUsecase
	ProcessOrderRefund     paymentUsecase.ProcessOrderRefundUsecase

	Courier courierUsecase.CourierService

	EstimateShippingOptions shipmentUsecase.EstimateShippingOptionsUsecase
	UpdateShipmentStatus    shipmentUsecase.UpdateShipmentStatusUsecase
	UpdateShipment          shipmentUsecase.UpdateShipmentUsecase

	CreateOrder             orderUsecase.CreateOrderUsecase
	FindOrders              orderUsecase.FindOrdersUsecase
	GetOrder                orderUsecase.GetOrderUsecase
	UpdateOrderStatus       orderUsecase.UpdateOrderStatusUsecase
	DispatchShopShipment    orderUsecase.DispatchShopShipmentUsecase
	GetOrderTracking        orderUsecase.GetOrderTrackingUsecase
	ExpireUnfulfilledOrders orderUsecase.ExpireUnfulfilledOrdersUsecase

	Wishlist wishlistUsecase.WishlistService
	Review   reviewUsecase.ReviewService

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
	product             productRepo.ProductRepository
	productImage        productRepo.ProductImageRepository
	productPerformance  productRepo.ProductPerformanceRepository
	productStockHistory productRepo.ProductStockHistoryRepository
	inventory           inventoryRepo.InventoryRepository
	account             authenRepo.AccountRepository
	challenge           authenRepo.VerificationChallengeRepository
	oauth               authenRepo.OAuthConnectionRepository
	session             authenRepo.SessionRepository
	refreshToken        authenRepo.RefreshTokenRepository
	cart                cartRepo.CartRepository
	user                userRepo.UserRepository
	address             addressRepo.CustomerAddressRepository
	addressShop         addressRepo.ShopAddressRepository
	payment             paymentRepo.PaymentRepository
	paymentMethod       paymentRepo.PaymentMethodRepository
	paymentEvent        paymentRepo.PaymentEventRepository
	paymentInstruction  paymentRepo.PaymentInstructionRepository
	paymentChannelData  paymentRepo.PaymentChannelDataRepository
	paymentWebhookEvent paymentRepo.PaymentWebhookEventRepository
	shop                shopRepo.ShopRepository
	courier             courierRepo.CourierRepository
	staff               staffRepo.StaffRepository
	customer            authenRepo.CustomerRepository
	membership          staffRepo.StaffMembershipRepository
	role                staffRepo.RoleRepository
	order               orderRepo.OrderRepository
	orderItem           orderRepo.OrderItemRepository
	invoice             orderRepo.InvoiceRepository
	invoiceItem         orderRepo.InvoiceItemRepository
	shipment            shipmentRepo.ShipmentRepository
	wishlist            wishlistRepo.WishlistRepository
	review              reviewRepo.ReviewRepository
}

func initRepositories() *repositories {
	return &repositories{
		product:             productPersistence.NewProductRepository(),
		productImage:        productPersistence.NewProductImageRepository(),
		productPerformance:  productPersistence.NewProductPerformanceRepository(),
		productStockHistory: productPersistence.NewProductStockHistoryRepository(),
		inventory:           inventoryPersistence.NewInventoryRepository(),
		account:             authenPersistence.NewAccountRepository(),
		challenge:           authenPersistence.NewChallengeRepository(),
		oauth:               authenPersistence.NewOAuthConnectionRepository(),
		session:             authenPersistence.NewSessionRepository(),
		refreshToken:        authenPersistence.NewRefreshTokenRepository(),
		cart:                cartPersistence.NewCartRepository(),
		user:                userPersistence.NewUserRepository(),
		address:             addressPersistence.NewCustomerAddressRepository(),
		addressShop:         addressPersistence.NewShopAddressRepository(),
		payment:             paymentPersistence.NewPaymentRepository(),
		paymentMethod:       paymentPersistence.NewPaymentMethodRepository(),
		paymentEvent:        paymentPersistence.NewPaymentEventRepository(),
		paymentInstruction:  paymentPersistence.NewPaymentInstructionRepository(),
		paymentChannelData:  paymentPersistence.NewPaymentChannelDataRepository(),
		paymentWebhookEvent: paymentPersistence.NewPaymentWebhookEventRepository(),
		shop:                shopPersistence.NewShopRepository(),
		courier:             courierPersistence.NewCourierRepository(),
		staff:               staffPersistence.NewStaffRepository(),
		customer:            authenPersistence.NewCustomerRepository(),
		membership:          staffPersistence.NewStaffMembershipRepository(),
		role:                staffPersistence.NewRoleRepository(),
		order:               orderPersistence.NewOrderRepository(),
		orderItem:           orderPersistence.NewOrderItemRepository(),
		invoice:             orderPersistence.NewInvoiceRepository(),
		invoiceItem:         orderPersistence.NewInvoiceItemRepository(),
		shipment:            shipmentPersistence.NewShipmentRepository(),
		wishlist:            wishlistPersistence.NewWishlistRepository(),
		review:              reviewPersistence.NewReviewRepository(),
	}
}

type sharedServices struct {
	tokenSvc             authenRepo.TokenService
	pwHasher             authenRepo.PasswordHasher
	tokenHasher          authenRepo.TokenHasher
	authMidd             authenRepo.Authenticator
	userDeletionSvc      authenRepo.UserDeletionService
	authorMdwr           authenRepo.Authorizer
	slugGen              sGen.Generator
	mailSender           mailerSvc.Sender
	otpGen               otpSvc.Generator
	imageTransformer     imgSvc.ImageTransformer
	imageVariantProvider imgSvc.VariantCreator
	pricingService       orderRepo.PricingService
}

func initSharedServices(cfg Config, infra *Dependency, repos *repositories) *sharedServices {
	tokenSvc := authenSvc.NewJWTService(cfg.JWT.Secret)
	pwHasher := authenSvc.NewBcryptHasher()
	tokenHasher := authenSvc.NewSHATokenHasher()
	authMidd := authenSvc.NewJWTAuthenticator(
		tokenSvc,
		repos.session,
		tokenHasher,
		repos.refreshToken,
	)

	userDeletionSvc := authenSvc.NewUserDeletionService(
		repos.account,
		repos.oauth,
		repos.session,
		repos.user,
	)
	authorMdwr := authenSvc.NewAuthorizer()

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

	pricingService := orderSvc.NewPricingService(
		repos.address,
		repos.cart,
		repos.courier,
		repos.inventory,
		repos.paymentMethod,
		repos.product,
		infra.ShippingProvider,
		repos.addressShop,
		repos.shop,
	)

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

	processPaymentWebhook := *paymentUsecase.NewProcessPaymentWebhookUsecase(
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

		FindProducts: *productUsecase.NewFindProductsUsecase(
			productRepo,
			inventoryRepo,
			productImageRepo,
			shopRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		).WithCache(infra.Cache),
		GetProduct: *productUsecase.NewGetProductUsecase(
			infra.TransactionExecutor,
			infra.StorageProvider,
			productRepo,
			inventoryRepo,
			productImageRepo,
			shopRepo,
			productPerformanceRepo,
		).WithCache(infra.Cache),
		SaveProduct: *productUsecase.NewSaveProductUsecase(
			infra.TransactionProvider,
			productRepo,
			slugGen,
			productPerformanceRepo,
		).WithCache(infra.Cache),
		GetProductStats: *productUsecase.NewGetProductStatsUsecase(
			productPerformanceRepo,
			productImageRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		),
		DeleteProduct: *productUsecase.NewDeleteProductUsecase(
			productRepo,
			infra.TransactionExecutor,
		).WithCache(infra.Cache),
		AddProductImages: *productUsecase.NewAddProductImagesUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			productRepo,
			productImageRepo,
			slugGen,
			imageVariantProvider,
			infra.StorageProvider,
		),
		Inventory: *inventoryUsecase.NewInventoryService(
			inventoryRepo,
			inventoryProductChecker,
			inventoryShopChecker,
			infra.TransactionExecutor,
			inventoryStockHistory,
		),

		Me: *authenUsecase.NewMeUsecase(
			infra.TransactionExecutor,
			accountRepo,
			userRepo,
			oauthRepo,
		),
		Logout: *authenUsecase.NewLogoutUsecase(
			infra.TransactionProvider,
			refreshTokenRepo,
			sessionRepo,
			auditLogger,
		),
		LoginCustomer: *authenUsecase.NewLoginCustomerUsecase(
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
		LoginStaff: *authenUsecase.NewLoginStaffUsecase(
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

		Staff: *staffUsecase.NewStaffService(
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

		RegisterCustomer: *authenUsecase.NewRegisterCustomerUsecase(
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
		VerifyAccount: *authenUsecase.NewVerifyAccountUsecase(
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
		GetAccount: *authenUsecase.NewGetAccountUsecase(
			accountRepo,
			infra.TransactionExecutor,
		),
		AuthenticateOAuth: *authenUsecase.NewAuthenticateOAuthUsecase(
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
		RequestPasswordReset: *authenUsecase.NewRequestPasswordResetUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			challengeRepo,
			pwHasher,
			otpGen,
			mailSender,
			auditLogger,
		),
		VerifyPasswordReset: *authenUsecase.NewVerifyPasswordResetUsecase(
			infra.TransactionExecutor,
			challengeRepo,
			pwHasher,
			auditLogger,
		),
		ResetPassword: *authenUsecase.NewResetPasswordUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			sessionRepo,
			challengeRepo,
			pwHasher,
			auditLogger,
		),
		RefreshToken: *authenUsecase.NewRefreshTokenUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			tokenSvc,
			tokenHasher,
			sessionRepo,
			refreshTokenRepo,
		),
		DeleteAccount: *authenUsecase.NewDeleteAccountUsecase(
			infra.TransactionProvider,
			userDeletionSvc,
			customerRepo,
			auditLogger,
		),

		Cart: *cartUsecase.NewCartService(
			cartRepo,
			inventoryRepo,
			productRepo,
			productImageRepo,
			shopRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),
		Checkout: *orderUsecase.NewCheckoutUsecase(
			infra.TransactionExecutor,
			pricingService,
		),

		User: *userUsecase.NewUserService(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			userAccountAdapter,
			userStaffProfileAdapter,
			userSessionAdapter,
			userRepo,
		),

		Address: *addressUsecase.NewAddressService(
			addressRepo,
			addressShopRepo,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),

		Shop: *shopUsecase.NewShopService(
			shopRepo,
			addressShopRepo,
			shopProductAdapter,
			slugGen,
			infra.TransactionExecutor,
		),

		SavePaymentMethod: *paymentUsecase.NewSavePaymentMethodUsecase(
			paymentMethodRepo,
			infra.TransactionExecutor,
		),
		ListPaymentMethod: *paymentUsecase.NewListPaymentMethodUsecase(
			paymentMethodRepo,
			infra.TransactionExecutor,
		),
		ProcessPaymentWebhook: processPaymentWebhook,
		SavePaymentInstruction: *paymentUsecase.NewSavePaymentInstructionUsecase(
			paymentMethodRepo,
			paymentInstructionRepo,
			infra.TransactionExecutor,
		),
		GetPaymentDetail: *paymentUsecase.NewGetPaymentDetailUsecase(
			infra.TransactionExecutor,
			orderPaymentAdapter,
			paymentRepo,
			paymentMethodRepo,
			paymentInstructionRepo,
			paymentChannelDataRepo,
		),
		CheckPaymentStatus: *paymentUsecase.NewCheckPaymentStatusUsecase(
			orderPaymentAdapter,
			paymentRepo,
			infra.PaymentGateway,
			&processPaymentWebhook,
			infra.TransactionExecutor,
		),
		SyncPendingPayments: *paymentUsecase.NewSyncPendingPaymentsUsecase(
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
		ExpirePastDuePayments: *paymentUsecase.NewExpirePastDuePaymentsUsecase(
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
		SyncPaymentMethods: *paymentUsecase.NewSyncPaymentMethodsUsecase(
			paymentMethodRepo,
			infra.TransactionExecutor,
			infra.PaymentGateway,
		),
		ProcessOrderRefund: *paymentUsecase.NewProcessOrderRefundUsecase(
			paymentRepo,
			infra.PaymentGateway,
			infra.TransactionExecutor,
			infra.TransactionProvider,
			log,
		),

		Courier: *courierUsecase.NewCourierService(
			infra.TransactionExecutor,
			courierRepo,
		),

		EstimateShippingOptions: *shipmentUsecase.NewEstimateShippingOptionsUsecase(
			infra.ShippingProvider,
			infra.TransactionExecutor,
			courierRepo,
		),
		UpdateShipmentStatus: *shipmentUsecase.NewUpdateShipmentStatusUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			shipmentRepo,
			orderDeliveryAdapter,
		),
		UpdateShipment: *shipmentUsecase.NewUpdateShipmentUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			shipmentRepo,
		),

		CreateOrder: *orderUsecase.NewCreateOrderUsecase(
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
		),
		FindOrders: *orderUsecase.NewFindOrdersUsecase(
			infra.TransactionExecutor,
			orderRepo,
			orderItemRepo,
			paymentRepo,
			paymentChannelDataRepo,
			shipmentRepo,
			addressRepo,
		),
		GetOrder: *orderUsecase.NewGetOrderUsecase(
			infra.TransactionExecutor,
			orderRepo,
			orderItemRepo,
			paymentRepo,
			paymentChannelDataRepo,
			shipmentRepo,
		),
		UpdateOrderStatus: *orderUsecase.NewUpdateOrderStatusUsecase(
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
		DispatchShopShipment: *orderUsecase.NewDispatchShopShipmentUsecase(
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
		GetOrderTracking: *orderUsecase.NewGetOrderTrackingUsecase(
			infra.TransactionExecutor,
			orderRepo,
			shipmentRepo,
		),
		ExpireUnfulfilledOrders: *orderUsecase.NewExpireUnfulfilledOrdersUsecase(
			orderRepo,
			orderItemRepo,
			inventoryRepo,
			paymentUsecase.NewProcessOrderRefundUsecase(
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

		Wishlist: *wishlistUsecase.NewWishlistService(
			wishlistRepo,
			productRepo,
			inventoryRepo,
			productImageRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		),

		Review: *reviewUsecase.NewReviewService(
			reviewRepo,
			productRepo,
			orderRepo,
			orderItemRepo,
			infra.Cache,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),

		Limiter: applimiter.NewInMemorySlidingWindowLimiter(10*time.Second, 30),
	}

	return c
}
