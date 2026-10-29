package inventoryhttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/inventory"
)

type Handler struct {
	service *inventory.Service
}

func New(service *inventory.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) AddInventory(w http.ResponseWriter, r *http.Request) error {
	var req createInventoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.Stock < 0 {
		return apperror.NewBadRequest("invalid stock")
	}

	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	productID, err := httpx.ParamUUID(r, "productID")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	input := inventory.CreateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     req.Stock,
	}
	if err := h.service.CreateInventory(r.Context(), input); err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "inventory successfully added",
	})
	return nil
}

func (h *Handler) UpdateInventory(w http.ResponseWriter, r *http.Request) error {
	var req updateInventoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.Stock < 0 {
		return apperror.NewBadRequest("invalid stock")
	}

	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	productID, err := httpx.ParamUUID(r, "productID")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	input := inventory.UpdateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     req.Stock,
	}
	if err := h.service.UpdateInventory(r.Context(), input); err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "inventory successfully updated",
	})
	return nil
}

func (h *Handler) RemoveInventory(w http.ResponseWriter, r *http.Request) error {
	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	productID, err := httpx.ParamUUID(r, "productID")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	input := inventory.DeleteInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
	}
	if err := h.service.DeleteInventory(r.Context(), input); err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "inventory successfully removed",
	})
	return nil
}
