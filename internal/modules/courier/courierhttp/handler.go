package courierhttp

import (
	"net/http"

	"komecore/internal/httpx"
	"komecore/internal/modules/courier/courierusecase"
)

type CourierHandler struct {
	service *courierusecase.CourierService
}

func NewCourierHandler(
	service *courierusecase.CourierService,
) *CourierHandler {
	return &CourierHandler{
		service: service,
	}
}

func (h *CourierHandler) ListAllCouriers(w http.ResponseWriter, r *http.Request) error {
	codes, err := h.service.ListAllCouriers(r.Context())
	if err != nil {
		return err
	}

	response := map[string][]string{
		"couriers": codes,
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}
