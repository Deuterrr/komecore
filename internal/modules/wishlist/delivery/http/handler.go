package http

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/wishlist/usecase"
)

type WishlistHandler struct {
	getWishlist        *usecase.GetWishlistUsecase
	addToWishlist      *usecase.AddToWishlistUsecase
	removeFromWishlist *usecase.RemoveFromWishlistUsecase
}

func NewWishlistHandler(
	getWishlist *usecase.GetWishlistUsecase,
	addToWishlist *usecase.AddToWishlistUsecase,
	removeFromWishlist *usecase.RemoveFromWishlistUsecase,
) *WishlistHandler {
	return &WishlistHandler{
		getWishlist:        getWishlist,
		addToWishlist:      addToWishlist,
		removeFromWishlist: removeFromWishlist,
	}
}

func (h *WishlistHandler) GetWishlist(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	views, err := h.getWishlist.Execute(r.Context(), customerID)
	if err != nil {
		return err
	}

	items := make([]wishlistItemResponse, 0, len(views))
	for _, v := range views {
		items = append(items, wishlistItemResponse{
			ProductID:    v.ProductID,
			SKU:          v.SKU,
			Name:         v.Name,
			Slug:         v.Slug,
			Price:        v.Price,
			PrimaryImage: v.PrimaryImage,
			InStock:      v.InStock,
			TotalStock:   v.TotalStock,
			CreatedAt:    v.CreatedAt,
		})
	}

	apphttp.WriteJSON(w, http.StatusOK, wishlistResponse{
		Items: items,
		Total: len(items),
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

	err = h.addToWishlist.Execute(r.Context(), usecase.AddToWishlistInput{
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

	err = h.removeFromWishlist.Execute(r.Context(), usecase.RemoveFromWishlistInput{
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
