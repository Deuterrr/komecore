package http

import (
	"net/http"

	"komecore/internal/common/authctx"
	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/modules/review/usecase"
)

type ReviewHandler struct {
	service *usecase.ReviewService
}

func NewReviewHandler(
	service *usecase.ReviewService,
) *ReviewHandler {
	return &ReviewHandler{
		service: service,
	}
}

func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := apphttp.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := apphttp.ParamUUID(r, "productId")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	var req createReviewRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	review, err := h.service.CreateReview(r.Context(), usecase.CreateReviewInput{
		CustomerID: customerID,
		ProductID:  productID,
		OrderID:    req.OrderID,
		Rating:     req.Rating,
		Title:      req.Title,
		Comment:    req.Comment,
	})
	if err != nil {
		return err
	}

	resp := reviewResponse{
		ID:         review.ID,
		ProductID:  review.ProductID,
		CustomerID: review.CustomerID,
		OrderID:    review.OrderID,
		Rating:     review.Rating,
		Title:      review.Title,
		Comment:    review.Comment,
		CreatedAt:  review.CreatedAt,
		UpdatedAt:  review.UpdatedAt,
	}

	apphttp.WriteJSON(w, http.StatusCreated, resp)
	return nil
}

func (h *ReviewHandler) ListProductReviews(w http.ResponseWriter, r *http.Request) error {
	productID, err := apphttp.ParamUUID(r, "productId")
	if err != nil {
		return apperrors.NewBadRequest("invalid product id")
	}

	page := apphttp.QueryIntDefault(r, "page", 1)
	limit := apphttp.QueryIntDefault(r, "limit", 10)

	result, err := h.service.ListReviews(r.Context(), usecase.ListReviewsInput{
		ProductID: productID,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return err
	}

	items := make([]reviewListItemResponse, 0, len(result.Reviews))
	for _, rev := range result.Reviews {
		items = append(items, reviewListItemResponse{
			ID:           rev.Review.ID,
			ProductID:    rev.Review.ProductID,
			CustomerID:   rev.Review.CustomerID,
			CustomerName: rev.CustomerName,
			AvatarURL:    rev.AvatarURL,
			OrderID:      rev.Review.OrderID,
			Rating:       rev.Review.Rating,
			Title:        rev.Review.Title,
			Comment:      rev.Review.Comment,
			CreatedAt:    rev.Review.CreatedAt,
			UpdatedAt:    rev.Review.UpdatedAt,
		})
	}

	resp := listReviewsResponse{
		Reviews:       items,
		AverageRating: result.AverageRating,
		ReviewCount:   result.ReviewCount,
		Page:          result.Page,
		Limit:         result.Limit,
		Total:         result.Total,
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor == nil {
		authCtx, err := apphttp.RequireAuth(r)
		if err != nil {
			return err
		}
		actor = authctx.ActorFromAuthContext(authCtx)
	}

	reviewID, err := apphttp.ParamUUID(r, "id")
	if err != nil {
		return apperrors.NewBadRequest("invalid review id")
	}

	err = h.service.DeleteReview(r.Context(), usecase.DeleteReviewInput{
		ReviewID: reviewID,
		Actor:    actor,
	})
	if err != nil {
		return err
	}

	apphttp.WriteJSON(w, http.StatusOK, messageResponse{
		Message: "review deleted successfully",
	})
	return nil
}
