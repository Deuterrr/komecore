package persistence

import (
	"komecore/internal/modules/product/domain"
	"komecore/pkg/money"
)

func (m *productModel) ToDomain() (*domain.Product, error) {
	price, err := money.ParsePriceToInt64(m.BasePrice)
	if err != nil {
		return nil, err
	}

	var weight *float64
	if m.Weight != nil {
		w, err := money.ParseStringToFloat(m.Weight)
		if err != nil {
			return nil, err
		}
		weight = w
	}

	return &domain.Product{
		ID:          m.ID,
		SKU:         m.SKU,
		Name:        m.Name,
		Description: m.Description,
		Status:      domain.ProductStatus(m.Status),

		Price:  price,
		Weight: weight,

		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		ArchivedAt: m.ArchivedAt,
	}, nil
}

func FromDomain(p *domain.Product) *productModel {
	basePrice := money.FormatInt64ToPrice(p.Price)

	var weight *string
	if p.Weight != nil {
		w := money.FormatFloatToString(*p.Weight)
		weight = &w
	}

	return &productModel{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),

		BasePrice: basePrice,
		Weight:    weight,

		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
		ArchivedAt: p.ArchivedAt,
	}
}
