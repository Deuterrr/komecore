package http

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	cartDomain "komecore/internal/modules/cart/domain"
	"komecore/internal/modules/order/usecase"

	"github.com/google/uuid"
)

func (h *orderHandler) Checkout(w http.ResponseWriter, r *http.Request) error {
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
				ShopID:      shop.ShopID,
				Name:        item.ProductName,
				Price:       item.UnitPrice,
				Quantity:    item.Quantity,
				Subtotal:    item.Subtotal,
				ItemOptions: normOpts,
			})
		}

		var shippingResponse []checkoutCouriersResponse
		for _, courier := range shop.CourierOptions {
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
			ID:          pM.PaymentMethodID,
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

	if result.GrandTotal > 0 {
		resp.TotalAll = &result.GrandTotal
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *orderHandler) CheckoutEstimate(w http.ResponseWriter, r *http.Request) error {
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
				ShopID:      shop.ShopID,
				Name:        item.ProductName,
				Price:       item.UnitPrice,
				Quantity:    item.Quantity,
				Subtotal:    item.Subtotal,
				ItemOptions: normOpts,
			})
		}

		shopReponse := shopCalculateResponse{
			ShopID:   shop.ShopID,
			ShopSlug: shop.ShopSlug,
			ShopName: shop.ShopName,
			Subtotal: shop.Subtotal,
			Total:    &shop.Total,
			Items:    itemsResponse,
			SelectedCourier: selectedCourierResponse{
				Code:    shop.SelectedCourier.Code,
				Service: shop.SelectedCourier.Service,
				Fee:     shop.SelectedCourier.Fee,
			},
		}

		shopsResponse = append(shopsResponse, shopReponse)
	}

	var paymentMethods []paymentMethodResponse
	for _, pM := range result.PaymentMethods {
		paymentMethod := paymentMethodResponse{
			ID:          pM.PaymentMethodID,
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
			ID:          result.SelectedPaymentMethod.PaymentMethodID,
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
		Shops:         shopsResponse,
		TotalShipping: result.TotalShippingFee,
		Subtotal:      result.Subtotal,
	}
	if selectedPayment != nil {
		resp.SelectedPaymentMethods = *selectedPayment
	}

	if result.GrandTotal > 0 {
		resp.TotalAll = &result.GrandTotal
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *orderHandler) parseCheckoutInput(
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

			var opt cartDomain.ItemOptions
			if itemReq.ItemOptions != nil {
				opt = cartDomain.ItemOptions(itemReq.ItemOptions)
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
