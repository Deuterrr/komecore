package userhttp

import (
	"encoding/json"
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/user/userusecase"
)

type UserHandler struct {
	service *userusecase.UserService
}

func NewUserHandler(
	service *userusecase.UserService,
) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) error {
	id, err := apphttp.ParamUUID(r, "id")
	if err != nil {
		return apperrors.NewBadRequest("invalid user id")
	}

	result, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		return err
	}
	if result == nil {
		return apperrors.NewNotFound("user not found")
	}

	response := userResponse{
		ID:          result.ID,
		Name:        result.Name,
		Username:    result.Username,
		Phone:       result.Phone,
		AvatarURL:   result.AvatarURL,
		LastLoginAt: result.LastLoginAt,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := apphttp.RequireAuth(r)
	if err != nil {
		return err
	}

	result, err := h.service.GetUserByID(r.Context(), authCtx.UserID)
	if err != nil {
		return err
	}
	if result == nil {
		return apperrors.NewNotFound("user not found")
	}

	response := map[string]userResponse{
		"me": {
			ID:          result.ID,
			Name:        result.Name,
			Username:    result.Username,
			Phone:       result.Phone,
			AvatarURL:   result.AvatarURL,
			LastLoginAt: result.LastLoginAt,
		},
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *UserHandler) GetCurrentProfile(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := apphttp.RequireAuth(r)
	if err != nil {
		return err
	}

	result, err := h.service.GetCurrentProfile(
		r.Context(),
		*authCtx,
	)
	if err != nil {
		return err
	}

	if result == nil {
		return nil
	}

	var response map[string]profileResponse
	if result.Customer != nil {
		response = map[string]profileResponse{
			"profile": {
				CustomerID:  &result.Customer.ID,
				UserID:      result.Customer.UserID,
				Name:        result.Customer.Name,
				Username:    result.Customer.Username,
				Phone:       result.Customer.Phone,
				AvatarURL:   result.Customer.AvatarURL,
				LastLoginAt: result.Customer.LastLoginAt,
				CreatedAt:   result.Customer.CreatedAt,
				UpdatedAt:   result.Customer.UpdatedAt,
			},
		}
	}

	if result.Staff != nil {
		response = map[string]profileResponse{
			"profile": {
				StaffID:     &result.Staff.ID,
				UserID:      result.Staff.UserID,
				Name:        result.Staff.Name,
				Username:    result.Staff.Username,
				Phone:       result.Staff.Phone,
				AvatarURL:   result.Staff.AvatarURL,
				LastLoginAt: result.Staff.LastLoginAt,
				CreatedAt:   result.Staff.CreatedAt,
				UpdatedAt:   result.Staff.UpdatedAt,
			},
		}
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *UserHandler) UpdateCurrentProfile(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := apphttp.RequireAuth(r)
	if err != nil {
		return err
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	result, err := h.service.UpdateCurrentProfile(
		r.Context(),
		*authCtx,
		userusecase.UpdateProfileInput{
			Name:      req.Name,
			Phone:     req.Phone,
			AvatarURL: req.AvatarURL,
		},
	)
	if err != nil {
		return err
	}

	if result == nil {
		return nil
	}

	var response map[string]profileResponse
	if result.Customer != nil {
		response = map[string]profileResponse{
			"profile": {
				CustomerID: &result.Customer.ID,
				UserID:     result.Customer.UserID,
				Name:       result.Customer.Name,
				Username:   result.Customer.Username,
				Phone:      result.Customer.Phone,
				AvatarURL:  result.Customer.AvatarURL,
				CreatedAt:  result.Customer.CreatedAt,
				UpdatedAt:  result.Customer.UpdatedAt,
			},
		}
	}

	if result.Staff != nil {
		response = map[string]profileResponse{
			"profile": {
				StaffID:   &result.Staff.ID,
				UserID:    result.Staff.UserID,
				Name:      result.Staff.Name,
				Username:  result.Staff.Username,
				Phone:     result.Staff.Phone,
				AvatarURL: result.Staff.AvatarURL,
				CreatedAt: result.Staff.CreatedAt,
				UpdatedAt: result.Staff.UpdatedAt,
			},
		}
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}
