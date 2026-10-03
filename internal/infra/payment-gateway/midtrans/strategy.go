package midtrans

import (
	"encoding/json"
	"fmt"
	"strings"

	"komecore/internal/config"
	paymentgateway "komecore/internal/infra/payment-gateway"
)

const (
	// DEFAULT_MANDIRI_BILLER_CODE is the default 5-digit company code assigned
	// by Bank Mandiri to Midtrans for its multi-payment / echannel service.
	DEFAULT_MANDIRI_BILLER_CODE = "70012"
)

// channelStrategy defines the channel-specific encoding and decoding mechanics
// required by Midtrans Core API (e.g. e-wallets, bank virtual accounts, QRIS).
//
// Implementations are stateless and registered into the provider's strategy map.
type channelStrategy interface {
	// BuildPayload populates channel-specific JSON fields into the outgoing charge request body.
	//
	// Parameters:
	//   - req: The domain charge request containing order details, amounts, and customer info.
	//   - body: The mutable payload map sent to Midtrans /v2/charge.
	BuildPayload(req paymentgateway.ChargeRequest, body map[string]any)

	// ParseInstructions extracts user-actionable payment details (such as VA numbers,
	// bill keys, QR strings, or deep-link redirect URLs) from the gateway's raw response JSON.
	//
	// Returns an empty slice if the channel does not require customer action (e.g. auto-debit).
	ParseInstructions(respBody []byte) ([]paymentgateway.PaymentInstruction, error)
}

type (
	// bcaStrategy handles BCA virtual account payment method.
	bcaStrategy struct{}
	// mandiriStrategy handles Mandiri virtual account/bill payment.
	mandiriStrategy struct{}
	// gopayStrategy handles GoPay payment method.
	gopayStrategy struct{}
	// shopeepayStrategy handles ShopeePay payment method.
	shopeepayStrategy struct{}
	// qrisStrategy handles QRIS and QR Code payment methods.
	qrisStrategy struct{}
)

func (s *bcaStrategy) BuildPayload(req paymentgateway.ChargeRequest, body map[string]any) {
	body["payment_type"] = "bank_transfer"
	body["bank_transfer"] = map[string]any{
		"bank": "bca",
	}
}

func (s *mandiriStrategy) BuildPayload(req paymentgateway.ChargeRequest, body map[string]any) {
	body["payment_type"] = "echannel"
	body["echannel"] = map[string]any{
		"bill_info1": "Payment:",
		"bill_info2": "Order Payment",
	}
}

func (s *gopayStrategy) BuildPayload(req paymentgateway.ChargeRequest, body map[string]any) {
	body["payment_type"] = "gopay"
	body["gopay"] = map[string]any{
		"enable_callback": true,
	}
}

func (s *shopeepayStrategy) BuildPayload(req paymentgateway.ChargeRequest, body map[string]any) {
	body["payment_type"] = "shopeepay"
	frontendURL := config.GetEnv("FRONTEND_URL", "https://komecore.store")
	body["shopeepay"] = map[string]any{
		"callback_url": fmt.Sprintf("%s/orders/%s", strings.TrimRight(frontendURL, "/"), req.OrderID.String()),
	}
}

func (s *qrisStrategy) BuildPayload(req paymentgateway.ChargeRequest, body map[string]any) {
	body["payment_type"] = "qris"
	body["qris"] = map[string]any{
		"acquirer": "gopay",
	}
}

func (s *bcaStrategy) ParseInstructions(respBody []byte) ([]paymentgateway.PaymentInstruction, error) {
	var resp bankTransferChargeAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	var instructions []paymentgateway.PaymentInstruction
	for _, va := range resp.VaNumbers {
		if strings.EqualFold(va.Bank, "bca") || va.Bank == "" {
			instructions = append(instructions, paymentgateway.PaymentInstruction{
				Type:  "bank_transfer",
				Label: "BCA Virtual Account",
				Value: va.VANumber,
			})
		}
	}
	return instructions, nil
}

func (s *mandiriStrategy) ParseInstructions(respBody []byte) ([]paymentgateway.PaymentInstruction, error) {

	var resp bankTransferChargeAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	var instructions []paymentgateway.PaymentInstruction
	if resp.BillKey != "" {
		label := "MANDIRI Bill Key"
		if resp.BillerCode != "" {
			label = fmt.Sprintf("MANDIRI Bill Key (Biller Code: %s)", resp.BillerCode)
		}
		instructions = append(instructions, paymentgateway.PaymentInstruction{
			Type:  "bank_transfer",
			Label: label,
			Value: resp.BillKey,
		})
	}
	if resp.BillerCode != "" {
		instructions = append(instructions, paymentgateway.PaymentInstruction{
			Type:  "bank_transfer",
			Label: "MANDIRI Biller Code",
			Value: resp.BillerCode,
		})
	}
	if resp.PermataVANumber != "" {
		instructions = append(instructions, paymentgateway.PaymentInstruction{
			Type:  "bank_transfer",
			Label: "PERMATA Virtual Account",
			Value: resp.PermataVANumber,
		})
	}
	for _, va := range resp.VaNumbers {
		instructions = append(instructions, paymentgateway.PaymentInstruction{
			Type:  "bank_transfer",
			Label: strings.ToUpper(va.Bank) + " Virtual Account",
			Value: va.VANumber,
		})
	}
	return instructions, nil
}

func (s *gopayStrategy) ParseInstructions(respBody []byte) ([]paymentgateway.PaymentInstruction, error) {
	var resp gopayChargeAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	var instructions []paymentgateway.PaymentInstruction
	for _, act := range resp.Actions {
		if strings.EqualFold(act.Name, "deeplink-redirect") ||
			strings.EqualFold(act.Name, "generate-qr-code") {
			instructions = append(instructions, paymentgateway.PaymentInstruction{
				Type:  "ewallet",
				Label: act.Name,
				Value: act.URL,
			})
		}
	}
	return instructions, nil
}

func (s *shopeepayStrategy) ParseInstructions(respBody []byte) ([]paymentgateway.PaymentInstruction, error) {
	var resp shopeepayChargeAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	var instructions []paymentgateway.PaymentInstruction
	for _, act := range resp.Actions {
		if strings.EqualFold(act.Name, "deeplink-redirect") ||
			strings.EqualFold(act.Name, "generate-qr-code") {
			instructions = append(instructions, paymentgateway.PaymentInstruction{
				Type:  "ewallet",
				Label: act.Name,
				Value: act.URL,
			})
		}
	}
	return instructions, nil
}

func (s *qrisStrategy) ParseInstructions(respBody []byte) ([]paymentgateway.PaymentInstruction, error) {
	var resp qrisChargeAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	var instructions []paymentgateway.PaymentInstruction
	if resp.QRString != "" {
		instructions = append(instructions, paymentgateway.PaymentInstruction{
			Type:  "qris",
			Label: "QRIS",
			Value: resp.QRString,
		})
	}
	for _, act := range resp.Actions {
		if strings.EqualFold(act.Name, "deeplink-redirect") ||
			strings.EqualFold(act.Name, "generate-qr-code") {
			instructions = append(instructions, paymentgateway.PaymentInstruction{
				Type:  "ewallet",
				Label: act.Name,
				Value: act.URL,
			})
		}
	}
	return instructions, nil
}
