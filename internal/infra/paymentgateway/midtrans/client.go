package midtrans

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type vaNumber struct {
	Bank     string `json:"bank"`
	VANumber string `json:"va_number"`
}

type action struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	URL    string `json:"url"`
}

type chargeAPIResponse struct {
	StatusCode        string `json:"status_code"`
	StatusMessage     string `json:"status_message"`
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	StaffID           string `json:"staff_id"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	ExpiryTime        string `json:"expiry_time"`

	Currency               string     `json:"currency,omitempty"`
	PermataVANumber        string     `json:"permata_va_number,omitempty"`
	VaNumbers              []vaNumber `json:"va_numbers,omitempty"`
	BillKey                string     `json:"bill_key,omitempty"`
	BillerCode             string     `json:"biller_code,omitempty"`
	Actions                []action   `json:"actions,omitempty"`
	QRString               string     `json:"qr_string,omitempty"`
	PaymentCode            string     `json:"payment_code,omitempty"`
	ChannelResponseCode    string     `json:"channel_response_code,omitempty"`
	ChannelResponseMessage string     `json:"channel_response_message,omitempty"`
}

type bankTransferChargeAPIResponse struct {
	chargeAPIResponse
	PermataVANumber string     `json:"permata_va_number"`
	VaNumbers       []vaNumber `json:"va_numbers"`
	BillKey         string     `json:"bill_key"`
	BillerCode      string     `json:"biller_code"`
}

type gopayChargeAPIResponse struct {
	chargeAPIResponse
	Actions []action `json:"actions"`
}

type qrisChargeAPIResponse struct {
	chargeAPIResponse
	Acquirer string   `json:"acquirer"`
	Actions  []action `json:"actions"`
	QRString string   `json:"qr_string"`
}

type shopeepayChargeAPIResponse struct {
	chargeAPIResponse
	Actions                []action `json:"actions"`
	ChannelResponseCode    string   `json:"channel_response_code"`
	ChannelResponseMessage string   `json:"channel_response_message"`
}

type statusAPIResponse struct {
	StatusCode        string `json:"status_code"`
	StatusMessage     string `json:"status_message"`
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
}

type cancelAPIResponse struct {
	StatusCode    string `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

// doRequest executes an authenticated HTTP request against the Midtrans Core API
// and returns the raw response body.
func (p *midtransAPIProvider) doRequest(
	ctx context.Context,
	method, url string,
	body []byte,
) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, method, url, bodyReader,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", p.authToken)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return respBody, nil
}
