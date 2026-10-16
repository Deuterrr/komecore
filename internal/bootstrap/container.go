package bootstrap

import (
	"time"

	appconfig "komecore/internal/config"
	appmiddleware "komecore/internal/httpx/middleware"
	"komecore/internal/infra/cache"
	"komecore/internal/infra/outbox"
	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	appclock "komecore/pkg/clock"
	imgSvc "komecore/pkg/imageutil"
	applogger "komecore/pkg/logger"
	mailerSvc "komecore/pkg/mailer"
	otpSvc "komecore/pkg/otp"
	applimiter "komecore/pkg/ratelimit"
	sGen "komecore/pkg/slug"

	"komecore/internal/modules/address"
	"komecore/internal/modules/address/addresspersistence"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authcrypto"
	"komecore/internal/modules/auth/authhttp"
	"komecore/internal/modules/auth/authpersistence"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/cart/cartpersistence"
	"komecore/internal/modules/courier"
	"komecore/internal/modules/courier/courierpersistence"
	"komecore/internal/modules/discount"
	"komecore/internal/modules/discount/discountpersistence"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/inventory/inventorypersistence"
	"komecore/internal/modules/order"
	"komecore/internal/modules/order/orderpersistence"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/payment/paymentpersistence"
	"komecore/internal/modules/product"
	"komecore/internal/modules/product/productpersistence"
	"komecore/internal/modules/review"
	"komecore/internal/modules/review/reviewpersistence"
	"komecore/internal/modules/shipment"
	"komecore/internal/modules/shipment/shipmentpersistence"
	"komecore/internal/modules/shop"
	"komecore/internal/modules/shop/shoppersistence"
	"komecore/internal/modules/staff"
	"komecore/internal/modules/staff/staffpersistence"
	"komecore/internal/modules/user"
	"komecore/internal/modules/user/userpersistence"
	"komecore/internal/modules/wishlist"
	"komecore/internal/modules/wishlist/wishlistpersistence"
)

type Container struct {
	Logger             applogger.Logger
	AuditLogger        applogger.AuditLogger
	CORSAllowedOrigins []string
	Authenticator      authhttp.Authenticator
	Authorizer         authhttp.Authorizer
	DBExecutor         transaction.Executor
	DBTransactor       transaction.Transactor
	GoogleOAuth        appconfig.GoogleOAuthConfig
	Cache              cache.Cache
	Idempotency        *appmiddleware.IdempotencyMiddleware
	paymentMethodRepo  payment.MethodRepository
	paymentGateway     paymentgateway.Provider
	OutboxWorker       *outbox.Worker
	OutboxRepo         outbox.Repository

	Auth           *auth.Service
	Catalog        *product.Service
	Inventory      *inventory.Service
	Staff          *staff.Service
	Cart           *cart.Service
	User           *user.Service
	Address        *address.Service
	Payment        *payment.Service
	PaymentWebhook *payment.WebhookProcessor
	Shop           *shop.Service
	Courier        *courier.Service
	Shipment       *shipment.Service
	Order          *order.Service
	CreateOrder    *order.CreateOrderWorkflow
	Wishlist       *wishlist.Service
	Review         *review.Service
	Discount       *discount.Service

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
	product             product.Repository
	productImage        product.ImageRepository
	productPerformance  product.PerformanceRepository
	productStockHistory product.StockHistoryRepository
	inventory           inventory.Repository
	account             auth.AccountRepository
	challenge           auth.VerificationChallengeRepository
	oauth               auth.OAuthConnectionRepository
	session             auth.SessionRepository
	refreshToken        auth.RefreshTokenRepository
	cart                cart.Repository
	user                user.Repository
	address             address.CustomerAddressRepository
	addressShop         address.ShopAddressRepository
	payment             payment.Repository
	paymentMethod       payment.MethodRepository
	paymentEvent        payment.EventRepository
	paymentInstruction  payment.InstructionRepository
	paymentChannelData  payment.ChannelDataRepository
	paymentWebhookEvent payment.WebhookEventRepository
	shop                shop.Repository
	courier             courier.Repository
	staff               staff.Repository
	customer            auth.CustomerRepository
	membership          staff.MembershipRepository
	role                staff.RoleRepository
	order               order.Repository
	orderItem           order.ItemRepository
	invoice             order.InvoiceRepository
	invoiceItem         order.InvoiceItemRepository
	shipment            shipment.Repository
	wishlist            wishlist.Repository
	review              review.Repository
	coupon              discount.Repository
	analytics           staff.AnalyticsRepository
	outbox              outbox.Repository
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
		analytics:           staffpersistence.NewAnalyticsRepository(),
		outbox:              outbox.NewOutboxRepository(),
	}
}

type sharedServices struct {
	tokenSvc             auth.TokenService
	pwHasher             auth.PasswordHasher
	tokenHasher          auth.TokenHasher
	authMidd             authhttp.Authenticator
	userDeletionSvc      auth.UserDeletionService
	authorMdwr           authhttp.Authorizer
	slugGen              sGen.Generator
	mailSender           mailerSvc.Sender
	otpGen               otpSvc.Generator
	imageTransformer     imgSvc.ImageTransformer
	imageVariantProvider imgSvc.VariantCreator
	pricingService       order.PricingService
	discountService      *discount.Service
}

func initSharedServices(cfg Config, infra *Dependency, repos *repositories) *sharedServices {
	tokenSvc := authcrypto.NewJWTService(cfg.JWT.Secret)
	pwHasher := authcrypto.NewBcryptHasher()
	tokenHasher := authcrypto.NewSHATokenHasher()
	authMidd := authhttp.NewJWTAuthenticator(
		tokenSvc,
		repos.session,
		tokenHasher,
		repos.refreshToken,
	)

	userDeletionSvc := auth.NewUserDeletionService(
		repos.account,
		repos.oauth,
		repos.session,
		repos.user,
	)
	authorMdwr := authhttp.NewAuthorizer()

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

	discountService := discount.NewService(
		repos.coupon,
		infra.TransactionExecutor,
		appclock.RealClock{},
	)

	pricingService := order.NewPricingService(
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

	outboxDispatcher := outbox.NewEmailDispatcher(mailSender)
	outboxWorker := outbox.NewWorker(
		repos.outbox,
		outboxDispatcher,
		infra.TransactionExecutor,
		infra.TransactionProvider,
		log,
		5*time.Second,
		20,
	)

	authService := auth.NewService(
		infra.TransactionExecutor,
		infra.TransactionProvider,
		accountRepo,
		sessionRepo,
		refreshTokenRepo,
		challengeRepo,
		oauthRepo,
		customerRepo,
		userRepo,
		staffRepo,
		membershipRepo,
		pwHasher,
		tokenHasher,
		tokenSvc,
		userDeletionSvc,
		otpGen,
		mailSender,
		auditLogger,
	)

	catalogService := product.NewService(
		infra.TransactionExecutor,
		infra.TransactionProvider,
		productRepo,
		inventoryRepo,
		productImageRepo,
		productPerformanceRepo,
		shopRepo,
		slugGen,
		imageVariantProvider,
		infra.StorageProvider,
	).WithCache(infra.Cache)

	inventoryService := inventory.NewService(
		inventoryRepo,
		inventoryProductChecker,
		inventoryShopChecker,
		infra.TransactionExecutor,
		inventoryStockHistory,
	)

	staffService := staff.NewService(
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
	).WithAnalyticsRepository(repos.analytics)

	cartService := cart.NewService(
		cartRepo,
		inventoryRepo,
		productRepo,
		productImageRepo,
		shopRepo,
		infra.StorageProvider,
		infra.TransactionExecutor,
		infra.TransactionProvider,
	)

	userService := user.NewService(
		infra.TransactionExecutor,
		infra.TransactionProvider,
		userAccountAdapter,
		userStaffProfileAdapter,
		userSessionAdapter,
		userRepo,
	)

	addressService := address.NewService(
		addressRepo,
		addressShopRepo,
		infra.TransactionExecutor,
		infra.TransactionProvider,
	)

	processPaymentWebhook := payment.NewWebhookProcessor(
		paymentRepo,
		paymentEventRepo,
		paymentWebhookEventRepo,
		orderPaymentAdapter,
		inventoryRepo,
		infra.PaymentGateway,
		auditLogger,
		infra.TransactionProvider,
		infra.TransactionExecutor,
	).WithOutboxRecorder(repos.outbox)

	paymentService := payment.NewService(
		paymentRepo,
		paymentMethodRepo,
		paymentInstructionRepo,
		paymentChannelDataRepo,
		orderPaymentAdapter,
		inventoryRepo,
		infra.PaymentGateway,
		processPaymentWebhook,
		infra.TransactionExecutor,
		infra.TransactionProvider,
		log,
		time.Duration(cfg.PaymentSync.LookbackHours)*time.Hour,
		cfg.PaymentExpiry.BatchSize,
		cfg.PaymentExpiry.Concurrency,
	).WithOutboxRecorder(repos.outbox)

	shopService := shop.NewService(
		shopRepo,
		addressShopRepo,
		shopProductAdapter,
		slugGen,
		infra.TransactionExecutor,
	)

	courierService := courier.NewService(
		infra.TransactionExecutor,
		courierRepo,
	)

	shipmentService := shipment.NewService(
		infra.TransactionExecutor,
		infra.TransactionProvider,
		shipmentRepo,
		courierRepo,
		infra.ShippingProvider,
		orderDeliveryAdapter,
	)

	orderService := order.NewService(
		infra.TransactionExecutor,
		infra.TransactionProvider,
		orderRepo,
		orderItemRepo,
		inventoryRepo,
		paymentRepo,
		paymentChannelDataRepo,
		productRepo,
		shipmentRepo,
		addressRepo,
		addressShopRepo,
		infra.LogisticsProvider,
		paymentService,
		log,
		auditLogger,
	).WithOutboxRecorder(repos.outbox)

	createOrderWorkflow := order.NewCreateOrderWorkflow(
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
	).WithCouponService(discountService).WithOutboxRecorder(repos.outbox)

	wishlistService := wishlist.NewService(
		wishlistRepo,
		productRepo,
		inventoryRepo,
		productImageRepo,
		infra.StorageProvider,
		infra.TransactionExecutor,
	)

	reviewService := review.NewService(
		reviewRepo,
		productRepo,
		orderRepo,
		orderItemRepo,
		infra.Cache,
		infra.TransactionExecutor,
		infra.TransactionProvider,
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
		OutboxWorker:       outboxWorker,
		OutboxRepo:         repos.outbox,

		Auth:           authService,
		Catalog:        catalogService,
		Inventory:      inventoryService,
		Staff:          staffService,
		Cart:           cartService,
		User:           userService,
		Address:        addressService,
		Payment:        paymentService,
		PaymentWebhook: processPaymentWebhook,
		Shop:           shopService,
		Courier:        courierService,
		Shipment:       shipmentService,
		Order:          orderService,
		CreateOrder:    createOrderWorkflow,
		Wishlist:       wishlistService,
		Review:         reviewService,
		Discount:       discountService,

		Limiter: applimiter.NewInMemorySlidingWindowLimiter(10*time.Second, 30),
	}

	return c
}
