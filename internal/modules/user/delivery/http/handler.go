package http

import (
	"encoding/json"
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	authendomain "komecore/internal/modules/auth/domain"
	"komecore/internal/modules/user/usecase"
)

type UserHandler struct {
	getUser       *usecase.GetUserUsecase
	getProfile    *usecase.GetCurrentProfileUsecase
	updateProfile *usecase.UpdateCurrentProfileUsecase
}

func NewUserHandler(
	getUser *usecase.GetUserUsecase,
	getProfile *usecase.GetCurrentProfileUsecase,
	updateProfile *usecase.UpdateCurrentProfileUsecase,
) *UserHandler {
	return &UserHandler{
		getUser:       getUser,
		getProfile:    getProfile,
		updateProfile: updateProfile,
	}
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) error {
	id, err := apphttp.ParamUUID(r, "id")
	if err != nil {
		return apperrors.NewBadRequest("invalid user id")
	}

	result, err := h.getUser.ByID(r.Context(), id)
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
	authCtx, ok := authendomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	result, err := h.getUser.ByID(r.Context(), authCtx.UserID)
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
	authCtx, ok := authendomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	result, err := h.getProfile.Execute(
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
	authCtx, ok := authendomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	result, err := h.updateProfile.Execute(
		r.Context(),
		*authCtx,
		usecase.UpdateProfileInput{
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
