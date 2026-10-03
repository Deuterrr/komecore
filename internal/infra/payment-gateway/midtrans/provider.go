package midtrans

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	config "komecore/internal/config"
	paymentgateway "komecore/internal/infra/payment-gateway"
	appclock "komecore/pkg/clock"
)

type midtransAPIProvider struct {
	cfg        config.MidTransConfig
	client     *http.Client
	baseURL    string
	authToken  string
	strategies map[string]channelStrategy
}

// NewMidtransAPIProvider creates an instance of the Midtrans API Provider using Core API.
func NewMidtransAPIProvider(cfg config.MidTransConfig) (*midtransAPIProvider, error) {
	if strings.TrimSpace(cfg.ServerKey) == "" {
		return nil, fmt.Errorf("midtrans: server key is required")
	}

	var baseURL string
	if strings.TrimSpace(cfg.URL) != "" {
		cleanedURL := strings.TrimRight(cfg.URL, "/")
		cleanedURL = strings.TrimSuffix(cleanedURL, "/v2/charge")
		cleanedURL = strings.TrimSuffix(cleanedURL, "/v2")
		baseURL = cleanedURL
	} else if cfg.IsProduction {
		baseURL = "https://api.midtrans.com"
	} else {
		baseURL = "https://api.sandbox.midtrans.com"
	}

	authToken := "Basic " +
		base64.StdEncoding.EncodeToString(
			[]byte(cfg.ServerKey+":"),
		)

	p := &midtransAPIProvider{
		cfg:       cfg,
		client:    &http.Client{Timeout: 30 * time.Second},
		baseURL:   baseURL,
		authToken: authToken,
	}

	p.strategies = map[string]channelStrategy{
		"gopay":        &gopayStrategy{},
		"shopeepay":    &shopeepayStrategy{},
		"qris":         &qrisStrategy{},
		"qr_code":      &qrisStrategy{},
		"mandiri":      &mandiriStrategy{},
		"mandiri_bill": &mandiriStrategy{},
		"bca":          &bcaStrategy{},
		"bca_va":       &bcaStrategy{},
	}

	return p, nil
}

func (p *midtransAPIProvider) getStrategy(code string) channelStrategy {
	code = strings.ToLower(code)
	if p.strategies == nil {
		p.strategies = map[string]channelStrategy{
			"gopay":        &gopayStrategy{},
			"shopeepay":    &shopeepayStrategy{},
			"qris":         &qrisStrategy{},
			"qr_code":      &qrisStrategy{},
			"mandiri":      &mandiriStrategy{},
			"mandiri_bill": &mandiriStrategy{},
			"bca":          &bcaStrategy{},
			"bca_va":       &bcaStrategy{},
		}
	}
	return p.strategies[code]
}

func (p *midtransAPIProvider) Supports(code string) bool {
	return p.getStrategy(code) != nil
}

func (p *midtransAPIProvider) Name() string {
	return "midtrans"
}

func (p *midtransAPIProvider) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod {
	return []paymentgateway.AllowedPaymentMethod{
		{
			Code:          "gopay",
			Name:          "GoPay",
			Type:          "ewallet",
			FeeType:       "percentage",
			FeePercentage: 0.02,
			Description:   "Pay using GoPay e-wallet",
		},
		{
			Code:          "shopeepay",
			Name:          "ShopeePay",
			Type:          "ewallet",
			FeeType:       "percentage",
			FeePercentage: 0.02,
			Description:   "Pay using ShopeePay e-wallet",
		},
		{
			Code:        "qris",
			Name:        "QRIS",
			Type:        "qr_code",
			FeeType:     "flat",
			FeeFixed:    1000,
			Description: "Pay using QRIS QR Code",
		},
		{
			Code:        "bca_va",
			Name:        "BCA Virtual Account",
			Type:        "bank_transfer",
			FeeType:     "flat",
			FeeFixed:    4000,
			Description: "Pay using BCA Virtual Account",
		},
		{
			Code:        "mandiri_bill",
			Name:        "Mandiri Bill",
			Type:        "bank_transfer",
			FeeType:     "flat",
			FeeFixed:    4000,
			Description: "Pay using Mandiri Bill / VA",
		},
	}
}

func (p *midtransAPIProvider) Charge(
	ctx context.Context,
	req paymentgateway.ChargeRequest,
) (*paymentgateway.ChargeResponse, error) {
	strategy := p.getStrategy(req.PaymentType)
	if strategy == nil {
		return nil, fmt.Errorf("midtrans: unsupported payment method %q", req.PaymentType)
	}

	body, err := p.buildChargeBody(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans: build charge request: %w", err)
	}

	url := p.baseURL + "/v2/charge"
	respBody, err := p.doRequest(
		ctx,
		http.MethodPost,
		url,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("midtrans: charge transaction: %w", err)
	}

	var baseResp chargeAPIResponse
	if err := json.Unmarshal(respBody, &baseResp); err != nil {
		return nil, fmt.Errorf("midtrans: unmarshal base charge response: %w", err)
	}

	if err := mapProviderStatusCode(
		baseResp.StatusCode,
		baseResp.StatusMessage,
		fmt.Sprintf("midtrans charge order %q", req.OrderID),
	); err != nil {
		return nil, err
	}

	instructions, err := strategy.ParseInstructions(respBody)
	if err != nil {
		return nil, fmt.Errorf("midtrans: parse instructions: %w", err)
	}

	grossAmount, err := parseAmount(baseResp.GrossAmount)
	if err != nil {
		return nil, fmt.Errorf("parse gross_amount: %w", err)
	}

	var expiresAt time.Time
	if baseResp.ExpiryTime != "" {
		t, parseErr := time.ParseInLocation("2006-01-02 15:04:05", baseResp.ExpiryTime, appclock.Location())
		if parseErr == nil {
			expiresAt = t
		}
	}

	var (
		accountNumber *string
		qrString      *string
		redirectURL   *string
	)
	for _, inst := range instructions {
		if inst.Value == "" {
			continue
		}
		v := inst.Value
		switch inst.Type {
		case "bank_transfer":
			if accountNumber == nil {
				accountNumber = &v
			}
		case "qris":
			if qrString == nil {
				qrString = &v
			}
		case "ewallet":
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				if redirectURL == nil {
					redirectURL = &v
				}
			} else {
				if qrString == nil {
					qrString = &v
				}
			}
		}
	}

	result := paymentgateway.ChargeResponse{
		GatewayTransactionID: baseResp.TransactionID,
		GatewayOrderID:       baseResp.OrderID,
		PaymentType:          baseResp.PaymentType,
		GrossAmount:          grossAmount,
		Status:               baseResp.TransactionStatus,
		ExpiresAt:            expiresAt,
		Instructions:         instructions,
		AccountNumber:        accountNumber,
		QRString:             qrString,
		RedirectURL:          redirectURL,
	}

	return &result, nil
}

func (p *midtransAPIProvider) GetTransactionStatus(
	ctx context.Context,
	gatewayOrderID string,
) (*paymentgateway.NotificationResult, error) {
	if gatewayOrderID == "" {
		return nil, fmt.Errorf("midtrans: GetTransactionStatus: gatewayOrderID is required")
	}
	return p.fetchTransactionStatus(ctx, gatewayOrderID)
}

func (p *midtransAPIProvider) fetchTransactionStatus(
	ctx context.Context,
	orderID string,
) (*paymentgateway.NotificationResult, error) {
	url := p.baseURL + "/v2/" + orderID + "/status"
	respBody, err := p.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("midtrans: check transaction %q: %w", orderID, err)
	}

	var result statusAPIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("midtrans: unmarshal status response: %w", err)
	}

	if err := mapProviderStatusCode(
		result.StatusCode,
		result.StatusMessage,
		fmt.Sprintf("midtrans check transaction %q", orderID),
	); err != nil {
		return nil, err
	}

	grossAmount, err := parseAmount(result.GrossAmount)
	if err != nil {
		return nil, fmt.Errorf("midtrans: parse notification gross_amount: %w", err)
	}

	return &paymentgateway.NotificationResult{
		GatewayTransactionID: result.TransactionID,
		GatewayOrderID:       result.OrderID,
		Status:               mapNotificationStatus(result.TransactionStatus, result.FraudStatus),
		GrossAmount:          grossAmount,
		FraudStatus:          result.FraudStatus,
		RawStatus:            result.TransactionStatus,
	}, nil
}

func (p *midtransAPIProvider) CancelTransaction(
	ctx context.Context,
	gatewayOrderID string,
) error {
	url := p.baseURL + "/v2/" + gatewayOrderID + "/cancel"
	respBody, err := p.doRequest(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("midtrans: cancel transaction %q: %w",
			gatewayOrderID, err)
	}

	var resp cancelAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("midtrans: unmarshal cancel response: %w", err)
	}

	if err := mapProviderStatusCode(
		resp.StatusCode,
		resp.StatusMessage,
		fmt.Sprintf("midtrans cancel transaction %q", gatewayOrderID),
	); err != nil {
		return err
	}

	return nil
}

func (p *midtransAPIProvider) RefundTransaction(
	ctx context.Context,
	req paymentgateway.RefundRequest,
) (*paymentgateway.RefundResponse, error) {
	url := fmt.Sprintf("%s/v2/%s/refund", p.baseURL, req.GatewayOrderID)
	refundKey := fmt.Sprintf("refund-%s-%d", req.GatewayOrderID, appclock.Now().Unix())
	reqBodyMap := map[string]any{
		"refund_key": refundKey,
		"amount":     req.RefundAmount,
		"reason":     req.Reason,
	}
	bodyBytes, err := json.Marshal(reqBodyMap)
	if err != nil {
		return nil, fmt.Errorf("midtrans: marshal refund req: %w", err)
	}

	respBody, err := p.doRequest(ctx, http.MethodPost, url, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("midtrans: refund transaction %q: %w", req.GatewayOrderID, err)
	}

	var resp struct {
		StatusCode    string `json:"status_code"`
		StatusMessage string `json:"status_message"`
		TransactionID string `json:"transaction_id"`
		OrderID       string `json:"order_id"`
		GrossAmount   string `json:"gross_amount"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("midtrans: unmarshal refund response: %w", err)
	}

	if err := mapProviderStatusCode(
		resp.StatusCode,
		resp.StatusMessage,
		fmt.Sprintf("midtrans refund transaction %q", req.GatewayOrderID),
	); err != nil {
		return nil, err
	}

	amount, _ := parseAmount(resp.GrossAmount)
	return &paymentgateway.RefundResponse{
		GatewayTransactionID: resp.TransactionID,
		GatewayOrderID:       resp.OrderID,
		RefundAmount:         amount,
		Status:               resp.StatusCode,
	}, nil
}

func (p *midtransAPIProvider) buildChargeBody(
	req paymentgateway.ChargeRequest,
) ([]byte, error) {
	body := map[string]any{
		"transaction_details": map[string]any{
			"order_id":     req.OrderID.String(),
			"gross_amount": req.Amount,
		},
		"customer_details": map[string]any{
			"first_name": req.CustomerName,
			"email":      req.CustomerEmail,
			"phone":      req.CustomerPhone,
		},
	}

	if len(req.Items) > 0 {
		itemDetails := make([]map[string]any, 0, len(req.Items))
		for _, item := range req.Items {
			itemDetails = append(itemDetails, map[string]any{
				"id":       item.ID,
				"price":    item.Price,
				"quantity": item.Quantity,
				"name":     item.Name,
			})
		}
		body["item_details"] = itemDetails
	}

	if !req.ExpiresAt.IsZero() {
		dur := time.Until(req.ExpiresAt)
		if dur > 0 {
			minutes := int(dur.Minutes())
			if minutes < 1 {
				minutes = 1
			}
			body["custom_expiry"] = map[string]any{
				"expiry_duration": minutes,
				"unit":            "minute",
			}
		}
	}

	strategy := p.getStrategy(req.PaymentType)
	if strategy == nil {
		return nil, fmt.Errorf("unsupported payment type %q", req.PaymentType)
	}

	strategy.BuildPayload(req, body)

	return json.Marshal(body)
}
