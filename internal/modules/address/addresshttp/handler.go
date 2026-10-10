package addresshttp

import (
	"net/http"
	"strconv"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/address/addressusecase"

	"github.com/google/uuid"
)

type AddressHandler struct {
	service *addressusecase.AddressService
}

func NewAddressHandler(
	service *addressusecase.AddressService,
) *AddressHandler {
	return &AddressHandler{
		service: service,
	}
}

func (h *AddressHandler) ListUserAddresses(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	addresses, err := h.service.ListCustomerAddresses(r.Context(), customerID)
	if err != nil {
		return err
	}

	result := make([]customerAddressResponse, 0, len(addresses))
	for _, r := range addresses {
		address := customerAddressResponse{
			AddressID:    r.ID,
			CustomerID:   r.CustomerID,
			ReceiverName: r.ReceiverName,
			Phone:        r.Phone,
			IsDefault:    r.IsDefault,
			Province:     r.Detail.Province,
			ProvinceID:   r.Detail.Province,
			City:         r.Detail.City,
			CityID:       r.Detail.City,
			District:     r.Detail.District,
			DistrictID:   r.Detail.District,
			FullAddress:  r.Detail.FullAddress,
			PostalCode:   r.Detail.PostalCode,
			Latitude:     r.Detail.Latitude,
			Longitude:    r.Detail.Longitude,
			UpdatedAt:    r.UpdatedAt,
		}
		result = append(result, address)
	}

	response := map[string][]customerAddressResponse{
		"addresses": result,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) SaveUserAddress(w http.ResponseWriter, r *http.Request) error {
	var req saveCustomerAddressRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	prov := req.GetProvince()
	if prov == "" {
		return apperror.NewBadRequest("invalid province")
	}
	city := req.GetCity()
	if city == "" {
		return apperror.NewBadRequest("invalid city")
	}
	dist := req.GetDistrict()
	if dist == "" {
		return apperror.NewBadRequest("invalid district")
	}
	if req.FullAddress == "" {
		return apperror.NewBadRequest("invalid full address")
	}
	if req.PostalCode == "" {
		return apperror.NewBadRequest("invalid postal code")
	}

	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	var addressID *uuid.UUID
	if req.AddressID != nil {
		parsed, err := uuid.Parse(*req.AddressID)
		if err != nil {
			return apperror.NewBadRequest("invalid address id")
		}
		addressID = &parsed
	}

	var parsedIsDefault = false
	if req.IsDefault != nil && *req.IsDefault != "" {
		parsed, err := strconv.ParseBool(*req.IsDefault)
		if err != nil {
			return apperror.NewBadRequest("invalid default status")
		}
		parsedIsDefault = parsed
	}

	input := addressusecase.SaveCustomerAddressInput{
		ID:           addressID,
		CustomerID:   customerID,
		ReceiverName: req.ReceiverName,
		Phone:        req.Phone,
		IsDefault:    &parsedIsDefault,
		Province:     prov,
		City:         city,
		District:     dist,
		FullAddress:  req.FullAddress,
		PostalCode:   req.PostalCode,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
	}

	err = h.service.SaveCustomerAddress(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address saved successfully",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) DeleteUserAddress(w http.ResponseWriter, r *http.Request) error {
	_, _, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	addressID, err := httpx.ParamUUID(r, "addressID")
	if err != nil {
		return apperror.NewBadRequest("invalid address id")
	}

	err = h.service.DeleteCustomerAddress(r.Context(), addressID)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address deleted successfully",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) ListShopAddresses(w http.ResponseWriter, r *http.Request) error {
	shopID, err := httpx.ParamUUID(r, "id")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	result, err := h.service.ListShopAddresses(r.Context(), shopID)
	if err != nil {
		return err
	}

	addresses := make([]shopAddressResponse, 0, len(result))
	for _, r := range result {
		address := shopAddressResponse{
			ShopID:      r.ShopID,
			Label:       r.Label,
			Phone:       r.Phone,
			IsActive:    r.IsActive,
			Province:    r.Detail.Province,
			ProvinceID:  r.Detail.Province,
			City:        r.Detail.City,
			CityID:      r.Detail.City,
			District:    r.Detail.District,
			DistrictID:  r.Detail.District,
			FullAddress: r.Detail.FullAddress,
			PostalCode:  r.Detail.PostalCode,
			Latitude:    r.Detail.Latitude,
			Longitude:   r.Detail.Longitude,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		}
		addresses = append(addresses, address)
	}

	response := map[string][]shopAddressResponse{
		"addresses": addresses,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) CreateShopAddress(w http.ResponseWriter, r *http.Request) error {
	var req createShopAddressRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	prov := req.GetProvince()
	if prov == "" {
		return apperror.NewBadRequest("invalid province")
	}
	city := req.GetCity()
	if city == "" {
		return apperror.NewBadRequest("invalid city")
	}
	dist := req.GetDistrict()
	if dist == "" {
		return apperror.NewBadRequest("invalid district")
	}

	if req.FullAddress == "" {
		return apperror.NewBadRequest("invalid full address")
	}
	if req.PostalCode == "" {
		return apperror.NewBadRequest("invalid postal code")
	}

	parsedShopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	var parsedIsActive bool
	parsedIsActive, err = strconv.ParseBool(req.IsActive)
	if err != nil {
		return apperror.NewBadRequest("invalid active status")
	}

	input := addressusecase.CreateShopAddressInput{
		ShopID:      parsedShopID,
		Label:       req.Label,
		Phone:       req.Phone,
		IsActive:    &parsedIsActive,
		Province:    prov,
		City:        city,
		District:    dist,
		FullAddress: req.FullAddress,
		PostalCode:  req.PostalCode,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
	}

	err = h.service.CreateShopAddress(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address successfully created",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) UpdateShopAddress(w http.ResponseWriter, r *http.Request) error {
	var req createShopAddressRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	prov := req.GetProvince()
	if prov == "" {
		return apperror.NewBadRequest("invalid province")
	}
	city := req.GetCity()
	if city == "" {
		return apperror.NewBadRequest("invalid city")
	}
	dist := req.GetDistrict()
	if dist == "" {
		return apperror.NewBadRequest("invalid district")
	}
	if req.FullAddress == "" {
		return apperror.NewBadRequest("invalid full address")
	}
	if req.PostalCode == "" {
		return apperror.NewBadRequest("invalid postal code")
	}

	parsedShopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	addressID, err := httpx.ParamUUID(r, "addressID")
	if err != nil {
		return apperror.NewBadRequest("invalid address id")
	}

	var parsedIsActive bool
	parsedIsActive, err = strconv.ParseBool(req.IsActive)
	if err != nil {
		return apperror.NewBadRequest("invalid active status")
	}

	input := addressusecase.UpdateShopAddressInput{
		ID:          addressID,
		ShopID:      parsedShopID,
		Label:       req.Label,
		Phone:       req.Phone,
		IsActive:    &parsedIsActive,
		Province:    prov,
		City:        city,
		District:    dist,
		FullAddress: req.FullAddress,
		PostalCode:  req.PostalCode,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
	}

	err = h.service.UpdateShopAddress(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address updated successfully",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) DeleteShopAddress(w http.ResponseWriter, r *http.Request) error {
	addressID, err := httpx.ParamUUID(r, "addressID")
	if err != nil {
		return apperror.NewBadRequest("invalid address id")
	}

	shopID, err := httpx.ParamUUID(r, "shopID")
	if err != nil {
		return apperror.NewBadRequest("invalid shop id")
	}

	err = h.service.DeleteShopAddress(r.Context(), shopID, addressID)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address deleted successfully",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}
