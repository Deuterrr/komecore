package producthttp

import (
	"errors"
	"io"
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	appmultipart "komecore/internal/common/http/multipart"
	"komecore/internal/modules/product/productusecase"

	"github.com/google/uuid"
)

type ProductHandler struct {
	findProducts    *productusecase.FindProductsUsecase
	getProduct      *productusecase.GetProductUsecase
	saveProduct     *productusecase.SaveProductUsecase
	deleteProduct   *productusecase.DeleteProductUsecase
	addProductImage *productusecase.AddProductImagesUsecase
	getProductStats *productusecase.GetProductStatsUsecase
}

func NewProductHandler(
	findProducts *productusecase.FindProductsUsecase,
	getProduct *productusecase.GetProductUsecase,
	saveProduct *productusecase.SaveProductUsecase,
	deleteProduct *productusecase.DeleteProductUsecase,
	addProductImage *productusecase.AddProductImagesUsecase,
	getProductStats *productusecase.GetProductStatsUsecase,
) *ProductHandler {
	return &ProductHandler{
		findProducts:    findProducts,
		getProduct:      getProduct,
		saveProduct:     saveProduct,
		deleteProduct:   deleteProduct,
		addProductImage: addProductImage,
		getProductStats: getProductStats,
	}
}

func (h *ProductHandler) FindProducts(w http.ResponseWriter, r *http.Request) error {
	page := apphttp.QueryIntDefault(r, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := apphttp.QueryIntDefault(r, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	name := apphttp.Query(r, "name")
	q := apphttp.Query(r, "q")
	id := apphttp.Query(r, "id")
	sort := apphttp.Query(r, "sort")
	shopID := apphttp.Query(r, "shop_id")
	shopSlug := apphttp.Query(r, "shop_slug")
	status := apphttp.Query(r, "status")
	includeArchived := apphttp.Query(r, "include_archived") == "true"

	if shopID == "" && shopSlug == "" {
		shopID = r.Header.Get("X-Shop-ID")
	}

	input := productusecase.FindProductsInput{
		Page:  page,
		Limit: limit,
		Sort:  sort,
	}
	if q != "" {
		input.SearchQuery = &q
	}
	if name != "" {
		input.Name = &name
	}
	if id != "" {
		input.ID = &id
	}
	if shopID != "" {
		input.ShopID = &shopID
	}
	if shopSlug != "" {
		input.ShopSlug = &shopSlug
	}
	if status != "" && status != "all" && status != "any" {
		input.Status = &status
	} else if !includeArchived && status != "all" && status != "any" {
		input.ExcludeArchived = true
	}

	products, total, err := h.findProducts.Execute(r.Context(), input)

	if err != nil {
		return err
	}

	results := make([]productCatalogResponse, 0, len(products))
	for _, p := range products {
		availabilities := make([]productAvailabilityResponse, 0, len(p.Availability))
		for _, avail := range p.Availability {
			availabilities = append(availabilities, productAvailabilityResponse{
				ShopID: avail.ShopID.String(),
				Slug:   avail.ShopSlug,
				Name:   avail.ShopName,
				Stock:  avail.Stock,
			})
		}

		var result productCatalogResponse

		result = productCatalogResponse{
			productBaseResponse: productBaseResponse{
				ID:     p.Product.ID,
				SKU:    p.Product.SKU,
				Name:   p.Product.Name,
				Slug:   p.Product.Slug,
				Status: string(p.Product.Status),
				IsAvailable: productStatusDTO(p.Product.Status) == ProductStatusActive &&
					p.Inventory.TotalStock > 0,
				Price:         p.Product.Price,
				TotalStock:    p.Inventory.TotalStock,
				AverageRating: p.Product.AverageRating,
				ReviewCount:   p.Product.ReviewCount,
				Banner: productImageResponse{
					Thumbnail: &p.Images.Thumbnail,
				},
			},
			Availability: availabilities,
		}

		if *result.Banner.Thumbnail == "" {
			result.Banner = productImageResponse{
				Thumbnail: nil,
			}
		}

		results = append(results, result)
	}

	response := map[string]interface{}{
		"products": results,
		"page":     page,
		"limit":    limit,
		"total":    total,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) error {
	productSlug := apphttp.Param(r, "slug")

	productDetail, err := h.getProduct.Execute(r.Context(), productSlug)
	if err != nil {
		return err
	}
	if productDetail == nil {
		return apperrors.NewNotFound("product not found")
	}

	var available int
	inventories := make([]productInventoryView, 0, len(productDetail.ShopInventories))
	for _, inventory := range productDetail.ShopInventories {
		available += inventory.Available()

		inventories = append(inventories, productInventoryView{
			ID:         inventory.ID,
			ShopID:     inventory.ShopID,
			TotalStock: inventory.TotalStock,
			Available:  inventory.Available(),
		})
	}

	images := make([]productImageResponse, 0, len(productDetail.Images))
	for _, img := range productDetail.Images {
		image := productImageResponse{}

		if img.Thumbnail != "" {
			image.Thumbnail = &img.Thumbnail
		}

		if img.Preview != "" {
			image.Preview = &img.Preview
		}

		if img.Detail != "" {
			image.Detail = &img.Detail
		}

		images = append(images, image)
	}

	availabilities := make(
		[]productAvailabilityResponse,
		0,
		len(productDetail.Availability),
	)
	for _, avail := range productDetail.Availability {
		availabilities = append(availabilities, productAvailabilityResponse{
			ShopID: avail.ShopID.String(),
			Slug:   avail.ShopSlug,
			Name:   avail.ShopName,
			Stock:  avail.Stock,
		})
	}

	var banner productImageResponse
	if len(images) > 0 {
		banner = images[0]
	}

	response := productDetailResponse{
		productBaseResponse: productBaseResponse{
			ID:            productDetail.Product.ID,
			SKU:           productDetail.Product.SKU,
			Name:          productDetail.Product.Name,
			Slug:          productDetail.Product.Slug,
			Status:        string(productDetail.Product.Status),
			IsAvailable:   productStatusDTO(productDetail.Product.Status) == ProductStatusActive && available > 0,
			Price:         productDetail.Product.Price,
			TotalStock:    available,
			AverageRating: productDetail.Product.AverageRating,
			ReviewCount:   productDetail.Product.ReviewCount,
			Banner:        banner,
		},
		Description:  productDetail.Product.Description,
		Weight:       productDetail.Product.Weight,
		UpdatedAt:    productDetail.Product.UpdatedAt,
		Gallery:      images,
		Availability: availabilities,
	}
	response.IsAvailable =
		productStatusDTO(productDetail.Product.Status) == ProductStatusActive &&
			available > 0

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ProductHandler) SaveProduct(w http.ResponseWriter, r *http.Request) error {
	var req saveProductRequest

	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return apperrors.NewBadRequest("invalid name")
	}
	if req.SKU == "" {
		return apperrors.NewBadRequest("invalid sku")
	}
	if req.Price < 0 {
		return apperrors.NewBadRequest("invalid price")
	}
	if !req.Status.isStatusValid() {
		return apperrors.NewBadRequest("invalid status")
	}

	var productID *uuid.UUID
	if req.ID != nil && *req.ID != "" {
		parsed, err := uuid.Parse(*req.ID)
		if err != nil {
			return apperrors.NewBadRequest("invalid product id")
		}
		productID = &parsed
	}

	input := productusecase.SaveProductInput{
		ID:                   productID,
		SKU:                  req.SKU,
		Name:                 req.Name,
		Description:          req.Description,
		Status:               string(req.Status),
		Price:                req.Price,
		Weight:               req.Weight,
		CostPrice:            req.CostPrice,
		SupplierLeadTimeDays: req.SupplierLeadTimeDays,
	}

	err := h.saveProduct.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "product successfully saved",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ProductHandler) AddProductImages(w http.ResponseWriter, r *http.Request) error {
	files, err := appmultipart.ParseMultiple(r, "image", 16<<20)
	if err != nil {
		return err
	}

	images := make([]productusecase.ProductImageInput, 0, len(files))
	for i, file := range files {
		data, err := io.ReadAll(file.File)
		if err != nil {
			return apperrors.NewInternal(errors.New("failed to read uploaded file"))
		}

		err = file.File.Close()
		if err != nil {
			return apperrors.NewInternal(errors.New("failed to close uploaded file"))
		}

		images = append(images, productusecase.ProductImageInput{
			Data:         data,
			OriginalName: file.Filename,
			MIMEType:     file.ContentType,
			SizeBytes:    file.Size,
			IsPrimary:    i == 0,
			DisplayOrder: i,
		})
	}

	productID, err := apphttp.ParamUUID(r, "id")
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	input := productusecase.AddProductImageInput{
		ProductID: productID,
		Images:    images,
	}

	err = h.addProductImage.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "product image successfully added",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) error {
	productID, err := apphttp.ParamUUID(r, "id")
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	err = h.deleteProduct.Execute(r.Context(), productID)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "product successfully deleted",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (s productStatusDTO) isStatusValid() bool {
	switch s {
	case ProductStatusActive,
		ProductStatusInactive,
		ProductStatusArchived:
		return true
	default:
		return false
	}
}

func (h *ProductHandler) GetProductStats(w http.ResponseWriter, r *http.Request) error {
	page := apphttp.QueryIntDefault(r, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := apphttp.QueryIntDefault(r, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	name := apphttp.Query(r, "name")
	id := apphttp.Query(r, "id")
	sort := apphttp.Query(r, "sort")

	input := productusecase.GetProductStatsInput{
		Page:  page,
		Limit: limit,
		Sort:  sort,
	}
	if name != "" {
		input.Name = &name
	}
	if id != "" {
		input.ID = &id
	}

	stats, total, err := h.getProductStats.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	results := make([]productStatsResponse, 0, len(stats))
	for _, s := range stats {
		var thumb *string
		if s.Thumbnail != "" {
			t := s.Thumbnail
			thumb = &t
		}

		results = append(results, productStatsResponse{
			ID:                   s.Product.ID,
			SKU:                  s.Product.SKU,
			Name:                 s.Product.Name,
			Slug:                 s.Product.Slug,
			Status:               string(s.Product.Status),
			Price:                s.Product.Price,
			CostPrice:            s.Performance.CostPrice,
			SupplierLeadTimeDays: s.Performance.SupplierLeadTimeDays,
			GrossMarginPct:       s.Performance.GrossMarginPct,
			ViewCount:            s.Performance.ViewCount,
			TotalStock:           s.TotalStock,
			SalesVelocity7d:      s.SalesVelocity7d,
			SalesVelocity30d:     s.SalesVelocity30d,
			SalesVelocity90d:     s.SalesVelocity90d,
			ConversionRate:       s.ConversionRate,
			RevenueContribPct:    s.RevenueContribPct,
			ReturnRate:           s.ReturnRate,
			AverageRating:        s.AverageRating,
			ReviewCount:          s.ReviewCount,
			Thumbnail:            thumb,
		})
	}

	response := map[string]interface{}{
		"stats": results,
		"page":  page,
		"limit": limit,
		"total": total,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}
