package midtrans

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"komecore/internal/infra/paymentgateway"
)

// ErrInvalidSignature is returned when an inbound webhook payload fails HMAC SHA-512 verification.
var ErrInvalidSignature = paymentgateway.ErrInvalidSignature

// VerifySignature validates that an incoming webhook notification was signed by Midtrans
// using SHA-512(order_id + status_code + gross_amount + ServerKey).
func VerifySignature(orderID, statusCode, grossAmount, serverKey, signatureKey string) bool {
	if orderID == "" ||
		statusCode == "" ||
		grossAmount == "" ||
		serverKey == "" ||
		signatureKey == "" {
		return false
	}
	raw := orderID + statusCode + grossAmount + serverKey
	hash := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(hash[:])
	return strings.EqualFold(expected, signatureKey)
}

// ParseNotification verifies and parses an inbound webhook notification payload.
//
// Security & Performance Contract:
//   - MUST verify the cryptographic signature (SHA-512) before processing.
//   - MUST NOT execute redundant outbound HTTP status requests if the signature is valid.
//   - Returns ErrInvalidSignature if the payload has been tampered with or signature is invalid.
func (p *midtransAPIProvider) ParseNotification(
	ctx context.Context,
	payload paymentgateway.NotificationPayload,
) (*paymentgateway.NotificationResult, error) {
	orderID, _ := payload["order_id"].(string)
	if orderID == "" {
		return nil, fmt.Errorf("midtrans: parse notification: missing order_id in payload")
	}

	statusCode, _ := payload["status_code"].(string)
	if statusCode == "" {
		return nil, fmt.Errorf("midtrans: parse notification: missing status_code in payload")
	}

	var grossAmountStr string
	switch v := payload["gross_amount"].(type) {
	case string:
		grossAmountStr = v
	case float64:
		grossAmountStr = strconv.FormatFloat(v, 'f', 2, 64)
	case int:
		grossAmountStr = fmt.Sprintf("%d.00", v)
	case int64:
		grossAmountStr = fmt.Sprintf("%d.00", v)
	default:
		grossAmountStr = fmt.Sprintf("%v", v)
	}

	signatureKey, _ := payload["signature_key"].(string)
	if signatureKey == "" {
		return nil, ErrInvalidSignature
	}

	if !VerifySignature(orderID, statusCode, grossAmountStr, p.cfg.ServerKey, signatureKey) {
		return nil, ErrInvalidSignature
	}

	txStatus, _ := payload["transaction_status"].(string)
	fraudStatus, _ := payload["fraud_status"].(string)
	txID, _ := payload["transaction_id"].(string)

	grossAmount, err := parseAmount(grossAmountStr)
	if err != nil {
		return nil, fmt.Errorf("midtrans: parse notification gross_amount: %w", err)
	}

	return &paymentgateway.NotificationResult{
		GatewayTransactionID: txID,
		GatewayOrderID:       orderID,
		Status:               mapNotificationStatus(txStatus, fraudStatus),
		GrossAmount:          grossAmount,
		FraudStatus:          fraudStatus,
		RawStatus:            txStatus,
	}, nil
}
