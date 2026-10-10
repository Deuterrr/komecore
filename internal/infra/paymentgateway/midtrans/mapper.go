package midtrans

import (
	"fmt"
	"strings"

	"komecore/internal/apperror"
	"komecore/internal/infra/paymentgateway"
)

// mapNotificationStatus converts Midtrans transaction_status + fraud_status
// into our normalised NotificationStatus.
func mapNotificationStatus(txStatus, fraudStatus string) paymentgateway.NotificationStatus {
	switch txStatus {
	case "capture":
		if fraudStatus == "challenge" {
			return paymentgateway.NotificationStatusChallenge
		}
		return paymentgateway.NotificationStatusSettlement

	case "settlement":
		return paymentgateway.NotificationStatusSettlement

	case "pending":
		return paymentgateway.NotificationStatusPending

	case "deny":
		return paymentgateway.NotificationStatusDeny

	case "expire":
		return paymentgateway.NotificationStatusExpire

	case "cancel":
		return paymentgateway.NotificationStatusCancel

	case "refund", "partial_refund":
		return paymentgateway.NotificationStatusRefund

	default:
		return paymentgateway.NotificationStatus(txStatus)
	}
}

// parseAmount converts the Midtrans gross_amount string (e.g. "150000.00")
// into an int64 representing the amount in the smallest unit.
func parseAmount(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	// Strip decimal part if present (IDR has no sub-unit).
	if idx := strings.Index(raw, "."); idx != -1 {
		raw = raw[:idx]
	}
	var amount int64
	_, err := fmt.Sscanf(raw, "%d", &amount)
	if err != nil {
		return 0, fmt.Errorf("parse %q as int64: %w", raw, err)
	}
	return amount, nil
}

// mapProviderStatusCode converts Midtrans status_code strings into
// valid completions or typed AppErrors (NotFound, BadRequest, Conflict).
func mapProviderStatusCode(statusCode, statusMessage, contextPrefix string) error {
	switch statusCode {
	case "200", "201", "202", "407":
		return nil
	case "404":
		return apperror.NewNotFound(fmt.Sprintf("%s: transaction not found on payment gateway (%s)", contextPrefix, statusMessage))
	case "400":
		return apperror.NewBadRequest(fmt.Sprintf("%s: invalid payment gateway request (%s)", contextPrefix, statusMessage))
	case "406":
		return apperror.NewConflict(fmt.Sprintf("%s: transaction conflict on payment gateway (%s)", contextPrefix, statusMessage))
	case "412":
		return apperror.NewBadRequest(fmt.Sprintf("%s: transaction status cannot be modified on gateway (%s)", contextPrefix, statusMessage))
	default:
		if strings.HasPrefix(statusCode, "2") {
			return nil
		}
		return fmt.Errorf("%s: %s (status %s)", contextPrefix, statusMessage, statusCode)
	}
}
