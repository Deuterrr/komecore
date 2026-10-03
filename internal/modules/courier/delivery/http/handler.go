package http

import (
	"net/http"

	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/courier/usecase"
)

type CourierHandler struct {
	listCouriers *usecase.ListCouriersUsecase
}

func NewCourierHandler(
	listCouriers *usecase.ListCouriersUsecase,
) *CourierHandler {
	return &CourierHandler{
		listCouriers: listCouriers,
	}
}

func (h *CourierHandler) ListAllCouriers(w http.ResponseWriter, r *http.Request) error {
	codes, err := h.listCouriers.Execute(r.Context())
	if err != nil {
		return err
	}

	response := map[string][]string{
		"couriers": codes,
	}
	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}
