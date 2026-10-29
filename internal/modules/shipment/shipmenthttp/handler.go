package shipmenthttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/shipment"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *shipment.Service
}

func New(service *shipment.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) EstimateShippingOptions(w http.ResponseWriter, r *http.Request) error {
	var req estimateShippingOptionsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}
	if req.Weight <= 0 {
		return apperror.NewBadRequest("invalid weight")
	}

	var shopID uuid.UUID
	if req.ShopID != "" {
		parsed, err := uuid.Parse(req.ShopID)
		if err == nil {
			shopID = parsed
		}
	}

	input := shipment.EstimateShippingOptionsInput{
		ShopID:      shopID,
		Origin:      req.Origin,
		Destination: req.Destination,
		Weight:      req.Weight,
		PriceFilter: req.PriceFilter,
	}
	results, err := h.service.EstimateShippingOptions(r.Context(), input)
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
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *Handler) UpdateShipmentStatus(w http.ResponseWriter, r *http.Request) error {
	shipmentIDStr := chi.URLParam(r, "shipmentID")
	shipmentID, err := uuid.Parse(shipmentIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid shipment id")
	}

	var req updateShipmentStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.Status == "" {
		return apperror.NewBadRequest("status is required")
	}

	input := shipment.UpdateShipmentStatusInput{
		ShipmentID:  shipmentID,
		Status:      shipment.ShipmentStatus(req.Status),
		Description: req.Description,
		Location:    req.Location,
	}
	res, err := h.service.UpdateShipmentStatus(r.Context(), input)
	if err != nil {
		return err
	}

	resp := buildShipmentResponse(res.Shipment)
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) UpdateShipment(w http.ResponseWriter, r *http.Request) error {
	shipmentIDStr := chi.URLParam(r, "shipmentID")
	shipmentID, err := uuid.Parse(shipmentIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid shipment id")
	}

	var req updateShipmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	input := shipment.UpdateShipmentInput{
		ShipmentID:     shipmentID,
		TrackingNumber: req.TrackingNumber,
		Courier:        req.Courier,
		Service:        req.Service,
	}
	res, err := h.service.UpdateShipment(r.Context(), input)
	if err != nil {
		return err
	}

	resp := buildShipmentResponse(res.Shipment)
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) DispatchShipment(w http.ResponseWriter, r *http.Request) error {
	shipmentIDStr := chi.URLParam(r, "shipmentID")
	shipmentID, err := uuid.Parse(shipmentIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid shipment id")
	}

	var req dispatchShipmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.TrackingNumber == "" {
		return apperror.NewBadRequest("tracking_number is required")
	}

	ship, err := h.service.Dispatch(r.Context(), shipmentID, req.TrackingNumber)
	if err != nil {
		return err
	}

	resp := buildShipmentResponse(ship)
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func buildShipmentResponse(s *shipment.Shipment) shipmentResponse {
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
