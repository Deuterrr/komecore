package discounthttp

import (
	"errors"
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/discount/discountdomain"
	"komecore/internal/modules/discount/discountusecase"
)

type DiscountHandler struct {
	service *discountusecase.DiscountService
}

func NewDiscountHandler(service *discountusecase.DiscountService) *DiscountHandler {
	return &DiscountHandler{
		service: service,
	}
}

func (h *DiscountHandler) ValidateCoupon(w http.ResponseWriter, r *http.Request) error {
	var req ValidateCouponRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	if req.Code == "" {
		return apperrors.NewBadRequest("coupon code is required")
	}

	result, err := h.service.ValidateCoupon(r.Context(), discountusecase.ValidateCouponInput{
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

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *DiscountHandler) CreateCoupon(w http.ResponseWriter, r *http.Request) error {
	var req CreateCouponRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	if req.Code == "" {
		return apperrors.NewBadRequest("coupon code is required")
	}

	coupon, err := h.service.CreateCoupon(r.Context(), discountusecase.CreateCouponInput{
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
	apphttp.WriteJSON(w, http.StatusCreated, resp)
	return nil
}

func (h *DiscountHandler) ListCoupons(w http.ResponseWriter, r *http.Request) error {
	page := apphttp.QueryIntDefault(r, "page", 1)
	limit := apphttp.QueryIntDefault(r, "limit", 10)

	var codePtr *string
	if code := apphttp.Query(r, "code"); code != "" {
		codePtr = &code
	}

	var isActivePtr *bool
	if active := apphttp.Query(r, "is_active"); active != "" {
		val := active == "true"
		isActivePtr = &val
	}

	coupons, total, err := h.service.ListCoupons(r.Context(), discountusecase.ListCouponsInput{
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

	apphttp.WriteJSON(w, http.StatusOK, ListCouponsResponse{
		Items: items,
		Total: total,
	})
	return nil
}

func toCouponResponse(c discountdomain.Coupon) CouponResponse {
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
		StartsAt:       c.StartsAt,
		ExpiresAt:      c.ExpiresAt,
		IsActive:       c.IsActive,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, discountdomain.ErrCouponNotFound):
		return apperrors.NewNotFound(err.Error())
	case errors.Is(err, discountdomain.ErrCouponInactive):
		return apperrors.NewBadRequest(err.Error())
	case errors.Is(err, discountdomain.ErrCouponExpired):
		return apperrors.NewBadRequest(err.Error())
	case errors.Is(err, discountdomain.ErrCouponNotStarted):
		return apperrors.NewBadRequest(err.Error())
	case errors.Is(err, discountdomain.ErrCouponQuotaExceeded):
		return apperrors.NewBadRequest(err.Error())
	case errors.Is(err, discountdomain.ErrCouponMinSpendNotMet):
		return apperrors.NewBadRequest(err.Error())
	case errors.Is(err, discountdomain.ErrCouponAlreadyRedeemed):
		return apperrors.NewConflict(err.Error())
	case errors.Is(err, discountdomain.ErrDuplicateCouponCode):
		return apperrors.NewConflict(err.Error())
	case errors.Is(err, discountdomain.ErrInvalidCouponType),
		errors.Is(err, discountdomain.ErrInvalidDiscountValue),
		errors.Is(err, discountdomain.ErrInvalidDates),
		errors.Is(err, discountdomain.ErrInvalidQuota):
		return apperrors.NewBadRequest(err.Error())
	default:
		return err
	}
}
