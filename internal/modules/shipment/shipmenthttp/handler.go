package shipmenthttp

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/shipment/shipmentdomain"
	"komecore/internal/modules/shipment/shipmentusecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ShipmentHandler struct {
	estimateShippOpts    *shipmentusecase.EstimateShippingOptionsUsecase
	updateShipmentStatus *shipmentusecase.UpdateShipmentStatusUsecase
	updateShipment       *shipmentusecase.UpdateShipmentUsecase
}

func NewShipmentHandler(
	estimateShippOpts *shipmentusecase.EstimateShippingOptionsUsecase,
	updateShipmentStatus *shipmentusecase.UpdateShipmentStatusUsecase,
	updateShipment *shipmentusecase.UpdateShipmentUsecase,
) *ShipmentHandler {
	return &ShipmentHandler{
		estimateShippOpts:    estimateShippOpts,
		updateShipmentStatus: updateShipmentStatus,
		updateShipment:       updateShipment,
	}
}

func (h *ShipmentHandler) EstimateShippingOptions(w http.ResponseWriter, r *http.Request) error {
	var req estimateShippingOptionsRequest

	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}

	var shopID uuid.UUID
	if req.ShopID != "" {
		parsed, err := uuid.Parse(req.ShopID)
		if err == nil {
			shopID = parsed
		}
	}

	if req.Weight <= 0 {
		return apperrors.NewBadRequest("invalid weight")
	}

	input := shipmentusecase.EstimateShippingOptionsInput{
		ShopID:      shopID,
		Origin:      req.Origin,
		Destination: req.Destination,
		Weight:      req.Weight,
		PriceFilter: req.PriceFilter,
	}

	results, err := h.estimateShippOpts.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	couriers := make([]estimateShippingOptionsResponse, 0, len(results))
	for _, result := range results {
		option := estimateShippingOptionsResponse{
			Name:                   result.Name,
			Code:                   result.Code,
			Service:                result.Service,
			Description:            result.Description,
			Cost:                   result.Cost,
			Etd:                    result.Etd,
			EstimatedDurationHours: result.EstimatedDurationHours,
			SLAConfidenceScore:     result.SLAConfidenceScore,
			DeliveryStatus:         result.DeliveryStatus,
		}

		couriers = append(couriers, option)
	}

	response := map[string]any{
		"couriers": couriers,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ShipmentHandler) UpdateShipmentStatus(w http.ResponseWriter, r *http.Request) error {
	shipmentIDStr := chi.URLParam(r, "shipmentID")
	shipmentID, err := uuid.Parse(shipmentIDStr)
	if err != nil {
		return apperrors.NewBadRequest("invalid shipment id")
	}

	var req updateShipmentStatusRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	if req.Status == "" {
		return apperrors.NewBadRequest("status is required")
	}

	input := shipmentusecase.UpdateShipmentStatusInput{
		ShipmentID:  shipmentID,
		Status:      shipmentdomain.ShipmentStatus(req.Status),
		Description: req.Description,
		Location:    req.Location,
	}

	res, err := h.updateShipmentStatus.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	resp := buildShipmentResponse(res.Shipment)

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *ShipmentHandler) UpdateShipment(w http.ResponseWriter, r *http.Request) error {
	shipmentIDStr := chi.URLParam(r, "shipmentID")
	shipmentID, err := uuid.Parse(shipmentIDStr)
	if err != nil {
		return apperrors.NewBadRequest("invalid shipment id")
	}

	var req updateShipmentRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	input := shipmentusecase.UpdateShipmentInput{
		ShipmentID:     shipmentID,
		TrackingNumber: req.TrackingNumber,
		Courier:        req.Courier,
		Service:        req.Service,
	}

	res, err := h.updateShipment.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	resp := buildShipmentResponse(res.Shipment)
	apphttp.WriteJSON(w, http.StatusOK, resp)

	return nil
}

func (h *ShipmentHandler) DispatchShipment(w http.ResponseWriter, r *http.Request) error {
	shipmentIDStr := chi.URLParam(r, "shipmentID")
	shipmentID, err := uuid.Parse(shipmentIDStr)
	if err != nil {
		return apperrors.NewBadRequest("invalid shipment id")
	}

	var req dispatchShipmentRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}
	if req.TrackingNumber == "" {
		return apperrors.NewBadRequest("tracking_number is required")
	}

	shipment, err := h.updateShipment.Dispatch(r.Context(), shipmentID, req.TrackingNumber)
	if err != nil {
		return err
	}

	resp := buildShipmentResponse(shipment)
	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func buildShipmentResponse(s *shipmentdomain.Shipment) shipmentResponse {
	return shipmentResponse{
		ID:                s.ID.String(),
		OrderID:           s.OrderID.String(),
		Status:            string(s.Status),
		FulfillmentMethod: string(s.FulfillmentMethod),
		TrackingNumber:    s.TrackingNumber,
		Courier:           s.Courier,
		Service:           s.Service,
		Cost:              s.Cost,
		Weight:            s.Weight,
		CreatedAt:         s.CreatedAt,
	}
}
