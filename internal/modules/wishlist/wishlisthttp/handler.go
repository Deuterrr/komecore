package wishlisthttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/wishlist"
)

type Handler struct {
	service *wishlist.Service
}

func New(service *wishlist.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	views, err := h.service.GetWishlist(r.Context(), customerID)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, wishlistResponse{
		Items: views,
		Total: len(views),
	})
	return nil
}

func (h *Handler) AddToWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := httpx.ParamUUID(r, "productId")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	err = h.service.AddToWishlist(r.Context(), wishlist.AddToWishlistInput{
		CustomerID: customerID,
		ProductID:  productID,
	})
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"message":    "product added to wishlist successfully",
		"product_id": productID,
	})
	return nil
}

func (h *Handler) RemoveFromWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := httpx.ParamUUID(r, "productId")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	err = h.service.RemoveFromWishlist(r.Context(), wishlist.RemoveFromWishlistInput{
		CustomerID: customerID,
		ProductID:  productID,
	})
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, messageResponse{
		Message: "product removed from wishlist successfully",
	})
	return nil
}
