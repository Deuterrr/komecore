package reviewhttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/httpx"
	"komecore/internal/modules/review/reviewusecase"
	"komecore/internal/pagination"
)

type ReviewHandler struct {
	service *reviewusecase.ReviewService
}

func NewReviewHandler(
	service *reviewusecase.ReviewService,
) *ReviewHandler {
	return &ReviewHandler{
		service: service,
	}
}

func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) error {
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

	review, err := h.service.CreateReview(r.Context(), reviewusecase.CreateReviewInput{
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
		ID:           review.ID,
		ProductID:    review.ProductID,
		CustomerID:   review.CustomerID,
		OrderID:      review.OrderID,
		Rating:       review.Rating,
		Title:        review.Title,
		Comment:      review.Comment,
		HelpfulCount: 0,
		Media:        make([]string, 0),
		CreatedAt:    review.CreatedAt.UTC(),
		UpdatedAt:    httpx.ToUTCPtr(review.UpdatedAt),
	}
	httpx.WriteJSON(w, http.StatusCreated, resp)
	return nil
}

func (h *ReviewHandler) ListProductReviews(w http.ResponseWriter, r *http.Request) error {
	productID, err := httpx.ParamUUID(r, "productId")
	if err != nil {
		return apperror.NewBadRequest("invalid product id")
	}

	page := httpx.QueryIntDefault(r, "page", 1)
	limit := httpx.QueryIntDefault(r, "limit", 10)

	result, err := h.service.ListReviews(r.Context(), reviewusecase.ListReviewsInput{
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

func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) error {
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

	err = h.service.DeleteReview(r.Context(), reviewusecase.DeleteReviewInput{
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
