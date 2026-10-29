package reviewhttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/httpx"
	"komecore/internal/modules/review"
	"komecore/internal/pagination"
)

type Handler struct {
	service *review.Service
}

func New(service *review.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	productID, err := httpx.ParamUUID(r, "productId")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	var req createReviewRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}

	rev, err := h.service.CreateReview(r.Context(), review.CreateReviewInput{
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
		ID:           rev.ID,
		ProductID:    rev.ProductID,
		CustomerID:   rev.CustomerID,
		OrderID:      rev.OrderID,
		Rating:       rev.Rating,
		Title:        rev.Title,
		Comment:      rev.Comment,
		HelpfulCount: 0,
		Media:        make([]string, 0),
		CreatedAt:    rev.CreatedAt.UTC(),
		UpdatedAt:    httpx.ToUTCPtr(rev.UpdatedAt),
	}
	httpx.WriteJSON(w, http.StatusCreated, resp)
	return nil
}

func (h *Handler) ListProductReviews(w http.ResponseWriter, r *http.Request) error {
	productID, err := httpx.ParamUUID(r, "productId")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	page := httpx.QueryIntDefault(r, "page", 1)
	limit := httpx.QueryIntDefault(r, "limit", 10)

	result, err := h.service.ListReviews(r.Context(), review.ListReviewsInput{
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
			HelpfulCount: 0,
			Media:        make([]string, 0),
			CreatedAt:    rev.Review.CreatedAt.UTC(),
			UpdatedAt:    httpx.ToUTCPtr(rev.Review.UpdatedAt),
		})
	}

	resp := listReviewsResponse{
		Items:      items,
		Pagination: pagination.NewPaginationMeta(result.Page, result.Limit, int64(result.Total)),
		RatingSummary: RatingSummaryResponse{
			AverageRating: result.AverageRating,
			ReviewsCount:  result.ReviewCount,
		},
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) error {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor == nil {
		authCtx, err := httpx.RequireAuth(r)
		if err != nil {
			return err
		}
		actor = authctx.ActorFromAuthContext(authCtx)
	}

	reviewID, err := httpx.ParamUUID(r, "id")
	if err != nil {
		return apperror.NewBadRequest("invalid review id")
	}

	err = h.service.DeleteReview(r.Context(), review.DeleteReviewInput{
		ReviewID: reviewID,
		Actor:    actor,
	})
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, messageResponse{
		Message: "review deleted successfully",
	})
	return nil
}
