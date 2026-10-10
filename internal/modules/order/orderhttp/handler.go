package orderhttp

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/httpx"
	"komecore/internal/modules/cart/cartdomain"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderusecase"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/shipment/shipmentdomain"
	"komecore/internal/modules/shop/shopusecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OrderHandler = orderHandler

type orderHandler struct {
	findOrders           *orderusecase.FindOrdersUsecase
	getOrder             *orderusecase.GetOrderUsecase
	createOrder          *orderusecase.CreateOrderUsecase
	updateOrderStatus    *orderusecase.UpdateOrderStatusUsecase
	dispatchShopShipment *orderusecase.DispatchShopShipmentUsecase
	getOrderTracking     *orderusecase.GetOrderTrackingUsecase
	shopService          *shopusecase.ShopService
	checkout             *orderusecase.CheckoutUsecase
}

func NewOrderHandler(
	findOrders *orderusecase.FindOrdersUsecase,
	getOrder *orderusecase.GetOrderUsecase,
	createOrder *orderusecase.CreateOrderUsecase,
	updateOrderStatus *orderusecase.UpdateOrderStatusUsecase,
	dispatchShopShipment *orderusecase.DispatchShopShipmentUsecase,
	getOrderTracking *orderusecase.GetOrderTrackingUsecase,
	shopService *shopusecase.ShopService,
	checkout *orderusecase.CheckoutUsecase,
) *orderHandler {
	return &orderHandler{
		findOrders:           findOrders,
		getOrder:             getOrder,
		createOrder:          createOrder,
		updateOrderStatus:    updateOrderStatus,
		dispatchShopShipment: dispatchShopShipment,
		getOrderTracking:     getOrderTracking,
		shopService:          shopService,
		checkout:             checkout,
	}
}

func (h *orderHandler) FindOrders(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}

	if actor.Type != authctx.AccountTypeStaff {
		return apperror.NewForbidden("forbidden: staff account required")
	}

	page := httpx.QueryIntDefault(r, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := httpx.QueryIntDefault(r, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	sort := httpx.Query(r, "sort")
	idStr := httpx.Query(r, "id")
	number := httpx.Query(r, "number")
	customerIDStr := httpx.Query(r, "customer_id")
	status := httpx.Query(r, "status")

	input := orderusecase.FindOrdersInput{
		Page:  page,
		Limit: limit,
		Sort:  sort,
	}

	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return apperror.NewBadRequest("invalid order id")
		}
		input.ID = &id
	}

	if number != "" {
		input.Number = &number
	}

	if customerIDStr != "" {
		customerID, err := uuid.Parse(customerIDStr)
		if err != nil {
			return apperror.NewBadRequest("invalid customer id")
		}
		input.CustomerID = &customerID
	}

	if status != "" {
		input.Status = &status
	} else if statusesParam := httpx.Query(r, "statuses"); statusesParam != "" {
		var parsedStatuses []string
		parts := strings.Split(statusesParam, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				parsedStatuses = append(parsedStatuses, trimmed)
			}
		}
		input.Statuses = parsedStatuses
	}

	fromDateStr := httpx.Query(r, "from_date")
	if fromDateStr != "" {
		if t, err := time.Parse(time.RFC3339, fromDateStr); err == nil {
			input.FromDate = &t
		} else if t, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			input.FromDate = &t
		} else {
			return apperror.NewBadRequest("invalid from_date format")
		}
	}

	toDateStr := httpx.Query(r, "to_date")
	if toDateStr != "" {
		if t, err := time.Parse(time.RFC3339, toDateStr); err == nil {
			input.ToDate = &t
		} else if t, err := time.Parse("2006-01-02", toDateStr); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			input.ToDate = &endOfDay
		} else {
			return apperror.NewBadRequest("invalid to_date format")
		}
	}

	shopID, shopSpecified, err := h.resolveShopFilter(r)
	if err != nil {
		return err
	}
	if shopSpecified {
		if shopID == nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"orders": []orderResponse{},
				"page":   page,
				"limit":  limit,
				"total":  0,
			})
			return nil
		}
		input.ShopID = shopID
	}

	if actor.StaffID != nil && !actor.IsSuperAdmin() {
		allAssigned := actor.GetAssignedShopIDs()
		var assignedIDs []uuid.UUID
		for _, sID := range allAssigned {
			if actor.HasPermission(sID, authctx.PermissionOrderRead) {
				assignedIDs = append(assignedIDs, sID)
			}
		}

		if len(assignedIDs) == 0 {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"orders": []orderResponse{},
				"page":   page,
				"limit":  limit,
				"total":  0,
			})
			return nil
		}

		if input.ShopID != nil {
			if !actor.HasPermission(*input.ShopID, authctx.PermissionOrderRead) {
				return apperror.NewForbidden("forbidden: missing order:read permission for this shop")
			}
		} else {
			input.ShopIDs = assignedIDs
		}
	}

	orders, total, err := h.findOrders.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	results := make([]orderResponse, len(orders))
	for i, o := range orders {
		results[i] = buildOrderResponse(o)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"orders": results,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
	return nil
}

// GetOrder handles GET /orders/{orderID} â€” staff-only, returns a single order with full detail.
func (h *orderHandler) GetOrder(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.Type != authctx.AccountTypeStaff {
		return apperror.NewForbidden("forbidden: staff account required")
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	result, err := h.getOrder.Execute(r.Context(), orderusecase.GetOrderInput{
		OrderID: orderID,
	})
	if err != nil {
		return err
	}
	if result == nil {
		return apperror.NewNotFound("order not found")
	}

	if actor.StaffID != nil && !actor.IsSuperAdmin() {
		var permittedItems []orderdomain.OrderItem
		for _, item := range result.Items {
			if actor.HasPermission(item.ShopID, authctx.PermissionOrderRead) {
				permittedItems = append(permittedItems, item)
			}
		}
		if len(permittedItems) == 0 {
			return apperror.NewForbidden("forbidden: missing order:read permission for this shop's order")
		}
		result.Items = permittedItems

		var permittedShipments []shipmentdomain.Shipment
		for _, s := range result.Shipments {
			hasItem := false
			for _, itm := range permittedItems {
				if itm.ShipmentID != nil && *itm.ShipmentID == s.ID {
					hasItem = true
					break
				}
			}
			if hasItem {
				permittedShipments = append(permittedShipments, s)
			}
		}
		result.Shipments = permittedShipments
		if len(permittedShipments) > 0 {
			result.Shipment = &permittedShipments[0]
		} else {
			result.Shipment = nil
		}
	}

	resp := buildOrderResponse(orderusecase.OrderSearchResult{
		Order:       result.Order,
		Items:       result.Items,
		Payment:     result.Payment,
		ChannelData: result.ChannelData,
		Shipment:    result.Shipment,
		Shipments:   result.Shipments,
	})

	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

// ListMyOrders handles GET /users/me/orders â€” customer-only, returns the caller's orders with detail.
func (h *orderHandler) ListMyOrders(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	page := httpx.QueryIntDefault(r, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := httpx.QueryIntDefault(r, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	sort := httpx.Query(r, "sort")
	status := httpx.Query(r, "status")

	input := orderusecase.FindOrdersInput{
		Page:       page,
		Limit:      limit,
		Sort:       sort,
		CustomerID: &customerID,
	}

	if status != "" {
		input.Status = &status
	} else if statusesParam := httpx.Query(r, "statuses"); statusesParam != "" {
		var parsedStatuses []string
		parts := strings.Split(statusesParam, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				parsedStatuses = append(parsedStatuses, trimmed)
			}
		}
		input.Statuses = parsedStatuses
	}

	fromDateStr := httpx.Query(r, "from_date")
	if fromDateStr != "" {
		if t, err := time.Parse(time.RFC3339, fromDateStr); err == nil {
			input.FromDate = &t
		} else if t, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			input.FromDate = &t
		} else {
			return apperror.NewBadRequest("invalid from_date format")
		}
	}

	toDateStr := httpx.Query(r, "to_date")
	if toDateStr != "" {
		if t, err := time.Parse(time.RFC3339, toDateStr); err == nil {
			input.ToDate = &t
		} else if t, err := time.Parse("2006-01-02", toDateStr); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			input.ToDate = &endOfDay
		} else {
			return apperror.NewBadRequest("invalid to_date format")
		}
	}

	shopID, shopSpecified, err := h.resolveShopFilter(r)
	if err != nil {
		return err
	}
	if shopSpecified {
		if shopID == nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"orders": []orderResponse{},
				"page":   page,
				"limit":  limit,
				"total":  0,
			})
			return nil
		}
		input.ShopID = shopID
	}

	orders, total, err := h.findOrders.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	results := make([]orderResponse, len(orders))
	for i, o := range orders {
		results[i] = buildOrderResponse(o)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"orders": results,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
	return nil
}

// GetMyOrder handles GET /users/me/orders/{orderID} â€” customer-only, returns their own order with full detail.
func (h *orderHandler) GetMyOrder(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	result, err := h.getOrder.Execute(r.Context(), orderusecase.GetOrderInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})
	if err != nil {
		return err
	}
	if result == nil {
		return apperror.NewNotFound("order not found")
	}

	resp := buildOrderResponse(orderusecase.OrderSearchResult{
		Order:       result.Order,
		Items:       result.Items,
		Payment:     result.Payment,
		ChannelData: result.ChannelData,
		Shipment:    result.Shipment,
		Shipments:   result.Shipments,
	})
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

// CreateOrder handles POST /order â€” customer-only.
func (h *orderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) error {
	authCtx, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	userID := authCtx.UserID

	var req createOrderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	parsedAddressID, err := uuid.Parse(req.AddressID)
	if err != nil {
		return apperror.NewBadRequest("invalid address id")
	}

	parsedPaymentMethodID, err := uuid.Parse(req.SelectedPayment.ID)
	if err != nil {
		return apperror.NewBadRequest("invalid payment method id")
	}

	var shopsInput []orderusecase.OrderShopInput
	for _, shopReq := range req.Shops {
		parsedShopID, err := uuid.Parse(shopReq.ShopID)
		if err != nil {
			return apperror.NewBadRequest("invalid shop id")
		}
		if shopReq.ShopName == "" {
			return apperror.NewBadRequest("invalid shop name")
		}

		var itemsInput []orderusecase.OrderItemInput
		for _, itemReq := range shopReq.Items {
			if itemReq.Quantity <= 0 {
				return apperror.NewBadRequest("invalid quantity")
			}

			productID, err := uuid.Parse(itemReq.ProductID)
			if err != nil {
				return apperror.NewBadRequest("invalid product id")
			}

			productName := itemReq.ProductName
			if productName == "" {
				return apperror.NewBadRequest("invalid product name")
			}

			var opt cartdomain.ItemOptions
			if itemReq.ItemOptions != nil {
				opt = cartdomain.ItemOptions(itemReq.ItemOptions)
			}

			itemsInput = append(
				itemsInput,
				orderusecase.OrderItemInput{
					ProductID:   productID,
					ItemOptions: opt.Normalized(),
					ProductName: productName,
					Quantity:    itemReq.Quantity,
				},
			)
		}

		var courierInput *orderusecase.OrderCourierInput
		if shopReq.Courier != nil {
			courierInput = &orderusecase.OrderCourierInput{
				Code:    shopReq.Courier.Code,
				Service: shopReq.Courier.Service,
			}
		}

		shopsInput = append(
			shopsInput,
			orderusecase.OrderShopInput{
				ShopID:   parsedShopID,
				ShopName: shopReq.ShopName,
				Courier:  courierInput,
				Items:    itemsInput,
			},
		)
	}

	input := orderusecase.CreateOrderInput{
		UserID:          userID,
		CustomerID:      customerID,
		AddressID:       parsedAddressID,
		PaymentMethodID: parsedPaymentMethodID,
		CouponCode:      req.CouponCode,
		Shops:           shopsInput,
	}
	result, err := h.createOrder.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	resp := createOrderResponse{
		OrderID:     result.OrderID.String(),
		Instruction: result.Instruction,
	}
	if result.PaymentAccount != nil {
		resp.PaymentAccount = &createOrderPaymentAccountResponse{
			AccountName:   result.PaymentAccount.AccountName,
			AccountNumber: result.PaymentAccount.AccountNumber,
			PhoneNumber:   result.PaymentAccount.PhoneNumber,
			QRString:      result.PaymentAccount.QRString,
		}
	}
	if result.ChannelData != nil {
		resp.ChannelData = &paymentChannelDataResponse{
			ChannelType: string(result.ChannelData.ChannelType),
			DisplayName: result.ChannelData.DisplayName,
			ActionURL:   result.ChannelData.ActionURL,
			ExpiresAt:   result.ChannelData.ExpiresAt,
		}
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

// UpdateOrderStatus handles PATCH /orders/{orderID}/status â€” staff-only.
// Transitions the order to the requested status. When the target status is
// "shipped", the configured LogisticsProvider creates a shipment record.
// In Komerce mode the provider calls the external API and returns a tracking
// number automatically. In manual mode the optional "tracking_number" field
// in the request body is used instead â€” no external call is made.
func (h *orderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.Type != authctx.AccountTypeStaff {
		return apperror.NewForbidden("forbidden: staff account required")
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	var req updateOrderStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.Status == "" {
		return apperror.NewBadRequest("status is required")
	}

	if actor.StaffID != nil && !actor.IsSuperAdmin() {
		existingOrder, err := h.getOrder.Execute(r.Context(), orderusecase.GetOrderInput{
			OrderID: orderID,
		})
		if err != nil || existingOrder == nil {
			return apperror.NewNotFound("order not found")
		}

		uniqueShops := make(map[uuid.UUID]bool)
		for _, item := range existingOrder.Items {
			if actor.HasPermission(item.ShopID, authctx.PermissionOrderUpdateStatus) {
				uniqueShops[item.ShopID] = true
			}
		}
		if len(uniqueShops) == 0 {
			return apperror.NewForbidden("forbidden: missing order:update_status permission for this shop's order")
		}
	}

	var shipmentsInput []orderusecase.ShipmentDispatchInput
	if len(req.Shipments) > 0 {
		for _, sReq := range req.Shipments {
			var itemUUIDs []uuid.UUID
			for _, idStr := range sReq.ItemIDs {
				parsed, err := uuid.Parse(idStr)
				if err != nil {
					return apperror.NewBadRequest("invalid item id in shipments")
				}
				itemUUIDs = append(itemUUIDs, parsed)
			}
			shipmentsInput = append(shipmentsInput, orderusecase.ShipmentDispatchInput{
				FulfillmentMethod: sReq.FulfillmentMethod,
				Courier:           sReq.Courier,
				Service:           sReq.Service,
				TrackingNumber:    sReq.TrackingNumber,
				ItemIDs:           itemUUIDs,
			})
		}
	}

	result, err := h.updateOrderStatus.Execute(r.Context(), orderusecase.UpdateOrderStatusInput{
		OrderID:           orderID,
		Status:            orderdomain.OrderStatus(req.Status),
		TrackingNumber:    req.TrackingNumber,
		FulfillmentMethod: req.FulfillmentMethod,
		Shipments:         shipmentsInput,
	})
	if err != nil {
		return err
	}

	resp := buildOrderResponse(orderusecase.OrderSearchResult{
		Order:     result.Order,
		Items:     []orderdomain.OrderItem{},
		Shipment:  result.Shipment,
		Shipments: result.Shipments,
	})
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

// DispatchOrderShipment handles POST /orders/{orderID}/shipments â€” staff-only.
// Creates a shipment for a specific shop's order items.
func (h *orderHandler) DispatchOrderShipment(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.Type != authctx.AccountTypeStaff {
		return apperror.NewForbidden("forbidden: staff account required")
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	var req dispatchShopShipmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	shopID, err := uuid.Parse(req.ShopID)
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	if len(req.ItemIDs) == 0 {
		return apperror.NewBadRequest("item_ids is required and must not be empty")
	}

	var itemUUIDs []uuid.UUID
	for _, idStr := range req.ItemIDs {
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			return apperror.NewBadRequest("invalid item id in item_ids")
		}
		itemUUIDs = append(itemUUIDs, parsed)
	}

	if actor.StaffID != nil && !actor.IsSuperAdmin() {
		if !actor.HasPermission(shopID, authctx.PermissionOrderUpdateStatus) {
			return apperror.NewForbidden("forbidden: missing order:update_status permission for this shop")
		}
	}

	res, err := h.dispatchShopShipment.Execute(r.Context(), orderusecase.DispatchShopShipmentInput{
		OrderID:           orderID,
		ShopID:            shopID,
		FulfillmentMethod: req.FulfillmentMethod,
		Courier:           req.Courier,
		Service:           req.Service,
		TrackingNumber:    req.TrackingNumber,
		ItemIDs:           itemUUIDs,
	})
	if err != nil {
		return err
	}

	resp := map[string]any{
		"order_id":          res.Order.ID.String(),
		"order_status":      string(res.Order.Status),
		"shipment_id":       res.Shipment.ID.String(),
		"all_items_shipped": res.AllItemsShipped,
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *orderHandler) GetMyOrderTracking(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	orderID, err := httpx.ParamUUID(r, "orderID")
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	input := orderusecase.GetOrderTrackingInput{
		OrderID:    orderID,
		CustomerID: customerID,
	}

	result, err := h.getOrderTracking.Execute(r.Context(), input)
	if err != nil {
		return err
	}
	if result == nil {
		return apperror.NewNotFound("tracking information not found")
	}

	if result.Warning != nil {
		w.Header().Set("X-Warning", *result.Warning)
	}

	httpx.WriteJSON(w, http.StatusOK, result)
	return nil
}

// GetOrderTrackingForStaff handles GET /orders/{orderID}/tracking â€” staff-only.
func (h *orderHandler) GetOrderTrackingForStaff(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.Type != authctx.AccountTypeStaff {
		return apperror.NewForbidden("forbidden: staff account required")
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	if actor.StaffID != nil && !actor.IsSuperAdmin() {
		existingOrder, err := h.getOrder.Execute(r.Context(), orderusecase.GetOrderInput{
			OrderID: orderID,
		})
		if err != nil || existingOrder == nil {
			return apperror.NewNotFound("order not found")
		}

		hasAccess := false
		for _, item := range existingOrder.Items {
			if actor.HasPermission(item.ShopID, authctx.PermissionOrderRead) {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return apperror.NewForbidden("forbidden: missing order:read permission for this shop's order")
		}
	}

	input := orderusecase.GetOrderTrackingInput{
		OrderID:    orderID,
		CustomerID: uuid.Nil,
	}

	result, err := h.getOrderTracking.Execute(r.Context(), input)
	if err != nil {
		return err
	}
	if result == nil {
		return apperror.NewNotFound("tracking information not found")
	}
	if result.Warning != nil {
		w.Header().Set("X-Warning", *result.Warning)
	}

	httpx.WriteJSON(w, http.StatusOK, result)
	return nil
}

// buildOrderResponse converts an OrderSearchResult into the wire response.
func buildOrderResponse(o orderusecase.OrderSearchResult) orderResponse {
	items := make([]orderItemResponse, len(o.Items))
	for j, item := range o.Items {
		var shipmentIDStr *string
		if item.ShipmentID != nil {
			s := item.ShipmentID.String()
			shipmentIDStr = &s
		}

		items[j] = orderItemResponse{
			ID:               item.ID.String(),
			ShipmentID:       shipmentIDStr,
			ProductID:        item.ProductID.String(),
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			UnitPrice:        item.UnitPrice,
			Subtotal:         item.Subtotal,
			ShopID:           item.ShopID.String(),
			ShopName:         item.ShopName,
			CourierCode:      item.CourierCode,
			CourierService:   item.CourierService,
			ShippingFeeTotal: item.ShippingFee,
			ItemOptions:      item.ItemOptions.Normalized(),
		}
	}

	shipments := make([]shipmentDetailResponse, len(o.Shipments))
	for i := range o.Shipments {
		shipments[i] = *mapShipmentDetail(&o.Shipments[i])
	}

	resp := orderResponse{
		ID:                o.Order.ID.String(),
		Number:            o.Order.Number,
		CustomerID:        o.Order.CustomerID.String(),
		AddressID:         o.Order.AddressID.String(),
		Status:            string(o.Order.Status),
		Subtotal:          o.Order.Subtotal,
		ShippingFee:       o.Order.ShippingFee,
		Total:             o.Order.Total,
		ConfirmedAt:       o.Order.ConfirmedAt,
		HandlingExpiresAt: o.Order.HandlingExpiresAt,
		CreatedAt:         o.Order.CreatedAt,
		UpdatedAt:         o.Order.UpdatedAt,
		Items:             items,
		Shipments:         shipments,
	}

	if o.Payment != nil {
		resp.Payment = mapPaymentDetail(o.Payment, o.ChannelData)
	}
	if o.Shipment != nil {
		resp.Shipment = mapShipmentDetail(o.Shipment)
	} else if len(shipments) > 0 {
		resp.Shipment = &shipments[0]
	}
	if o.Address != nil {
		resp.Address = &orderAddressResponse{
			ID:           o.Address.ID.String(),
			CustomerID:   o.Address.CustomerID.String(),
			ReceiverName: o.Address.ReceiverName,
			Phone:        o.Address.Phone,
			IsDefault:    o.Address.IsDefault,
			Province:     o.Address.Detail.Province,
			City:         o.Address.Detail.City,
			District:     o.Address.Detail.District,
			FullAddress:  o.Address.Detail.FullAddress,
			PostalCode:   o.Address.Detail.PostalCode,
			Latitude:     o.Address.Detail.Latitude,
			Longitude:    o.Address.Detail.Longitude,
		}
	}

	return resp
}

func mapPaymentDetail(p *paymentdomain.Payment, cd *paymentdomain.PaymentChannelData) *paymentDetailResponse {
	resp := &paymentDetailResponse{
		ID:        p.ID.String(),
		Status:    string(p.Status),
		Provider:  p.Provider,
		Amount:    p.Amount,
		ExpiresAt: p.ExpiresAt,
		CreatedAt: p.CreatedAt,
	}
	if cd != nil {
		resp.ChannelData = &paymentChannelDataResponse{
			ChannelType: string(cd.ChannelType),
			DisplayName: cd.DisplayName,
			ActionURL:   cd.ActionURL,
			ExpiresAt:   cd.ExpiresAt,
		}
	}

	return resp
}

func mapShipmentDetail(s *shipmentdomain.Shipment) *shipmentDetailResponse {
	return &shipmentDetailResponse{
		ID:                s.ID.String(),
		OrderID:           s.OrderID.String(),
		Status:            string(s.Status),
		FulfillmentMethod: string(s.FulfillmentMethod),
		Courier:           s.Courier,
		Service:           s.Service,
		TrackingNumber:    s.TrackingNumber,
		Cost:              s.Cost,
		CreatedAt:         s.CreatedAt,
	}
}

func (h *orderHandler) resolveShopFilter(r *http.Request) (*uuid.UUID, bool, error) {
	shopIDStr := httpx.Query(r, "shop_id")
	shopSlug := httpx.Query(r, "shop_slug")
	shopParam := httpx.Query(r, "shop")

	if shopIDStr == "all" || shopSlug == "all" || shopParam == "all" {
		return nil, false, nil
	}

	targetIDStr := ""
	targetSlug := ""

	if shopIDStr != "" {
		targetIDStr = shopIDStr
	} else if shopSlug != "" {
		targetSlug = shopSlug
	} else if shopParam != "" {
		if parsed, err := uuid.Parse(shopParam); err == nil {
			return &parsed, true, nil
		}
		targetSlug = shopParam
	}

	if targetIDStr != "" {
		id, err := uuid.Parse(targetIDStr)
		if err != nil {
			return nil, true, apperror.NewBadRequest("invalid shop id")
		}
		return &id, true, nil
	}

	if targetSlug != "" {
		if targetSlug == "all" {
			return nil, false, nil
		}
		if h.shopService == nil {
			return nil, true, apperror.NewInternal(errors.New("shop filter service unavailable"))
		}
		shop, err := h.shopService.GetBySlug(r.Context(), targetSlug)
		if err != nil {
			return nil, true, err
		}
		if shop == nil {
			return nil, true, nil
		}
		return &shop.ID, true, nil
	}

	return nil, false, nil
}
