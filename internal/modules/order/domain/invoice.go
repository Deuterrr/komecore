package domain

import (
	"fmt"
	"strings"
	"time"

	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type InvoiceStatus string

const (
	InvoiceStatusIssued InvoiceStatus = "issued"
	InvoiceStatusVoid   InvoiceStatus = "void"
)

type Invoice struct {
	ID uuid.UUID

	Number  string
	OrderID uuid.UUID

	Status InvoiceStatus

	Subtotal    int64
	ShippingFee int64
	Total       int64

	IssuedAt  time.Time
	CreatedAt time.Time
}

func (i Invoice) NewInvoiceItemFromOrderItem(orderItem OrderItem) InvoiceItem {
	return InvoiceItem{
		ID:             uuid.New(),
		InvoiceID:      i.ID,
		ShopID:         orderItem.ShopID,
		ShopName:       orderItem.ShopName,
		ProductID:      orderItem.ProductID,
		ProductName:    orderItem.ProductName,
		Quantity:       orderItem.Quantity,
		UnitPrice:      orderItem.UnitPrice,
		Subtotal:       orderItem.Subtotal,
		CourierCode:    orderItem.CourierCode,
		CourierService: orderItem.CourierService,
		ShippingFee:    orderItem.ShippingFee,
	}
}

func NewInvoiceNumber() string {
	return fmt.Sprintf(
		"INV-%s-%s",
		appclock.Now().Format("20060102"),
		strings.ToUpper(uuid.NewString()[:6]),
	)
}
