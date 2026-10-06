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
	orderSvc "komecore/internal/modules/order/infra/service"

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
	CreateInventory  inventoryUsecase.CreateInventoryUsecase
	UpdateInventory  inventoryUsecase.UpdateInventoryUsecase
	DeleteInventory  inventoryUsecase.DeleteInventoryUsecase

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

	FindStaff          staffUsecase.FindStaffUsecase
	CreateStaff        staffUsecase.CreateStaffUsecase
	AddStaffAccount    staffUsecase.AddStaffAccountUsecase
	ListStaffAccounts  staffUsecase.ListStaffAccountsUsecase
	UpdateStaff        staffUsecase.UpdateStaffUsecase
	DeleteStaff        staffUsecase.DeleteStaffUsecase
	RemoveStaffAccount staffUsecase.RemoveStaffAccountUsecase

	GetCart    cartUsecase.GetCartUsecase
	AddItem    cartUsecase.AddItemUsecase
	UpdateItem cartUsecase.UpdateItemUsecase
	RemoveItem cartUsecase.RemoveItemUsecase
	Checkout   cartUsecase.CheckoutUsecase

	GetUser              userUsecase.GetUserUsecase
	GetCurrentProfile    userUsecase.GetCurrentProfileUsecase
	UpdateCurrentProfile userUsecase.UpdateCurrentProfileUsecase

	ListUserAddresses addressUsecase.ListCustomerAddressesUsecase
	CreateUserAddress addressUsecase.SaveCustomerAddressUsecase
	DeleteUserAddress addressUsecase.DeleteCustomerAddressUsecase

	ListShopAddresses addressUsecase.ListShopAddressesUsecase
	SaveShopAddress   addressUsecase.CreateShopAddressUsecase
	UpdateShopAddress addressUsecase.UpdateShopAddressUsecase
	DeleteShopAddress addressUsecase.DeleteShopAddressUsecase

	FindShops  shopUsecase.FindShopsUsecase
	GetShop    shopUsecase.GetShopUsecase
	SaveShop   shopUsecase.SaveShopUsecase
	DeleteShop shopUsecase.DeleteShopUsecase

	GetShopAddresses shopUsecase.GetShopAddressesUsecase
	GetShopProducts  shopUsecase.GetShopProductsUsecase

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

	ListAllCouriers courierUsecase.ListCouriersUsecase

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

	GetWishlist        wishlistUsecase.GetWishlistUsecase
	AddToWishlist      wishlistUsecase.AddToWishlistUsecase
	RemoveFromWishlist wishlistUsecase.RemoveFromWishlistUsecase

	CreateReview reviewUsecase.CreateReviewUsecase
	ListReviews  reviewUsecase.ListReviewsUsecase
	DeleteReview reviewUsecase.DeleteReviewUsecase

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
		session:             authenPersistence.NewSessionRepositoryImpl(),
		refreshToken:        authenPersistence.NewRefreshTokenRepositoryImpl(),
		cart:                cartPersistence.NewCartRepositoryImpl(),
		user:                userPersistence.NewUserRepositoryImpl(),
		address:             addressPersistence.NewCustomerAddressRepositoryImpl(),
		addressShop:         addressPersistence.NewShopAddressRepositoryImpl(),
		payment:             paymentPersistence.NewPaymentRepositoryImpl(),
		paymentMethod:       paymentPersistence.NewPaymentMethodRepository(),
		paymentEvent:        paymentPersistence.NewPaymentEventRepositoryImpl(),
		paymentInstruction:  paymentPersistence.NewPaymentInstructionRepositoryImpl(),
		paymentChannelData:  paymentPersistence.NewPaymentChannelDataRepositoryImpl(),
		paymentWebhookEvent: paymentPersistence.NewPaymentWebhookEventRepositoryImpl(),
		shop:                shopPersistence.NewShopRepositoryImpl(),
		courier:             courierPersistence.NewCourierRepositoryImpl(),
		staff:               staffPersistence.NewStaffRepositoryImpl(),
		customer:            authenPersistence.NewCustomerRepositoryImpl(),
		membership:          staffPersistence.NewStaffMembershipRepositoryImpl(),
		role:                staffPersistence.NewRoleRepositoryImpl(),
		order:               orderPersistence.NewOrderRepositoryImpl(),
		orderItem:           orderPersistence.NewOrderItemRepositoryImpl(),
		invoice:             orderPersistence.NewInvoiceRepositoryImpl(),
		invoiceItem:         orderPersistence.NewInvoiceItemRepositoryImpl(),
		shipment:            shipmentPersistence.NewShipmentRepositoryImpl(),
		wishlist:            wishlistPersistence.NewWishlistRepositoryImpl(),
		review:              reviewPersistence.NewReviewRepositoryImpl(),
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

	processPaymentWebhook := *paymentUsecase.NewProcessPaymentWebhookUsecase(
		paymentRepo,
		paymentEventRepo,
		paymentWebhookEventRepo,
		orderRepo,
		orderItemRepo,
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
		CreateInventory: *inventoryUsecase.NewCreateInventoryUsecase(inventoryRepo,
			productRepo,
			shopRepo,
			infra.TransactionExecutor,
			productStockHistoryRepo,
		),
		UpdateInventory: *inventoryUsecase.NewUpdateInventoryUsecase(inventoryRepo,
			infra.TransactionExecutor,
			productStockHistoryRepo,
		),
		DeleteInventory: *inventoryUsecase.NewDeleteInventoryUsecase(inventoryRepo,
			infra.TransactionExecutor,
			productStockHistoryRepo,
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

		FindStaff: *staffUsecase.NewFindStaffUsecase(
			infra.TransactionExecutor,
			staffRepo,
		),
		CreateStaff: *staffUsecase.NewCreateStaffUsecase(
			staffRepo,
			userRepo,
			infra.TransactionExecutor,
			infra.TransactionProvider,
			auditLogger,
		),
		AddStaffAccount: *staffUsecase.NewAddStaffAccountUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			pwHasher,
			userRepo,
			staffRepo,
			membershipRepo,
			roleRepo,
			auditLogger,
		),
		ListStaffAccounts: *staffUsecase.NewListStaffAccountsUsecase(
			infra.TransactionExecutor,
			staffRepo,
			membershipRepo,
			auditLogger,
		),
		UpdateStaff: *staffUsecase.NewUpdateStaffUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			staffRepo,
			membershipRepo,
			auditLogger,
		),
		DeleteStaff: *staffUsecase.NewDeleteStaffUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			staffRepo,
			membershipRepo,
			userDeletionSvc,
			auditLogger,
		),
		RemoveStaffAccount: *staffUsecase.NewRemoveStaffAccountUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			staffRepo,
			membershipRepo,
			accountRepo,
			sessionRepo,
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

		GetCart: *cartUsecase.NewGetCartUsecase(
			cartRepo,
			inventoryRepo,
			productRepo,
			productImageRepo,
			shopRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		),
		AddItem: *cartUsecase.NewAddItemUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			cartRepo,
			inventoryRepo,
			productRepo,
			shopRepo,
		),
		UpdateItem: *cartUsecase.NewUpdateItemUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			cartRepo,
			inventoryRepo,
			productRepo,
		),
		RemoveItem: *cartUsecase.NewRemoveItemUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			cartRepo,
		),
		Checkout: *cartUsecase.NewCheckoutUsecase(
			infra.TransactionExecutor,
			pricingService,
		),

		GetUser: *userUsecase.NewGetUserUsecase(
			userRepo,
			infra.TransactionExecutor,
		),
		GetCurrentProfile: *userUsecase.NewGetCurrentProfileUsecase(
			infra.TransactionExecutor,
			accountRepo,
			userRepo,
			staffRepo,
			sessionRepo,
		),
		UpdateCurrentProfile: *userUsecase.NewUpdateCurrentProfileUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			accountRepo,
			staffRepo,
			userRepo,
		),

		ListUserAddresses: *addressUsecase.NewListCustomerAddressesUsecase(
			addressRepo,
			infra.TransactionExecutor,
		),
		CreateUserAddress: *addressUsecase.NewSaveCustomerAddressUsecase(
			infra.TransactionExecutor,
			infra.TransactionProvider,
			addressRepo,
		),
		DeleteUserAddress: *addressUsecase.NewDeleteCustomerAddressUsecase(
			infra.TransactionExecutor,
			addressRepo,
		),

		ListShopAddresses: *addressUsecase.NewListShopAddressesUsecase(
			addressShopRepo,
			infra.TransactionExecutor,
		),
		SaveShopAddress: *addressUsecase.NewCreateShopAddressUsecase(
			addressShopRepo,
			infra.TransactionExecutor,
		),
		UpdateShopAddress: *addressUsecase.NewUpdateShopAddressUsecase(
			addressShopRepo,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),
		DeleteShopAddress: *addressUsecase.NewDeleteShopAddressUsecase(
			addressShopRepo,
			infra.TransactionExecutor,
		),

		FindShops: *shopUsecase.NewFindShopsUsecase(
			infra.TransactionExecutor,
			shopRepo,
		),
		GetShop: *shopUsecase.NewGetShopUsecase(
			shopRepo,
			infra.TransactionExecutor,
		),
		SaveShop: *shopUsecase.NewSaveShopUsecase(
			shopRepo,
			slugGen,
			infra.TransactionExecutor,
		),
		DeleteShop: *shopUsecase.NewDeleteShopUsecase(
			shopRepo,
			infra.TransactionExecutor,
		),

		GetShopAddresses: *shopUsecase.NewGetShopAddressesUsecase(
			addressShopRepo,
			infra.TransactionExecutor,
		),
		GetShopProducts: *shopUsecase.NewGetShopProductsUsecase(
			inventoryRepo,
			productRepo,
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
			orderRepo,
			invoiceRepo,
			paymentRepo,
			paymentMethodRepo,
			paymentInstructionRepo,
			paymentChannelDataRepo,
		),
		CheckPaymentStatus: *paymentUsecase.NewCheckPaymentStatusUsecase(
			orderRepo,
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
			orderRepo,
			orderItemRepo,
			inventoryRepo,
		),
		ExpirePastDuePayments: *paymentUsecase.NewExpirePastDuePaymentsUsecase(
			paymentRepo,
			infra.PaymentGateway,
			infra.TransactionExecutor,
			infra.TransactionProvider,
			orderRepo,
			orderItemRepo,
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

		ListAllCouriers: *courierUsecase.NewListCouriersUsecase(
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
			orderRepo,
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

		GetWishlist: *wishlistUsecase.NewGetWishlistUsecase(
			wishlistRepo,
			productRepo,
			inventoryRepo,
			productImageRepo,
			infra.StorageProvider,
			infra.TransactionExecutor,
		),
		AddToWishlist: *wishlistUsecase.NewAddToWishlistUsecase(
			wishlistRepo,
			productRepo,
			infra.TransactionExecutor,
		),
		RemoveFromWishlist: *wishlistUsecase.NewRemoveFromWishlistUsecase(
			wishlistRepo,
			infra.TransactionExecutor,
		),

		CreateReview: *reviewUsecase.NewCreateReviewUsecase(
			reviewRepo,
			productRepo,
			orderRepo,
			orderItemRepo,
			infra.Cache,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),
		ListReviews: *reviewUsecase.NewListReviewsUsecase(
			reviewRepo,
			productRepo,
			infra.TransactionExecutor,
		),
		DeleteReview: *reviewUsecase.NewDeleteReviewUsecase(
			reviewRepo,
			productRepo,
			infra.Cache,
			infra.TransactionExecutor,
			infra.TransactionProvider,
		),

		Limiter: applimiter.NewInMemorySlidingWindowLimiter(10*time.Second, 30),
	}

	return c
}
