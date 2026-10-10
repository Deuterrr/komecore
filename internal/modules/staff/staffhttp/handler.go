package staffhttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/httpx"
	"komecore/internal/modules/staff/staffusecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type StaffHandler = staffHandler

type staffHandler struct {
	service *staffusecase.StaffService
}

func NewStaffHandler(
	service *staffusecase.StaffService,
) *staffHandler {
	return &staffHandler{
		service: service,
	}
}

func (h *staffHandler) AddStaffAccount(w http.ResponseWriter, r *http.Request) error {
	staffIDStr := chi.URLParam(r, "staffID")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid staff id")
	}

	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.StaffID == nil {
		return apperror.NewForbidden(authctx.ErrInsufficientRole.Error())
	}

	var req addStaffAccountRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperror.NewBadRequest("email is required")
	}
	if req.Password == "" {
		return apperror.NewBadRequest("password is required")
	}

	input := staffusecase.AddStaffAccountParams{
		ActorAccountID: actor.AccountID,
		ActorStaffID:   *actor.StaffID,
		StaffID:        staffID,
		Email:          req.Email,
		Password:       req.Password,
	}

	err = h.service.AddStaffAccount(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "staff account successfully created",
	}

	httpx.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *staffHandler) CreateStaff(w http.ResponseWriter, r *http.Request) error {
	var req createStaffRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.Name == "" {
		return apperror.NewBadRequest("name is required")
	}
	if req.Username == "" {
		return apperror.NewBadRequest("username is required")
	}

	input := staffusecase.CreateStaffInput{
		Name:        req.Name,
		Username:    req.Username,
		Description: req.Description,
		LogoUrl:     req.LogoUrl,
		BannerUrl:   req.BannerUrl,
	}

	err := h.service.CreateStaff(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "staff successfully created",
	}

	httpx.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *staffHandler) FindStaff(w http.ResponseWriter, r *http.Request) error {
	page := httpx.QueryIntDefault(r, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := httpx.QueryIntDefault(r, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	idStr := httpx.Query(r, "id")
	sort := httpx.Query(r, "sort")

	input := staffusecase.FindStaffInput{
		Page:  page,
		Limit: limit,
		Sort:  sort,
	}
	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return apperror.NewBadRequest("invalid staff id")
		}
		input.ID = &id
	}

	staff, total, err := h.service.FindStaff(r.Context(), input)
	if err != nil {
		return err
	}

	results := make([]staffResponse, 0, len(staff))
	for _, m := range staff {
		results = append(results, staffResponse{
			ID:        m.ID,
			UserID:    m.UserID,
			Name:      m.Name,
			Username:  m.Username,
			Phone:     m.Phone,
			AvatarURL: m.AvatarURL,
			CreatedAt: m.CreatedAt,
		})
	}

	response := map[string]interface{}{
		"staff": results,
		"page":  page,
		"limit": limit,
		"total": total,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *staffHandler) ListStaffAccounts(w http.ResponseWriter, r *http.Request) error {
	staffIDStr := chi.URLParam(r, "staffID")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid staff id")
	}

	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.StaffID == nil {
		return apperror.NewForbidden(authctx.ErrInsufficientRole.Error())
	}

	input := staffusecase.ListStaffAccountsParams{
		ActorAccountID: actor.AccountID,
		ActorStaffID:   *actor.StaffID,
		StaffID:        staffID,
	}

	accounts, err := h.service.ListStaffAccounts(r.Context(), input)
	if err != nil {
		return err
	}

	results := make([]staffAccountResponse, 0, len(accounts))
	for _, m := range accounts {
		results = append(results, staffAccountResponse{
			AccountID: m.AccountID,
			UserID:    m.UserID,
			Email:     m.Email,
			Name:      m.Name,
			Username:  m.Username,
			Phone:     m.Phone,
			AvatarURL: m.AvatarURL,
			Role: staffAccountRoleResponse{
				ID:   m.Role.ID,
				Code: string(m.Role.Code),
				Name: m.Role.Name,
			},
			LastLoginAt: m.LastLoginAt,
			CreatedAt:   m.CreatedAt,
		})
	}

	response := listStaffAccountsResponse{
		StaffID:  staffID,
		Total:    len(results),
		Accounts: results,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *staffHandler) UpdateStaff(w http.ResponseWriter, r *http.Request) error {
	staffIDStr := chi.URLParam(r, "staffID")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid staff id")
	}

	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.StaffID == nil {
		return apperror.NewForbidden(authctx.ErrInsufficientRole.Error())
	}

	var req updateStaffRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.Name == "" {
		return apperror.NewBadRequest("name is required")
	}

	input := staffusecase.UpdateStaffInput{
		ActorAccountID: actor.AccountID,
		ActorStaffID:   *actor.StaffID,
		StaffID:        staffID,
		Name:           req.Name,
		Description:    req.Description,
		LogoUrl:        req.LogoUrl,
		BannerUrl:      req.BannerUrl,
	}

	err = h.service.UpdateStaff(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "staff successfully updated",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *staffHandler) DeleteStaff(w http.ResponseWriter, r *http.Request) error {
	staffIDStr := chi.URLParam(r, "staffID")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid staff id")
	}

	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.StaffID == nil {
		return apperror.NewForbidden(authctx.ErrInsufficientRole.Error())
	}

	input := staffusecase.DeleteStaffInput{
		ActorAccountID: actor.AccountID,
		ActorStaffID:   *actor.StaffID,
		StaffID:        staffID,
	}

	err = h.service.DeleteStaff(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "staff successfully deleted",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *staffHandler) RemoveStaffAccount(w http.ResponseWriter, r *http.Request) error {
	staffIDStr := chi.URLParam(r, "staffID")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid staff id")
	}

	accountIDStr := chi.URLParam(r, "accountID")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid account id")
	}

	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		return apperror.NewUnauthorized("authentication required")
	}
	if actor.StaffID == nil {
		return apperror.NewForbidden(authctx.ErrInsufficientRole.Error())
	}

	input := staffusecase.RemoveStaffAccountInput{
		ActorAccountID: actor.AccountID,
		ActorStaffID:   *actor.StaffID,
		StaffID:        staffID,
		AccountID:      accountID,
	}

	err = h.service.RemoveStaffAccount(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "staff account successfully removed",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}
