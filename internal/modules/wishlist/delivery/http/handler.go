package http

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/wishlist/usecase"
)

type WishlistHandler struct {
	service *usecase.WishlistService
}

func NewWishlistHandler(service *usecase.WishlistService) *WishlistHandler {
	return &WishlistHandler{
		service: service,
	}
}

func (h *WishlistHandler) GetWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	views, err := h.service.GetWishlist(r.Context(), customerID)
	if err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusOK, wishlistResponse{
		Items: views,
		Total: len(views),
	})
	return nil
}

func (h *WishlistHandler) AddToWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := apphttp.ParamUUID(r, "productId")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	err = h.service.AddToWishlist(r.Context(), usecase.AddToWishlistInput{
		CustomerID: customerID,
		ProductID:  productID,
	})
	if err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusCreated, map[string]any{
		"message":    "product added to wishlist successfully",
		"product_id": productID,
	})
	return nil
}

func (h *WishlistHandler) RemoveFromWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := apphttp.ParamUUID(r, "productId")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	err = h.service.RemoveFromWishlist(r.Context(), usecase.RemoveFromWishlistInput{
		CustomerID: customerID,
		ProductID:  productID,
	})
	if err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusOK, messageResponse{
		Message: "product removed from wishlist successfully",
	})
	return nil
}
