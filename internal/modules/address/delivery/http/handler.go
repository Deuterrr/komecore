package http

import (
	"net/http"
	"strconv"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/address/usecase"

	"github.com/google/uuid"
)

type AddressHandler struct {
	service *usecase.AddressService
}

func NewAddressHandler(
	service *usecase.AddressService,
) *AddressHandler {
	return &AddressHandler{
		service: service,
	}
}

func (h *AddressHandler) ListUserAddresses(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
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

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) SaveUserAddress(w http.ResponseWriter, r *http.Request) error {
	var req saveCustomerAddressRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}

	prov := req.GetProvince()
	if prov == "" {
		return apperrors.NewBadRequest("invalid province")
	}
	city := req.GetCity()
	if city == "" {
		return apperrors.NewBadRequest("invalid city")
	}
	dist := req.GetDistrict()
	if dist == "" {
		return apperrors.NewBadRequest("invalid district")
	}
	if req.FullAddress == "" {
		return apperrors.NewBadRequest("invalid full address")
	}
	if req.PostalCode == "" {
		return apperrors.NewBadRequest("invalid postal code")
	}

	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	var addressID *uuid.UUID
	if req.AddressID != nil {
		parsed, err := uuid.Parse(*req.AddressID)
		if err != nil {
			return apperrors.NewBadRequest("invalid address id")
		}
		addressID = &parsed
	}

	var parsedIsDefault = false
	if req.IsDefault != nil && *req.IsDefault != "" {
		parsed, err := strconv.ParseBool(*req.IsDefault)
		if err != nil {
			return apperrors.NewBadRequest("invalid default status")
		}
		parsedIsDefault = parsed
	}

	input := usecase.SaveCustomerAddressInput{
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

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) DeleteUserAddress(w http.ResponseWriter, r *http.Request) error {
	_, _, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	addressID, err := apphttp.ParamUUID(r, "addressID")
	if err != nil {
		return apperrors.NewBadRequest("invalid address id")
	}

	err = h.service.DeleteCustomerAddress(r.Context(), addressID)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address deleted successfully",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) ListShopAddresses(w http.ResponseWriter, r *http.Request) error {
	shopID, err := apphttp.ParamUUID(r, "id")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
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

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) CreateShopAddress(w http.ResponseWriter, r *http.Request) error {
	var req createShopAddressRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	prov := req.GetProvince()
	if prov == "" {
		return apperrors.NewBadRequest("invalid province")
	}
	city := req.GetCity()
	if city == "" {
		return apperrors.NewBadRequest("invalid city")
	}
	dist := req.GetDistrict()
	if dist == "" {
		return apperrors.NewBadRequest("invalid district")
	}

	if req.FullAddress == "" {
		return apperrors.NewBadRequest("invalid full address")
	}
	if req.PostalCode == "" {
		return apperrors.NewBadRequest("invalid postal code")
	}

	parsedShopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	var parsedIsActive bool
	parsedIsActive, err = strconv.ParseBool(req.IsActive)
	if err != nil {
		return apperrors.NewBadRequest("invalid active status")
	}

	input := usecase.CreateShopAddressInput{
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

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) UpdateShopAddress(w http.ResponseWriter, r *http.Request) error {
	var req createShopAddressRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	prov := req.GetProvince()
	if prov == "" {
		return apperrors.NewBadRequest("invalid province")
	}
	city := req.GetCity()
	if city == "" {
		return apperrors.NewBadRequest("invalid city")
	}
	dist := req.GetDistrict()
	if dist == "" {
		return apperrors.NewBadRequest("invalid district")
	}
	if req.FullAddress == "" {
		return apperrors.NewBadRequest("invalid full address")
	}
	if req.PostalCode == "" {
		return apperrors.NewBadRequest("invalid postal code")
	}

	parsedShopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	addressID, err := apphttp.ParamUUID(r, "addressID")
	if err != nil {
		return apperrors.NewBadRequest("invalid address id")
	}

	var parsedIsActive bool
	parsedIsActive, err = strconv.ParseBool(req.IsActive)
	if err != nil {
		return apperrors.NewBadRequest("invalid active status")
	}

	input := usecase.UpdateShopAddressInput{
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

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *AddressHandler) DeleteShopAddress(w http.ResponseWriter, r *http.Request) error {
	addressID, err := apphttp.ParamUUID(r, "addressID")
	if err != nil {
		return apperrors.NewBadRequest("invalid address id")
	}

	shopID, err := apphttp.ParamUUID(r, "shopID")
	if err != nil {
		return apperrors.NewBadRequest("invalid shop id")
	}

	err = h.service.DeleteShopAddress(r.Context(), shopID, addressID)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "address deleted successfully",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}
