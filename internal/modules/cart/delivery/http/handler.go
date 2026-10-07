package http

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	domain "komecore/internal/modules/cart/domain"
	"komecore/internal/modules/cart/usecase"

	"github.com/google/uuid"
)

type CartHandler struct {
	addItem    *usecase.AddItemUsecase
	getCart    *usecase.GetCartUsecase
	updateItem *usecase.UpdateItemUsecase
	removeItem *usecase.RemoveItemUsecase
	checkout   *usecase.CheckoutUsecase
}

func NewCartHandler(
	addItem *usecase.AddItemUsecase,
	getCart *usecase.GetCartUsecase,
	updateItem *usecase.UpdateItemUsecase,
	removeItem *usecase.RemoveItemUsecase,
	checkout *usecase.CheckoutUsecase,
) *CartHandler {
	return &CartHandler{
		addItem:    addItem,
		getCart:    getCart,
		updateItem: updateItem,
		removeItem: removeItem,
		checkout:   checkout,
	}
}

func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	result, err := h.getCart.Execute(r.Context(), customerID)
	if err != nil {
		return err
	}
	if result == nil || result.Cart == nil {
		return apperrors.NewNotFound("cart not found")
	}

	var total int64
	items := make([]cartItemView, 0, len(result.Cart.Items))

	for _, item := range result.Cart.Items {
		var shopName, shopSlug string
		if result.Shops != nil {
			if s, ok := result.Shops[item.ShopID]; ok {
				shopName = s.Name
				shopSlug = s.Slug
			}
		}

		if result.Products == nil {
			continue
		}

		productData, ok := result.Products[item.ProductID]
		if !ok {
			continue
		}

		normOpts := item.ItemOptions.Normalized()
		price := productData.Product.Price
		quantity := item.Quantity
		subtotal := price * int64(quantity)

		image := productImageResponse{}
		if productData.Images.Thumbnail != "" {
			image.Thumbnail = &productData.Images.Thumbnail
		}

		items = append(items, cartItemView{
			CartItemID:  item.ID,
			ProductID:   item.ProductID,
			ShopID:      item.ShopID,
			ShopName:    shopName,
			ShopSlug:    shopSlug,
			Name:        productData.Product.Name,
			Price:       price,
			Quantity:    quantity,
			Subtotal:    subtotal,
			Image:       image,
			ItemOptions: normOpts,
		})

		total += subtotal
	}

	response := cartResponse{
		CartID: result.Cart.ID,
		Items:  items,
		Total:  total,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) error {
	var req addItemRequest

	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	shopID, err := uuid.Parse(req.ShopID)
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	if req.ProductID == "" {
		return apperrors.NewBadRequest("product_id is required")
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}
	if req.Quantity <= 0 {
		return apperrors.NewBadRequest("invalid quantity")
	}

	var opt domain.ItemOptions
	if req.ItemOptions != nil {
		opt = domain.ItemOptions(req.ItemOptions)
	}

	input := usecase.AddItemInput{
		CustomerID:  customerID,
		ProductID:   productID,
		ShopID:      shopID,
		Quantity:    req.Quantity,
		ItemOptions: opt.Normalized(),
	}
	if err := h.addItem.Execute(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "item added",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) error {
	var req updateItemRequest

	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := apphttp.ParamUUID(r, "productID")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	shopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	if req.Quantity < 0 {
		return apperrors.NewBadRequest("invalid quantity")
	}

	var opt *domain.ItemOptions
	if req.ItemOptions != nil {
		norm := domain.ItemOptions(req.ItemOptions).Normalized()
		opt = &norm
	}

	input := usecase.UpdateItemInput{
		CustomerID:  customerID,
		ProductID:   productID,
		ShopID:      shopID,
		Quantity:    req.Quantity,
		ItemOptions: opt,
	}

	if err := h.updateItem.Execute(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "item updated",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *CartHandler) UpdateItemByID(w http.ResponseWriter, r *http.Request) error {
	var req updateItemByIDRequest

	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	cartItemID, err := apphttp.ParamUUID(r, "cartItemID")
	if err != nil {
		return apperrors.NewBadRequest("invalid cart item id")
	}

	if req.Quantity <= 0 {
		return apperrors.NewBadRequest("invalid quantity")
	}

	var opt *domain.ItemOptions
	if req.ItemOptions != nil {
		norm := domain.ItemOptions(req.ItemOptions).Normalized()
		opt = &norm
	}

	input := usecase.UpdateItemByIDInput{
		CustomerID:  customerID,
		CartItemID:  cartItemID,
		Quantity:    req.Quantity,
		ItemOptions: opt,
	}

	if err := h.updateItem.ExecuteByID(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "item updated",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := apphttp.ParamUUID(r, "productID")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	shopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	input := usecase.RemoveItemInput{
		CustomerID: customerID,
		ProductID:  productID,
		ShopID:     shopID,
	}

	if err := h.removeItem.Execute(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "item removed",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *CartHandler) RemoveItemByID(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	cartItemID, err := apphttp.ParamUUID(r, "cartItemID")
	if err != nil {
		return apperrors.NewBadRequest("invalid cart item id")
	}

	input := usecase.RemoveItemByIDInput{
		CustomerID: customerID,
		CartItemID: cartItemID,
	}

	if err := h.removeItem.ExecuteByID(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "item removed",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *CartHandler) Checkout(w http.ResponseWriter, r *http.Request) error {
	authCtx, _, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	var req checkoutRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	reqCheckoutCalc := checkoutCalculateRequest{
		Shops: req.Shops,
	}
	input, err := h.parseCheckoutInput(reqCheckoutCalc)
	if err != nil {
		return err
	}

	result, err := h.checkout.Execute(r.Context(), *authCtx, input)
	if err != nil {
		return err
	}

	var shopsResponse []shopResponse
	for _, shop := range result.Shops {
		var itemsResponse []checkoutItemResponse
		for _, item := range shop.Items {
			normOpts := item.ItemOptions.Normalized()
			itemsResponse = append(itemsResponse, checkoutItemResponse{
				ProductID:   item.ProductID,
				CartItemID:  item.CartItemID,
				ShopID:      item.ShopID,
				Name:        item.Name,
				Price:       item.Price,
				Quantity:    item.Quantity,
				Subtotal:    item.Subtotal,
				ItemOptions: normOpts,
			})
		}

		var shippingResponse []checkoutCouriersResponse
		for _, courier := range shop.CostCouriers {
			shippingResponse = append(shippingResponse, checkoutCouriersResponse{
				Code:    courier.Code,
				Name:    courier.Name,
				Service: courier.Service,
				ETD:     courier.ETD,
				Fee:     courier.Fee,
			})
		}

		shopReponse := shopResponse{
			ShopID:       shop.ShopID,
			ShopSlug:     shop.ShopSlug,
			ShopName:     shop.ShopName,
			Subtotal:     shop.Subtotal,
			Total:        &shop.Total,
			Items:        itemsResponse,
			CostCouriers: shippingResponse,
		}

		shopsResponse = append(shopsResponse, shopReponse)
	}

	var paymentMethods []paymentMethodResponse
	for _, pM := range result.PaymentMethods {
		paymentMethod := paymentMethodResponse{
			ID:          pM.ID,
			Name:        pM.Name,
			Type:        pM.Type,
			Description: pM.Description,
			Fee:         pM.Fee,
			Subtotal:    pM.Subtotal,
			Total:       pM.Total,
		}

		paymentMethods = append(paymentMethods, paymentMethod)
	}

	resp := checkoutResponse{
		Address: checkoutAddressResponse{
			ID:            result.Address.ID,
			RecipientName: result.Address.RecipientName,
			Phone:         result.Address.Phone,
			FullAddress:   result.Address.FullAddress,
		},
		Shops:          shopsResponse,
		TotalShipping:  result.TotalShippingFee,
		Subtotal:       result.Subtotal,
		PaymentMethods: paymentMethods,
	}

	if result.TotalAll > 0 {
		resp.TotalAll = &result.TotalAll
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *CartHandler) CheckoutEstimate(w http.ResponseWriter, r *http.Request) error {
	authCtx, _, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	var req checkoutCalculateRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	if req.AddressID == nil {
		return apperrors.NewBadRequest("address id required")
	}
	if req.PaymentMethodID == nil {
		return apperrors.NewBadRequest("payment method id required")
	}

	for _, shop := range req.Shops {
		if shop.Courier == nil {
			return apperrors.NewBadRequest("courier required")
		}
	}

	input, err := h.parseCheckoutInput(req)
	if err != nil {
		return err
	}

	result, err := h.checkout.Execute(r.Context(), *authCtx, input)
	if err != nil {
		return err
	}

	var shopsResponse []shopCalculateResponse
	for _, shop := range result.Shops {
		var itemsResponse []checkoutItemResponse
		for _, item := range shop.Items {
			normOpts := item.ItemOptions.Normalized()
			itemsResponse = append(itemsResponse, checkoutItemResponse{
				ProductID:   item.ProductID,
				CartItemID:  item.CartItemID,
				ShopID:      item.ShopID,
				Name:        item.Name,
				Price:       item.Price,
				Quantity:    item.Quantity,
				Subtotal:    item.Subtotal,
				ItemOptions: normOpts,
			})
		}

		var shippingResponse []checkoutCouriersResponse
		for _, courier := range shop.CostCouriers {
			shippingResponse = append(shippingResponse, checkoutCouriersResponse{
				Code:    courier.Code,
				Name:    courier.Name,
				Service: courier.Service,
				ETD:     courier.ETD,
				Fee:     courier.Fee,
			})
		}

		shopReponse := shopCalculateResponse{
			ShopID:   shop.ShopID,
			ShopSlug: shop.ShopSlug,
			ShopName: shop.ShopName,
			Subtotal: shop.Subtotal,
			Total:    &shop.Total,
			Items:    itemsResponse,
		}
		if shop.SelectedCourier != nil {
			shopReponse.SelectedCourier = selectedCourierResponse{
				Code:    shop.SelectedCourier.Code,
				Service: shop.SelectedCourier.Service,
				Fee:     shop.SelectedCourier.Fee,
			}
		}

		shopsResponse = append(shopsResponse, shopReponse)
	}

	var paymentMethods []paymentMethodResponse
	for _, pM := range result.PaymentMethods {
		paymentMethod := paymentMethodResponse{
			ID:          pM.ID,
			Name:        pM.Name,
			Type:        pM.Type,
			Description: pM.Description,
			Fee:         pM.Fee,
			Subtotal:    pM.Subtotal,
			Total:       pM.Total,
		}

		paymentMethods = append(paymentMethods, paymentMethod)
	}

	var selectedPayment *paymentMethodResponse
	if result.SelectedPaymentMethod != nil {
		selectedPayment = &paymentMethodResponse{
			ID:          result.SelectedPaymentMethod.ID,
			Name:        result.SelectedPaymentMethod.Name,
			Type:        result.SelectedPaymentMethod.Type,
			Description: result.SelectedPaymentMethod.Description,
			Fee:         result.SelectedPaymentMethod.Fee,
			Subtotal:    result.SelectedPaymentMethod.Subtotal,
			Total:       result.SelectedPaymentMethod.Total,
		}
	}

	resp := checkoutCalculateResponse{
		Address: checkoutAddressResponse{
			ID:            result.Address.ID,
			RecipientName: result.Address.RecipientName,
			Phone:         result.Address.Phone,
			FullAddress:   result.Address.FullAddress,
		},
		Shops:                  shopsResponse,
		TotalShipping:          result.TotalShippingFee,
		Subtotal:               result.Subtotal,
		SelectedPaymentMethods: *selectedPayment,
	}

	if result.TotalAll > 0 {
		resp.TotalAll = &result.TotalAll
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *CartHandler) parseCheckoutInput(
	req checkoutCalculateRequest,
) (usecase.CheckoutInput, error) {
	var paymentMethodID *uuid.UUID
	if req.PaymentMethodID != nil {
		parsed, err := uuid.Parse(*req.PaymentMethodID)
		if err != nil {
			return usecase.CheckoutInput{}, apperrors.NewBadRequest("invalid address id")
		}
		paymentMethodID = &parsed
	}

	var addressID *uuid.UUID
	if req.AddressID != nil {
		parsed, err := uuid.Parse(*req.AddressID)
		if err != nil {
			return usecase.CheckoutInput{}, apperrors.NewBadRequest("invalid address id")
		}
		addressID = &parsed
	}

	var shopInput []usecase.CheckoutShopInput
	for _, shopReq := range req.Shops {
		shopID, err := uuid.Parse(shopReq.ShopID)
		if err != nil {
			return usecase.CheckoutInput{}, apperrors.NewBadRequest("invalid shop id")
		}

		var items []usecase.CheckoutItemInput
		for _, itemReq := range shopReq.Items {
			var productID *uuid.UUID
			var cartItemID *uuid.UUID

			if itemReq.ProductID != nil &&
				*itemReq.ProductID != "" &&
				*itemReq.ProductID != "null" &&
				*itemReq.ProductID != "undefined" {

				if parsed, err := uuid.Parse(*itemReq.ProductID); err == nil {
					productID = &parsed
				} else {
					return usecase.CheckoutInput{}, apperrors.NewBadRequest("invalid product id")
				}
			}

			if itemReq.CartItemID != nil &&
				*itemReq.CartItemID != "" &&
				*itemReq.CartItemID != "null" &&
				*itemReq.CartItemID != "undefined" {

				if parsed, err := uuid.Parse(*itemReq.CartItemID); err == nil {
					cartItemID = &parsed
				}
			}

			if productID == nil && cartItemID == nil {
				return usecase.CheckoutInput{}, apperrors.NewBadRequest("product id or cart item id is required")
			}

			if itemReq.Quantity <= 0 {
				return usecase.CheckoutInput{}, apperrors.NewBadRequest("invalid quantity")
			}

			var opt domain.ItemOptions
			if itemReq.ItemOptions != nil {
				opt = domain.ItemOptions(itemReq.ItemOptions)
			}

			var checkoutProductID uuid.UUID
			if productID != nil {
				checkoutProductID = *productID
			}
			items = append(items, usecase.CheckoutItemInput{
				ProductID:   checkoutProductID,
				CartItemID:  cartItemID,
				ItemOptions: opt.Normalized(),
				Quantity:    itemReq.Quantity,
			})
		}

		var courier *usecase.SelectedCourierInput
		if shopReq.Courier != nil {
			courier = &usecase.SelectedCourierInput{
				Code:    shopReq.Courier.Code,
				Service: shopReq.Courier.Service,
			}
		}

		shopInput = append(shopInput, usecase.CheckoutShopInput{
			ShopID:  shopID,
			Items:   items,
			Courier: courier,
		})
	}

	return usecase.CheckoutInput{
		PaymentMethodID: paymentMethodID,
		AddressID:       addressID,
		ShopInput:       shopInput,
	}, nil
}
