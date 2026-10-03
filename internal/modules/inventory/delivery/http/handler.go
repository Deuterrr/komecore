package http

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/inventory/usecase"
)

type InventoryHandler struct {
	createInventory *usecase.CreateInventoryUsecase
	updateInventory *usecase.UpdateInventoryUsecase
	deleteInventory *usecase.DeleteInventoryUsecase
}

func NewInventoryHandler(
	createInventory *usecase.CreateInventoryUsecase,
	updateInventory *usecase.UpdateInventoryUsecase,
	deleteInventory *usecase.DeleteInventoryUsecase,
) *InventoryHandler {
	return &InventoryHandler{
		createInventory: createInventory,
		updateInventory: updateInventory,
		deleteInventory: deleteInventory,
	}
}

func (h *InventoryHandler) AddInventory(w http.ResponseWriter, r *http.Request) error {
	var req createInventoryRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	shopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	productID, err := apphttp.ParamUUID(r, "productID")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	if req.Stock < 0 {
		return apperrors.NewBadRequest("invalid stock")
	}

	input := usecase.CreateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     req.Stock,
	}
	if err := h.createInventory.Execute(r.Context(), input); err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "inventory successfully added",
	})
	return nil
}

func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) error {
	var req updateInventoryRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	shopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	productID, err := apphttp.ParamUUID(r, "productID")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	if req.Stock < 0 {
		return apperrors.NewBadRequest("invalid stock")
	}

	input := usecase.UpdateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     req.Stock,
	}
	if err := h.updateInventory.Execute(r.Context(), input); err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "inventory successfully updated",
	})
	return nil
}

func (h *InventoryHandler) RemoveInventory(w http.ResponseWriter, r *http.Request) error {
	shopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	productID, err := apphttp.ParamUUID(r, "productID")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	input := usecase.DeleteInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
	}
	if err := h.deleteInventory.Execute(r.Context(), input); err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "inventory successfully removed",
	})
	return nil
}
