package shophttp

import (
	"net/http"
	"strconv"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/httpx"
	"komecore/internal/modules/shop/shopdomain"
	"komecore/internal/modules/shop/shopusecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ShopHandler struct {
	service *shopusecase.ShopService
}

func NewShopHandler(
	service *shopusecase.ShopService,
) *ShopHandler {
	return &ShopHandler{
		service: service,
	}
}

func (h *ShopHandler) resolveShopID(r *http.Request) (uuid.UUID, error) {
	param := chi.URLParam(r, "shopID")
	if param == "" {
		param = chi.URLParam(r, "id")
	}
	if param == "" {
		return uuid.Nil, apperror.NewBadRequest("invalid shop id")
	}

	if parsed, err := uuid.Parse(param); err == nil {
		return parsed, nil
	}

	if h.service == nil {
		return uuid.Nil, apperror.NewNotFound("shop not found")
	}

	shop, err := h.service.GetBySlug(r.Context(), param)
	if err != nil {
		return uuid.Nil, err
	}
	if shop == nil {
		return uuid.Nil, apperror.NewNotFound("shop not found")
	}

	return shop.ID, nil
}

func (h *ShopHandler) FindShops(w http.ResponseWriter, r *http.Request) error {
	page := httpx.QueryIntDefault(r, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := httpx.QueryIntDefault(r, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	name := httpx.Query(r, "name")
	id := httpx.Query(r, "id")
	sort := httpx.Query(r, "sort")
	activeParam := httpx.Query(r, "active")
	approvalParam := httpx.Query(r, "approval_status")

	input := shopusecase.FindShopsInput{
		Page:  page,
		Limit: limit,
		Sort:  sort,
	}
	if name != "" {
		input.Name = &name
	}
	if id != "" {
		input.ID = &id
	}
	if activeParam != "" {
		if activeBool, err := strconv.ParseBool(activeParam); err == nil {
			input.IsActive = &activeBool
		}
	}
	if approvalParam != "" {
		input.ApprovalStatus = &approvalParam
	}

	actor, ok := authctx.GetActor(r.Context())
	if ok && actor.StaffID != nil && !actor.IsSuperAdmin() {
		allAssigned := actor.GetAssignedShopIDs()
		var assignedIDs []uuid.UUID
		for _, sID := range allAssigned {
			if actor.HasPermission(sID, authctx.PermissionShopView) || actor.HasPermission(sID, authctx.PermissionOrderRead) {
				assignedIDs = append(assignedIDs, sID)
			}
		}

		if len(assignedIDs) == 0 {
			res := listShopsResponse{
				Page:  page,
				Limit: limit,
				Total: 0,
				Shops: []getShopResponse{},
			}
			httpx.WriteJSON(w, http.StatusOK, res)
			return nil
		}
		input.ShopIDs = assignedIDs
	}

	shops, total, err := h.service.FindShops(r.Context(), input)
	if err != nil {
		return err
	}

	var shopsResponse []getShopResponse
	for _, shop := range shops {
		s := getShopResponse{
			ID:             shop.ID,
			Name:           shop.Name,
			Slug:           shop.Slug,
			Description:    shop.Description,
			IsActive:       shop.IsActive,
			ApprovalStatus: string(shop.ApprovalStatus),
			CreatedAt:      shop.CreatedAt,
			UpdatedAt:      shop.UpdatedAt,
		}

		shopsResponse = append(shopsResponse, s)
	}

	response := map[string]any{
		"shops": shopsResponse,
		"page":  page,
		"limit": limit,
		"total": total,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ShopHandler) GetShopByID(w http.ResponseWriter, r *http.Request) error {
	param := chi.URLParam(r, "shopID")
	if param == "" {
		param = chi.URLParam(r, "id")
	}
	if param == "" {
		return apperror.NewBadRequest("invalid shop id")
	}

	var result *shopdomain.Shop
	if parsed, err := uuid.Parse(param); err == nil {
		var getErr error
		result, getErr = h.service.GetByID(r.Context(), parsed)
		if getErr != nil {
			return getErr
		}
	} else {
		var getErr error
		result, getErr = h.service.GetBySlug(r.Context(), param)
		if getErr != nil {
			return getErr
		}
	}

	if result == nil {
		return apperror.NewNotFound("shop not found")
	}

	response := map[string]getShopResponse{
		"shop": {
			ID:             result.ID,
			Name:           result.Name,
			Slug:           result.Slug,
			Description:    result.Description,
			IsActive:       result.IsActive,
			ApprovalStatus: string(result.ApprovalStatus),
			CreatedAt:      result.CreatedAt,
			UpdatedAt:      result.UpdatedAt,
		},
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ShopHandler) SaveShop(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}

	var req saveShopRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return apperror.NewBadRequest("invalid name")
	}

	var shopID *uuid.UUID
	if req.ShopID != nil && *req.ShopID != "" {
		parsed, err := uuid.Parse(*req.ShopID)
		if err != nil {
			return apperror.NewBadRequest("invalid shop id")
		}

		shopID = &parsed
	}

	var isActivePtr *bool
	if req.IsActive != nil && *req.IsActive != "" {
		parsedIsActive, err := strconv.ParseBool(*req.IsActive)
		if err != nil {
			return apperror.NewBadRequest("invalid active status")
		}
		isActivePtr = &parsedIsActive
	}

	input := shopusecase.SaveShopInput{
		ID:             shopID,
		Name:           req.Name,
		Description:    req.Description,
		IsActive:       isActivePtr,
		ApprovalStatus: req.ApprovalStatus,
	}

	err := h.service.SaveShop(
		r.Context(),
		*actor,
		input,
	)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "shop successfully saved",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ShopHandler) GetShopAddresses(w http.ResponseWriter, r *http.Request) error {
	shopID, err := h.resolveShopID(r)
	if err != nil {
		return err
	}

	result, err := h.service.GetShopAddresses(r.Context(), shopID)
	if err != nil {
		return err
	}

	addresses := make([]shopAddressResponse, 0, len(result))
	for _, a := range result {
		addresses = append(addresses, shopAddressResponse{
			ID:          a.ID,
			Label:       a.Label,
			Phone:       a.Phone,
			IsActive:    a.IsActive,
			Province:    a.Detail.Province,
			City:        a.Detail.City,
			District:    a.Detail.District,
			FullAddress: a.Detail.FullAddress,
			PostalCode:  a.Detail.PostalCode,
			Latitude:    a.Detail.Latitude,
			Longitude:   a.Detail.Longitude,
			CreatedAt:   a.CreatedAt,
			UpdatedAt:   a.UpdatedAt,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"shop_id":   shopID,
		"addresses": addresses,
	})
	return nil
}

func (h *ShopHandler) GetShopProducts(w http.ResponseWriter, r *http.Request) error {
	shopID, err := h.resolveShopID(r)
	if err != nil {
		return err
	}

	result, err := h.service.GetShopProducts(r.Context(), shopID)
	if err != nil {
		return err
	}

	products := make([]shopProductResponse, 0, len(result))
	for _, item := range result {
		products = append(products, shopProductResponse{
			ID:          item.Product.ID,
			SKU:         item.Product.SKU,
			Name:        item.Product.Name,
			Slug:        item.Product.Slug,
			Description: item.Product.Description,
			Status:      string(item.Product.Status),
			Price:       item.Product.Price,
			Weight:      item.Product.Weight,
			Inventory: shopProductInventoryResponse{
				TotalStock:    item.Inventory.TotalStock,
				ReservedStock: item.Inventory.ReservedStock,
				Available:     item.Inventory.Available(),
			},
			CreatedAt: item.Product.CreatedAt,
			UpdatedAt: item.Product.UpdatedAt,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"shop_id":  shopID,
		"products": products,
	})
	return nil
}

func (h *ShopHandler) DeleteShop(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}

	shopID, err := h.resolveShopID(r)
	if err != nil {
		return err
	}

	if err := h.service.DeleteShop(r.Context(), *actor, shopID); err != nil {
		return err
	}

	response := map[string]string{
		"message": "shop successfully deleted",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}
