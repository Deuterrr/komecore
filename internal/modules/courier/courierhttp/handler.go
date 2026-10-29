package courierhttp

import (
	"net/http"

	"komecore/internal/httpx"
	"komecore/internal/modules/courier"
)

type Handler struct {
	service *courier.Service
}

func New(service *courier.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListAllCouriers(w http.ResponseWriter, r *http.Request) error {
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
