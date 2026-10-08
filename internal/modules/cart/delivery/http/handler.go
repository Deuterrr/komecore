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
}

func NewCartHandler(
	addItem *usecase.AddItemUsecase,
	getCart *usecase.GetCartUsecase,
	updateItem *usecase.UpdateItemUsecase,
	removeItem *usecase.RemoveItemUsecase,
) *CartHandler {
	return &CartHandler{
		addItem:    addItem,
		getCart:    getCart,
		updateItem: updateItem,
		removeItem: removeItem,
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
