package discounthttp

import (
	"errors"
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/discount"
	"komecore/internal/pagination"
)

type Handler struct {
	service *discount.Service
}

func New(service *discount.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ValidateCoupon(w http.ResponseWriter, r *http.Request) error {
	var req ValidateCouponRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.Code == "" {
		return apperror.NewBadRequest("coupon code is required")
	}

	result, err := h.service.ValidateCoupon(r.Context(), discount.ValidateCouponInput{
		Code:        req.Code,
		Subtotal:    req.Subtotal,
		ShippingFee: req.ShippingFee,
	})
	if err != nil {
		return mapDomainError(err)
	}

	resp := ValidateCouponResponse{
		Code:           result.Coupon.Code,
		Type:           result.Coupon.Type,
		DiscountValue:  result.Coupon.DiscountValue,
		DiscountAmount: result.DiscountAmount,
		Subtotal:       result.Subtotal,
		ShippingFee:    result.ShippingFee,
		FinalTotal:     result.FinalTotal,
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) CreateCoupon(w http.ResponseWriter, r *http.Request) error {
	var req CreateCouponRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if req.Code == "" {
		return apperror.NewBadRequest("coupon code is required")
	}

	coupon, err := h.service.CreateCoupon(r.Context(), discount.CreateCouponInput{
		Code:          req.Code,
		Description:   req.Description,
		Type:          req.Type,
		DiscountValue: req.DiscountValue,
		MinSpend:      req.MinSpend,
		MaxDiscount:   req.MaxDiscount,
		QuotaTotal:    req.QuotaTotal,
		StartsAt:      req.StartsAt,
		ExpiresAt:     req.ExpiresAt,
		IsActive:      req.IsActive,
	})
	if err != nil {
		return mapDomainError(err)
	}

	resp := toCouponResponse(*coupon)
	httpx.WriteJSON(w, http.StatusCreated, resp)
	return nil
}

func (h *Handler) ListCoupons(w http.ResponseWriter, r *http.Request) error {
	page := httpx.QueryIntDefault(r, "page", 1)
	limit := httpx.QueryIntDefault(r, "limit", 10)

	var codePtr *string
	if code := httpx.Query(r, "code"); code != "" {
		codePtr = &code
	}

	var isActivePtr *bool
	if active := httpx.Query(r, "is_active"); active != "" {
		val := active == "true"
		isActivePtr = &val
	}

	coupons, total, err := h.service.ListCoupons(r.Context(), discount.ListCouponsInput{
		Code:     codePtr,
		IsActive: isActivePtr,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return err
	}

	items := make([]CouponResponse, 0, len(coupons))
	for _, c := range coupons {
		items = append(items, toCouponResponse(c))
	}

	meta := pagination.NewPaginationMeta(page, limit, int64(total))
	httpx.WriteJSON(w, http.StatusOK, httpx.NewPaginatedResponse(items, meta))
	return nil
}

func toCouponResponse(c discount.Coupon) CouponResponse {
	return CouponResponse{
		ID:             c.ID,
		Code:           c.Code,
		Description:    c.Description,
		Type:           c.Type,
		DiscountValue:  c.DiscountValue,
		MinSpend:       c.MinSpend,
		MaxDiscount:    c.MaxDiscount,
		QuotaTotal:     c.QuotaTotal,
		QuotaRemaining: c.QuotaRemaining,
		StartsAt:       c.StartsAt.UTC(),
		ExpiresAt:      c.ExpiresAt.UTC(),
		IsActive:       c.IsActive,
		CreatedAt:      c.CreatedAt.UTC(),
		UpdatedAt:      httpx.ToUTCPtr(c.UpdatedAt),
	}
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, discount.ErrCouponNotFound):
		return apperror.NewNotFoundWithCode(apperror.CodeCouponNotFound, err.Error())
	case errors.Is(err, discount.ErrCouponInactive):
		return apperror.NewBadRequest(err.Error())
	case errors.Is(err, discount.ErrCouponExpired):
		return apperror.NewBadRequestWithCode(apperror.CodeCouponExpired, err.Error())
	case errors.Is(err, discount.ErrCouponNotStarted):
		return apperror.NewBadRequest(err.Error())
	case errors.Is(err, discount.ErrCouponQuotaExceeded):
		return apperror.NewConflictWithCode(apperror.CodeCouponQuotaExhausted, err.Error())
	case errors.Is(err, discount.ErrCouponMinSpendNotMet):
		return apperror.NewBadRequestWithCode(apperror.CodeCouponMinSpendNotMet, err.Error())
	case errors.Is(err, discount.ErrCouponAlreadyRedeemed):
		return apperror.NewConflict(err.Error())
	case errors.Is(err, discount.ErrDuplicateCouponCode):
		return apperror.NewConflict(err.Error())
	case errors.Is(err, discount.ErrInvalidCouponType),
		errors.Is(err, discount.ErrInvalidDiscountValue),
		errors.Is(err, discount.ErrInvalidDates),
		errors.Is(err, discount.ErrInvalidQuota):
		return apperror.NewBadRequest(err.Error())
	default:
		return err
	}
}
