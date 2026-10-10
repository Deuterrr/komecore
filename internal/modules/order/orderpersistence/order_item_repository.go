package orderpersistence

import (
	"context"
	"encoding/json"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderdomain"

	"github.com/google/uuid"
)

type OrderItemRepository struct{}

func NewOrderItemRepository() *OrderItemRepository {
	return &OrderItemRepository{}
}

func (r *OrderItemRepository) ListByOrderID(
	ctx context.Context,
	exec transaction.Executor,
	orderID uuid.UUID,
) ([]orderdomain.OrderItem, error) {
	query := `
		SELECT
			id,
			order_id,
			shipment_id,
			shop_id,
			shop_name,
			product_id,
			product_name,
			quantity,
			unit_price,
			subtotal,
			courier_code,
			courier_service,
			shipping_fee_total,
			item_options
		FROM
			order_items
		WHERE
			order_id = $1
	`

	rows, err := exec.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query order items by order id failed: %w", err)
	}
	defer rows.Close()

	items, err := transaction.CollectRows(rows, func(row transaction.CollectableRow) (orderdomain.OrderItem, error) {
		var item orderdomain.OrderItem
		var rawOptions []byte
		err := row.Scan(
			&item.ID,
			&item.OrderID,
			&item.ShipmentID,
			&item.ShopID,
			&item.ShopName,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CourierCode,
			&item.CourierService,
			&item.ShippingFee,
			&rawOptions,
		)
		if len(rawOptions) > 0 {
			_ = json.Unmarshal(rawOptions, &item.ItemOptions)
		}
		item.ItemOptions = item.ItemOptions.Normalized()
		return item, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan order items failed: %w", err)
	}

	return items, nil
}

func (r *OrderItemRepository) SaveBulk(
	ctx context.Context,
	exec transaction.Executor,
	items []orderdomain.OrderItem,
) error {
	query := `
		INSERT INTO order_items (
			id,
			order_id,
			shipment_id,
			shop_id,
			shop_name,
			product_id,
			product_name,
			quantity,
			unit_price,
			subtotal,
			courier_code,
			courier_service,
			shipping_fee_total,
			item_options
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb)
		ON CONFLICT (id)
		DO UPDATE SET
			order_id = EXCLUDED.order_id,
			shipment_id = EXCLUDED.shipment_id,
			shop_id = EXCLUDED.shop_id,
			shop_name = EXCLUDED.shop_name,
			product_id = EXCLUDED.product_id,
			product_name = EXCLUDED.product_name,
			quantity = EXCLUDED.quantity,
			unit_price = EXCLUDED.unit_price,
			subtotal = EXCLUDED.subtotal,
			courier_code = EXCLUDED.courier_code,
			courier_service = EXCLUDED.courier_service,
			shipping_fee_total = EXCLUDED.shipping_fee_total,
			item_options = EXCLUDED.item_options
	`

	for _, item := range items {
		optBytes, _ := json.Marshal(item.ItemOptions.Normalized())

		_, err := exec.Exec(ctx, query,
			item.ID,
			item.OrderID,
			item.ShipmentID,
			item.ShopID,
			item.ShopName,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
			item.CourierCode,
			item.CourierService,
			item.ShippingFee,
			string(optBytes),
		)
		if err != nil {
			return fmt.Errorf("query to save order item: %w", err)
		}
	}

	return nil
}

func (r *OrderItemRepository) ListByOrderIDs(
	ctx context.Context,
	exec transaction.Executor,
	orderIDs []uuid.UUID,
) ([]orderdomain.OrderItem, error) {
	if len(orderIDs) == 0 {
		return []orderdomain.OrderItem{}, nil
	}

	query := `
		SELECT
			id,
			order_id,
			shipment_id,
			shop_id,
			shop_name,
			product_id,
			product_name,
			quantity,
			unit_price,
			subtotal,
			courier_code,
			courier_service,
			shipping_fee_total,
			item_options
		FROM
			order_items
		WHERE
			order_id = ANY($1::uuid[])
	`

	orderIDStrings := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		orderIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, orderIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query order items by order ids failed: %w", err)
	}
	defer rows.Close()

	items, err := transaction.CollectRows(rows, func(row transaction.CollectableRow) (orderdomain.OrderItem, error) {
		var item orderdomain.OrderItem
		var rawOptions []byte
		err := row.Scan(
			&item.ID,
			&item.OrderID,
			&item.ShipmentID,
			&item.ShopID,
			&item.ShopName,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CourierCode,
			&item.CourierService,
			&item.ShippingFee,
			&rawOptions,
		)
		if len(rawOptions) > 0 {
			_ = json.Unmarshal(rawOptions, &item.ItemOptions)
		}
		item.ItemOptions = item.ItemOptions.Normalized()
		return item, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan order items failed: %w", err)
	}

	return items, nil
}

func (r *OrderItemRepository) ListByShipmentID(
	ctx context.Context,
	exec transaction.Executor,
	shipmentID uuid.UUID,
) ([]orderdomain.OrderItem, error) {
	query := `
		SELECT
			id,
			order_id,
			shipment_id,
			shop_id,
			shop_name,
			product_id,
			product_name,
			quantity,
			unit_price,
			subtotal,
			courier_code,
			courier_service,
			shipping_fee_total,
			item_options
		FROM
			order_items
		WHERE
			shipment_id = $1
	`

	rows, err := exec.Query(ctx, query, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("query order items by shipment id failed: %w", err)
	}
	defer rows.Close()

	items, err := transaction.CollectRows(rows, func(row transaction.CollectableRow) (orderdomain.OrderItem, error) {
		var item orderdomain.OrderItem
		var rawOptions []byte
		err := row.Scan(
			&item.ID,
			&item.OrderID,
			&item.ShipmentID,
			&item.ShopID,
			&item.ShopName,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CourierCode,
			&item.CourierService,
			&item.ShippingFee,
			&rawOptions,
		)
		if len(rawOptions) > 0 {
			_ = json.Unmarshal(rawOptions, &item.ItemOptions)
		}
		item.ItemOptions = item.ItemOptions.Normalized()
		return item, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan order items failed: %w", err)
	}

	return items, nil
}

func (r *OrderItemRepository) AssignShipment(
	ctx context.Context,
	exec transaction.Executor,
	shipmentID uuid.UUID,
	itemIDs []uuid.UUID,
) error {
	if len(itemIDs) == 0 {
		return nil
	}

	query := `
		UPDATE order_items
		SET shipment_id = $1
		WHERE id = ANY($2::uuid[])
	`

	itemIDStrings := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		itemIDStrings[i] = id.String()
	}

	_, err := exec.Exec(ctx, query, shipmentID, itemIDStrings)
	if err != nil {
		return fmt.Errorf("assign shipment to order items failed: %w", err)
	}

	return nil
}
