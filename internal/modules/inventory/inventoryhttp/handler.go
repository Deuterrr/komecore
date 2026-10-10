package inventoryhttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/inventory/inventoryusecase"
)

type InventoryHandler struct {
	service *inventoryusecase.InventoryService
}

func NewInventoryHandler(
	service *inventoryusecase.InventoryService,
) *InventoryHandler {
	return &InventoryHandler{
		service: service,
	}
}

func (h *InventoryHandler) AddInventory(w http.ResponseWriter, r *http.Request) error {
	var req createInventoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	productID, err := httpx.ParamUUID(r, "productID")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	if req.Stock < 0 {
		return apperror.NewBadRequest("invalid stock")
	}

	input := inventoryusecase.CreateInventoryInput{
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

func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) error {
	var req updateInventoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	productID, err := httpx.ParamUUID(r, "productID")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	if req.Stock < 0 {
		return apperror.NewBadRequest("invalid stock")
	}

	input := inventoryusecase.UpdateInventoryInput{
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

func (h *InventoryHandler) RemoveInventory(w http.ResponseWriter, r *http.Request) error {
	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	productID, err := httpx.ParamUUID(r, "productID")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	input := inventoryusecase.DeleteInventoryInput{
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
