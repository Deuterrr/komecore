package bootstrap

import (
	"net/http"

	apphttp "komecore/internal/common/http"
	appcookie "komecore/internal/common/http/cookie"
	appmiddleware "komecore/internal/common/middleware"
	"komecore/internal/infra/cache"

	"komecore/internal/common/authctx"
	authenRepo "komecore/internal/modules/auth/repository"

	addressH "komecore/internal/modules/address/delivery/http"
	authH "komecore/internal/modules/auth/delivery/http"
	cartH "komecore/internal/modules/cart/delivery/http"
	courierH "komecore/internal/modules/courier/delivery/http"
	inventoryH "komecore/internal/modules/inventory/delivery/http"
	orderH "komecore/internal/modules/order/delivery/http"
	paymentH "komecore/internal/modules/payment/delivery/http"
	productH "komecore/internal/modules/product/delivery/http"
	reviewH "komecore/internal/modules/review/delivery/http"
	shipmentH "komecore/internal/modules/shipment/delivery/http"
	shopH "komecore/internal/modules/shop/delivery/http"
	staffH "komecore/internal/modules/staff/delivery/http"
	userH "komecore/internal/modules/user/delivery/http"
	wishlistH "komecore/internal/modules/wishlist/delivery/http"

	"github.com/go-chi/chi/v5"
)

// RouteChains encapsulates all pre-built middleware chains for
// different routing policies.
type RouteChains struct {
	Core                    func(apphttp.AppHandler) http.HandlerFunc
	CoreAuth                func(apphttp.AppHandler) http.HandlerFunc
	StaffOnly               func(apphttp.AppHandler) http.HandlerFunc
	StaffAdminOnly          func(apphttp.AppHandler) http.HandlerFunc
	CustomerOnly            func(apphttp.AppHandler) http.HandlerFunc
	CustomerWithIdempotency func(apphttp.AppHandler) http.HandlerFunc
}

// NewRouteChains builds and returns the route chains using
// the provided Container.
func NewRouteChains(c *Container) *RouteChains {
	idempotencyMw := c.Idempotency
	if idempotencyMw == nil {
		idempotencyMw = appmiddleware.NewIdempotencyMiddleware(cache.NewNoopCache())
	}

	buildChain := func(extra ...appmiddleware.Middleware) func(apphttp.AppHandler) http.HandlerFunc {
		base := []appmiddleware.Middleware{
			appmiddleware.CORS(c.CORSAllowedOrigins),
			appmiddleware.Recovery(c.Logger),
			appmiddleware.Logging(c.Logger),
			appmiddleware.Response(),
			appmiddleware.RateLimit(c.Limiter, c.Logger),
		}

		mws := append(base, extra...)

		return func(h apphttp.AppHandler) http.HandlerFunc {
			return appmiddleware.Chain(h, mws...)
		}
	}

	return &RouteChains{
		Core: buildChain(
			c.Authenticator.OptionalAuth(
				c.DBExecutor,
				c.DBTransactor,
				appcookie.CookieCustomer,
				appcookie.CookieStaff,
			),
			c.Authorizer.OptionalLoadActor(c.DBExecutor),
		),

		CoreAuth: buildChain(
			c.Authenticator.RequireMultiAuth(
				c.DBExecutor,
				c.DBTransactor,
				appcookie.CookieCustomer,
				appcookie.CookieStaff,
			),
			c.Authorizer.RequireAccountType(
				authctx.AccountTypeStaff,
				authctx.AccountTypeCustomer,
			),
			c.Authorizer.LoadActor(c.DBExecutor),
		),
		StaffOnly: buildChain(
			c.Authenticator.RequireAuth(
				c.DBExecutor,
				c.DBTransactor,
				appcookie.CookieStaff,
			),
			c.Authorizer.RequireAccountType(authctx.AccountTypeStaff),
			c.Authorizer.LoadActor(c.DBExecutor),
			c.Authorizer.RequireStaffRole(authctx.RoleStaff, authctx.RoleStaffAdmin),
		),
		StaffAdminOnly: buildChain(
			c.Authenticator.RequireAuth(
				c.DBExecutor,
				c.DBTransactor,
				appcookie.CookieStaff,
			),
			c.Authorizer.RequireAccountType(authctx.AccountTypeStaff),
			c.Authorizer.LoadActor(c.DBExecutor),
			c.Authorizer.RequireStaffRole(authctx.RoleStaffAdmin),
		),
		CustomerOnly: buildChain(
			c.Authenticator.RequireAuth(
				c.DBExecutor,
				c.DBTransactor,
				appcookie.CookieCustomer,
			),
			c.Authorizer.RequireAccountType(authctx.AccountTypeCustomer),
			c.Authorizer.LoadActor(c.DBExecutor),
		),
		CustomerWithIdempotency: buildChain(
			c.Authenticator.RequireAuth(
				c.DBExecutor,
				c.DBTransactor,
				appcookie.CookieCustomer,
			),
			c.Authorizer.RequireAccountType(authctx.AccountTypeCustomer),
			c.Authorizer.LoadActor(c.DBExecutor),
			idempotencyMw.RequireIdempotency(),
		),
	}
}

func (rc *RouteChains) StaffWithPerm(authorizer authenRepo.Authorizer, permission string, h apphttp.AppHandler) http.HandlerFunc {
	return rc.StaffOnly(authorizer.RequirePermission(permission)(h))
}

func NewRouter(c *Container) *chi.Mux {
	chains := NewRouteChains(c)
	h := initHandlers(c)

	r := chi.NewRouter()

	// Unversioned infrastructure endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	mountModuleRoutes := func(router chi.Router) {
		bindIdentityRoutes(router, h, chains)
		bindCatalogRoutes(router, h, chains, c)
		bindCommerceRoutes(router, h, chains)
	}
	r.Route("/api/v1", mountModuleRoutes)
	r.Group(mountModuleRoutes)

	return r
}

type handlers struct {
	product   *productH.ProductHandler
	inventory *inventoryH.InventoryHandler
	auth      *authH.AuthHandler
	staff     *staffH.StaffHandler
	cart      *cartH.CartHandler
	user      *userH.UserHandler
	address   *addressH.AddressHandler
	payment   *paymentH.PaymentHandler
	shop      *shopH.ShopHandler
	courier   *courierH.CourierHandler
	shipment  *shipmentH.ShipmentHandler
	order     *orderH.OrderHandler
	wishlist  *wishlistH.WishlistHandler
	review    *reviewH.ReviewHandler
}

func initHandlers(c *Container) *handlers {
	return &handlers{
		product: productH.NewProductHandler(
			&c.FindProducts,
			&c.GetProduct,
			&c.SaveProduct,
			&c.DeleteProduct,
			&c.AddProductImages,
			&c.GetProductStats,
		),
		inventory: inventoryH.NewInventoryHandler(
			&c.Inventory,
		),
		auth: authH.NewAuthHandler(
			&c.Me,
			&c.Logout,
			&c.LoginCustomer,
			&c.LoginStaff,
			&c.RegisterCustomer,
			&c.VerifyAccount,
			&c.GetAccount,
			&c.AuthenticateOAuth,
			&c.RequestPasswordReset,
			&c.VerifyPasswordReset,
			&c.ResetPassword,
			&c.RefreshToken,
			&c.DeleteAccount,
			c.GoogleOAuth,
		),
		staff: staffH.NewStaffHandler(
			&c.Staff,
		),
		cart: cartH.NewCartHandler(
			&c.Cart,
		),
		user: userH.NewUserHandler(
			&c.User,
		),
		address: addressH.NewAddressHandler(
			&c.Address,
		),
		payment: paymentH.NewPaymentHandler(
			&c.SavePaymentMethod,
			&c.ListPaymentMethod,
			&c.ProcessPaymentWebhook,
			&c.SavePaymentInstruction,
			&c.GetPaymentDetail,
			&c.CheckPaymentStatus,
		),
		shop: shopH.NewShopHandler(
			&c.Shop,
		),
		courier: courierH.NewCourierHandler(
			&c.Courier,
		),
		shipment: shipmentH.NewShipmentHandler(
			&c.EstimateShippingOptions,
			&c.UpdateShipmentStatus,
			&c.UpdateShipment,
		),
		order: orderH.NewOrderHandler(
			&c.FindOrders,
			&c.GetOrder,
			&c.CreateOrder,
			&c.UpdateOrderStatus,
			&c.DispatchShopShipment,
			&c.GetOrderTracking,
			&c.Shop,
			&c.Checkout,
		),
		wishlist: wishlistH.NewWishlistHandler(
			&c.Wishlist,
		),
		review: reviewH.NewReviewHandler(
			&c.Review,
		),
	}
}

func bindIdentityRoutes(r chi.Router, h *handlers, chains *RouteChains) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signin", chains.Core(h.auth.SignInEmail))
		r.Post("/staff/signin", chains.Core(h.auth.SignInStaffEmail))
		r.Post("/signup", chains.Core(h.auth.SignUpAccount))
		r.Post("/verify", chains.Core(h.auth.VerifyAccount))

		r.Post("/forgot-password", chains.Core(h.auth.ForgotPasswordCustomer))
		r.Post("/staff/forgot-password", chains.Core(h.auth.ForgotPasswordStaff))
		r.Post("/forgot-password/verify", chains.Core(h.auth.VerifyPasswordReset))
		r.Post("/forgot-password/reset", chains.Core(h.auth.ResetPassword))

		r.Post("/logout", chains.CustomerOnly(h.auth.Logout))
		r.Post("/staff/logout", chains.StaffOnly(h.auth.LogoutStaff))
		r.Post("/refresh", chains.Core(h.auth.RefreshCustomer))
		r.Post("/staff/refresh", chains.Core(h.auth.RefreshStaff))
		r.Get("/me", chains.CustomerOnly(h.auth.Me))
		r.Get("/staff/me", chains.StaffOnly(h.auth.Me))

		r.Get("/google/login", chains.Core(h.auth.GoogleLogin))
		r.Get("/google/callback", chains.Core(h.auth.GoogleCallback))
	})

	r.Route("/profile", func(r chi.Router) {
		r.Get("/", chains.CoreAuth(h.user.GetCurrentProfile))
		r.Put("/", chains.CoreAuth(h.user.UpdateCurrentProfile))
		r.Get("/staff", chains.StaffOnly(h.user.GetCurrentProfile))
		r.Put("/staff", chains.StaffOnly(h.user.UpdateCurrentProfile))
		r.Delete("/", chains.CustomerOnly(h.auth.DeleteAccount))
	})

	r.Route("/users", func(r chi.Router) {
		r.Get("/{id}", chains.StaffAdminOnly(h.user.GetUserByID))
	})

	r.Route("/users/me", func(r chi.Router) {
		r.Get("/", chains.CustomerOnly(h.user.GetCurrentUser))

		r.Route("/addresses", func(r chi.Router) {
			r.Get("/", chains.CustomerOnly(h.address.ListUserAddresses))
			r.Post("/", chains.CustomerOnly(h.address.SaveUserAddress))
			r.Delete("/{addressID}", chains.CustomerOnly(h.address.DeleteUserAddress))
		})

		r.Route("/orders", func(r chi.Router) {
			r.Get("/", chains.CustomerOnly(h.order.ListMyOrders))
			r.Get("/{orderID}", chains.CustomerOnly(h.order.GetMyOrder))
			r.Get("/{orderID}/tracking", chains.CustomerOnly(h.order.GetMyOrderTracking))
			r.Get("/{orderID}/payment", chains.CustomerOnly(h.payment.GetMyOrderPayment))
			r.Post("/{orderID}/payment/check", chains.CustomerOnly(h.payment.CheckMyOrderPaymentStatus))
		})

		r.Route("/wishlist", func(r chi.Router) {
			r.Get("/", chains.CustomerOnly(h.wishlist.GetWishlist))
			r.Post("/{productId}", chains.CustomerOnly(h.wishlist.AddToWishlist))
			r.Delete("/{productId}", chains.CustomerOnly(h.wishlist.RemoveFromWishlist))
		})
	})

	r.Route("/staff", func(r chi.Router) {
		r.Get("/", chains.StaffAdminOnly(h.staff.FindStaff))
		r.Post("/", chains.StaffAdminOnly(h.staff.CreateStaff))
		r.Route("/{staffID}", func(r chi.Router) {
			r.Put("/", chains.StaffAdminOnly(h.staff.UpdateStaff))
			r.Delete("/", chains.StaffAdminOnly(h.staff.DeleteStaff))

			r.Route("/accounts", func(r chi.Router) {
				r.Get("/", chains.StaffAdminOnly(h.staff.ListStaffAccounts))
				r.Post("/", chains.StaffAdminOnly(h.staff.AddStaffAccount))
				r.Delete("/{accountID}", chains.StaffAdminOnly(h.staff.RemoveStaffAccount))
			})
		})
	})
}

func bindCatalogRoutes(r chi.Router, h *handlers, chains *RouteChains, c *Container) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", chains.Core(h.product.FindProducts))
		r.Post("/", chains.StaffOnly(h.product.SaveProduct))
		r.Get("/stats", chains.StaffAdminOnly(h.product.GetProductStats))

		r.Route("/{productId}/reviews", func(r chi.Router) {
			r.Get("/", chains.Core(h.review.ListProductReviews))
			r.Post("/", chains.CustomerOnly(h.review.CreateReview))
		})

		r.Get("/{slug}", chains.Core(h.product.GetProduct))

		r.Route("/id/{id}", func(r chi.Router) {
			r.Delete("/", chains.StaffAdminOnly(h.product.DeleteProduct))
			r.Post("/images", chains.StaffOnly(h.product.AddProductImages))
		})
	})

	r.Route("/reviews", func(r chi.Router) {
		r.Delete("/{id}", chains.CoreAuth(h.review.DeleteReview))
	})

	r.Route("/shops", func(r chi.Router) {
		r.Get("/", chains.Core(h.shop.FindShops))
		r.Post("/", chains.StaffAdminOnly(h.shop.SaveShop))

		r.Route("/{shopID}", func(r chi.Router) {
			r.Get("/", chains.Core(h.shop.GetShopByID))
			r.Put("/", chains.StaffWithPerm(c.Authorizer, authctx.PermissionShopUpdate, h.shop.SaveShop))
			r.Delete("/", chains.StaffAdminOnly(h.shop.DeleteShop))

			r.Route("/addresses", func(r chi.Router) {
				r.Get("/", chains.Core(h.shop.GetShopAddresses))
				r.Post("/", chains.StaffWithPerm(c.Authorizer, authctx.PermissionAddressManage, h.address.CreateShopAddress))
				r.Put("/{addressID}", chains.StaffWithPerm(c.Authorizer, authctx.PermissionAddressManage, h.address.UpdateShopAddress))
				r.Delete("/{addressID}", chains.StaffWithPerm(c.Authorizer, authctx.PermissionAddressManage, h.address.DeleteShopAddress))
			})

			r.Route("/products", func(r chi.Router) {
				r.Get("/", chains.Core(h.shop.GetShopProducts))
				r.Post("/{productID}/inventories", chains.StaffWithPerm(c.Authorizer, authctx.PermissionInventoryManage, h.inventory.AddInventory))
				r.Put("/{productID}/inventories", chains.StaffWithPerm(c.Authorizer, authctx.PermissionInventoryManage, h.inventory.UpdateInventory))
				r.Delete("/{productID}/inventories", chains.StaffWithPerm(c.Authorizer, authctx.PermissionInventoryManage, h.inventory.RemoveInventory))
			})
		})
	})
}

func bindCommerceRoutes(r chi.Router, h *handlers, chains *RouteChains) {
	r.Route("/carts", func(r chi.Router) {
		r.Get("/", chains.CustomerOnly(h.cart.GetCart))

		r.Route("/checkout", func(r chi.Router) {
			r.Post("/", chains.CustomerOnly(h.order.Checkout))
			r.Post("/calculate", chains.CustomerOnly(h.order.CheckoutEstimate))
		})

		r.Route("/items", func(r chi.Router) {
			r.Post("/", chains.CustomerOnly(h.cart.AddItem))
			r.Put("/{shopID}/{productID}", chains.CustomerOnly(h.cart.UpdateItem))
			r.Put("/{cartItemID}", chains.CustomerOnly(h.cart.UpdateItemByID))
			r.Patch("/{cartItemID}", chains.CustomerOnly(h.cart.UpdateItemByID))
			r.Delete("/{cartItemID}", chains.CustomerOnly(h.cart.RemoveItemByID))
			r.Delete("/{shopID}/{productID}", chains.CustomerOnly(h.cart.RemoveItem))
		})
	})

	r.Route("/couriers", func(r chi.Router) {
		r.Get("/", chains.CoreAuth(h.courier.ListAllCouriers))
	})

	r.Route("/midtrans", func(r chi.Router) {
		r.Post("/webhook", chains.Core(h.payment.HandleMidtransWebhook))
	})

	r.Route("/payments", func(r chi.Router) {
		r.Route("/methods", func(r chi.Router) {
			r.Get("/", chains.StaffOnly(h.payment.ListPaymentMethod))
			r.Patch("/{methodID}", chains.StaffAdminOnly(h.payment.UpdatePaymentMethodActive))
			r.Post("/{methodID}/instruction", chains.StaffAdminOnly(h.payment.SavePaymentInstruction))
		})
	})

	r.Route("/shipping", func(r chi.Router) {
		r.Post("/cost", chains.CoreAuth(h.shipment.EstimateShippingOptions))
	})

	r.Route("/order", func(r chi.Router) {
		r.Post("/", chains.CustomerWithIdempotency(h.order.CreateOrder))
	})

	r.Route("/orders", func(r chi.Router) {
		r.Get("/", chains.StaffOnly(h.order.FindOrders))
		r.Get("/{orderID}", chains.StaffOnly(h.order.GetOrder))
		r.Get("/{orderID}/tracking", chains.StaffOnly(h.order.GetOrderTrackingForStaff))
		r.Patch("/{orderID}/status", chains.StaffOnly(h.order.UpdateOrderStatus))
		r.Post("/{orderID}/shipments", chains.StaffOnly(h.order.DispatchOrderShipment))
	})

	r.Route("/shipments", func(r chi.Router) {
		r.Patch("/{shipmentID}/status", chains.StaffOnly(h.shipment.UpdateShipmentStatus))
		r.Patch("/{shipmentID}/dispatch", chains.StaffOnly(h.shipment.DispatchShipment))
		r.Patch("/{shipmentID}", chains.StaffOnly(h.shipment.UpdateShipment))
	})
}
