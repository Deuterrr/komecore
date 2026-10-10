package midtrans

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"komecore/internal/apperror"
	config "komecore/internal/config"
	"komecore/internal/infra/paymentgateway"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestProvider(t *testing.T, handler http.HandlerFunc) (*midtransAPIProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	p, err := NewMidtransAPIProvider(config.MidTransConfig{
		ServerKey: "test-server-key",
		URL:       srv.URL,
	})
	if err != nil {
		t.Fatalf("NewMidtransAPIProvider: %v", err)
	}
	return p, srv
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewMidtransAPIProvider_MissingServerKey(t *testing.T) {
	_, err := NewMidtransAPIProvider(config.MidTransConfig{ServerKey: ""})
	if err == nil {
		t.Fatal("expected error for empty server key, got nil")
	}
	if !strings.Contains(err.Error(), "server key is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewMidtransAPIProvider_WhitespaceServerKey(t *testing.T) {
	_, err := NewMidtransAPIProvider(config.MidTransConfig{ServerKey: "   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only server key, got nil")
	}
}

func TestNewMidtransAPIProvider_URLCleaning(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://api.sandbox.midtrans.com/v2/charge", "https://api.sandbox.midtrans.com"},
		{"https://api.sandbox.midtrans.com/v2", "https://api.sandbox.midtrans.com"},
		{"https://api.sandbox.midtrans.com/", "https://api.sandbox.midtrans.com"},
		{"https://api.sandbox.midtrans.com", "https://api.sandbox.midtrans.com"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			p, err := NewMidtransAPIProvider(config.MidTransConfig{
				ServerKey: "key",
				URL:       tc.raw,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.baseURL != tc.want {
				t.Errorf("baseURL = %q, want %q", p.baseURL, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildChargeBody (white-box)
// ---------------------------------------------------------------------------

func TestBuildChargeBody_BankTransfer_Mandiri(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      150000,
		PaymentType: "mandiri",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["payment_type"] != "echannel" {
		t.Errorf("payment_type = %v, want echannel", m["payment_type"])
	}
	echannel, ok := m["echannel"].(map[string]any)
	if !ok {
		t.Fatal("echannel field missing or wrong type")
	}
	if echannel["bill_info1"] != "Payment:" {
		t.Errorf("bill_info1 = %v, want Payment:", echannel["bill_info1"])
	}
}

func TestBuildChargeBody_QRIS(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      50000,
		PaymentType: "qris",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["payment_type"] != "qris" {
		t.Errorf("payment_type = %v, want qris", m["payment_type"])
	}
}

func TestBuildChargeBody_GoPay(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      75000,
		PaymentType: "gopay",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["payment_type"] != "gopay" {
		t.Errorf("payment_type = %v, want gopay", m["payment_type"])
	}
	gp, ok := m["gopay"].(map[string]any)
	if !ok {
		t.Fatal("gopay field missing")
	}
	if gp["enable_callback"] != true {
		t.Errorf("enable_callback = %v, want true", gp["enable_callback"])
	}
}

func TestBuildChargeBody_ShopeePay(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      30000,
		PaymentType: "shopeepay",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["payment_type"] != "shopeepay" {
		t.Errorf("payment_type = %v, want shopeepay", m["payment_type"])
	}
	sp, ok := m["shopeepay"].(map[string]any)
	if !ok || sp["callback_url"] == "" {
		t.Errorf("expected shopeepay callback_url, got %v", m["shopeepay"])
	}
}

func TestBuildChargeBody_BCA(t *testing.T) {
	p := &midtransAPIProvider{}
	for _, paymentType := range []string{"bca", "bca_va"} {
		body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
			OrderID:     uuid.New(),
			PaymentID:   uuid.New(),
			Amount:      50000,
			PaymentType: paymentType,
		})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", paymentType, err)
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if m["payment_type"] != "bank_transfer" {
			t.Errorf("%s: payment_type = %v, want bank_transfer", paymentType, m["payment_type"])
		}
		bt, ok := m["bank_transfer"].(map[string]any)
		if !ok || bt["bank"] != "bca" {
			t.Errorf("%s: expected bank_transfer.bank = bca, got %v", paymentType, m["bank_transfer"])
		}
	}
}

func TestBuildChargeBody_MandiriBill(t *testing.T) {
	p := &midtransAPIProvider{}
	for _, paymentType := range []string{"mandiri", "mandiri_bill"} {
		body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
			OrderID:     uuid.New(),
			PaymentID:   uuid.New(),
			Amount:      50000,
			PaymentType: paymentType,
		})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", paymentType, err)
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if m["payment_type"] != "echannel" {
			t.Errorf("%s: payment_type = %v, want echannel", paymentType, m["payment_type"])
		}
		echannel, ok := m["echannel"].(map[string]any)
		if !ok || echannel["bill_info1"] == "" {
			t.Errorf("%s: expected echannel bill_info1, got %v", paymentType, m["echannel"])
		}
	}
}

func TestBuildChargeBody_UnsupportedPaymentType(t *testing.T) {
	p := &midtransAPIProvider{}
	_, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "credit_card",
	})
	if err == nil {
		t.Fatal("expected error for unsupported payment type, got nil")
	}
}

func TestBuildChargeBody_QRCodeAlias(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qr_code",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	if m["payment_type"] != "qris" {
		t.Errorf("payment_type = %v, want qris", m["payment_type"])
	}
	qrisPayload, ok := m["qris"].(map[string]any)
	if !ok || qrisPayload["acquirer"] != "gopay" {
		t.Errorf("expected qris.acquirer = gopay, got %v", m["qris"])
	}
}

func TestBuildChargeBody_WithItems(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      200000,
		PaymentType: "gopay",
		Items: []paymentgateway.ChargeItem{
			{ID: "prod-1", Name: "Wireless Mouse", Quantity: 2, Price: 75000},
			{ID: "prod-2", Name: "Mechanical Keyboard", Quantity: 1, Price: 50000},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	items, ok := m["item_details"].([]any)
	if !ok || len(items) != 2 {
		t.Errorf("expected 2 item_details, got %v", m["item_details"])
	}
}

func TestBuildChargeBody_WithExpiry(t *testing.T) {
	p := &midtransAPIProvider{}
	expires := time.Now().Add(2 * time.Hour)
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
		ExpiresAt:   expires,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	expiry, ok := m["custom_expiry"].(map[string]any)
	if !ok {
		t.Fatal("custom_expiry field missing")
	}
	if expiry["unit"] != "minute" {
		t.Errorf("unit = %v, want minute", expiry["unit"])
	}
	dur, _ := expiry["expiry_duration"].(float64)
	if dur < 1 {
		t.Errorf("expiry_duration = %v, want >= 1", dur)
	}
}

func TestBuildChargeBody_ExpiredExpiryIsSkipped(t *testing.T) {
	p := &midtransAPIProvider{}
	body, err := p.buildChargeBody(paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // already in the past
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	if _, found := m["custom_expiry"]; found {
		t.Error("custom_expiry should not be present for past expiry times")
	}
}

// ---------------------------------------------------------------------------
// Charge — HTTP-mock based
// ---------------------------------------------------------------------------

func TestCharge_BankTransfer_VANumber(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/charge" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, bankTransferChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "200",
				StatusMessage:     "Success",
				TransactionID:     "tx-001",
				OrderID:           orderID.String(),
				GrossAmount:       "150000.00",
				PaymentType:       "bank_transfer",
				TransactionStatus: "pending",
				ExpiryTime:        "2026-07-02 19:00:00",
			},
			VaNumbers: []vaNumber{
				{Bank: "bca", VANumber: "1234567890"},
			},
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      150000,
		PaymentType: "mandiri",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GatewayTransactionID != "tx-001" {
		t.Errorf("GatewayTransactionID = %v, want tx-001", resp.GatewayTransactionID)
	}
	if resp.GrossAmount != 150000 {
		t.Errorf("GrossAmount = %v, want 150000", resp.GrossAmount)
	}
	if len(resp.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(resp.Instructions))
	}
	if resp.Instructions[0].Type != "bank_transfer" {
		t.Errorf("instruction Type = %v, want bank_transfer", resp.Instructions[0].Type)
	}
	if resp.Instructions[0].Value != "1234567890" {
		t.Errorf("instruction Value = %v, want 1234567890", resp.Instructions[0].Value)
	}
	if resp.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
}

func TestCharge_BankTransfer_PermataAndMandiriBillKey(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bankTransferChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "200",
				TransactionID:     "tx-002",
				OrderID:           orderID.String(),
				GrossAmount:       "200000.00",
				PaymentType:       "bank_transfer",
				TransactionStatus: "pending",
			},
			PermataVANumber: "888001234567",
			BillKey:         "1234567",
			BillerCode:      "70012",
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      200000,
		PaymentType: "mandiri",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect PERMATA VA, BillKey, BillerCode = 3 instructions
	if len(resp.Instructions) != 3 {
		t.Errorf("expected 3 instructions, got %d: %+v", len(resp.Instructions), resp.Instructions)
	}
}

func TestCharge_QRIS_QRString(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, qrisChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "200",
				TransactionID:     "tx-003",
				OrderID:           orderID.String(),
				GrossAmount:       "50000.00",
				PaymentType:       "qris",
				TransactionStatus: "pending",
			},
			QRString: "00020101021226570011ID.DANA.WWW...",
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      50000,
		PaymentType: "qris",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Instructions) == 0 {
		t.Fatal("expected at least 1 instruction for QRIS")
	}
	if resp.Instructions[0].Type != "qris" {
		t.Errorf("instruction type = %v, want qris", resp.Instructions[0].Type)
	}
	if resp.Instructions[0].Value != "00020101021226570011ID.DANA.WWW..." {
		t.Errorf("instruction value mismatch: %v", resp.Instructions[0].Value)
	}
}

func TestCharge_QRIS_DeeplinkAction(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, qrisChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "200",
				TransactionID:     "tx-004",
				OrderID:           orderID.String(),
				GrossAmount:       "50000.00",
				PaymentType:       "qris",
				TransactionStatus: "pending",
			},
			Actions: []action{
				{Name: "deeplink-redirect", Method: "GET", URL: "https://deeplink.example.com"},
				{Name: "some-other-action", Method: "GET", URL: "https://other.example.com"},
			},
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      50000,
		PaymentType: "qris",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only deeplink-redirect should be extracted (not some-other-action)
	if len(resp.Instructions) != 1 {
		t.Errorf("expected 1 instruction, got %d", len(resp.Instructions))
	}
	if resp.Instructions[0].Type != "ewallet" {
		t.Errorf("instruction type = %v, want ewallet", resp.Instructions[0].Type)
	}
}

func TestCharge_GoPay_DeeplinkAndGenerateQRCode(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, gopayChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "200",
				TransactionID:     "tx-005",
				OrderID:           orderID.String(),
				GrossAmount:       "75000.00",
				PaymentType:       "gopay",
				TransactionStatus: "pending",
			},
			Actions: []action{
				{Name: "deeplink-redirect", Method: "GET", URL: "https://gopay.gojek.com/pay"},
				{Name: "generate-qr-code", Method: "GET", URL: "https://gopay.gojek.com/qr"},
				{Name: "get-status", Method: "GET", URL: "https://gopay.gojek.com/status"},
			},
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      75000,
		PaymentType: "gopay",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only deeplink-redirect and generate-qr-code should be extracted
	if len(resp.Instructions) != 2 {
		t.Errorf("expected 2 instructions, got %d: %+v", len(resp.Instructions), resp.Instructions)
	}
}

func TestCharge_ShopeePay(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, shopeepayChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "200",
				TransactionID:     "tx-006",
				OrderID:           orderID.String(),
				GrossAmount:       "30000.00",
				PaymentType:       "shopeepay",
				TransactionStatus: "pending",
			},
			Actions: []action{
				{Name: "deeplink-redirect", Method: "GET", URL: "https://shopee.co.id/pay"},
			},
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      30000,
		PaymentType: "shopeepay",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Instructions) != 1 {
		t.Errorf("expected 1 instruction, got %d", len(resp.Instructions))
	}
}

func TestCharge_BCA_VA(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bankTransferChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "201",
				TransactionID:     "tx-bca-001",
				OrderID:           orderID.String(),
				GrossAmount:       "50000.00",
				PaymentType:       "bank_transfer",
				TransactionStatus: "pending",
			},
			VaNumbers: []vaNumber{
				{Bank: "bca", VANumber: "12345678901"},
			},
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      50000,
		PaymentType: "bca_va",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(resp.Instructions))
	}
	if resp.Instructions[0].Type != "bank_transfer" || resp.Instructions[0].Value != "12345678901" {
		t.Errorf("unexpected instruction: %+v", resp.Instructions[0])
	}
}

func TestCharge_Mandiri_Bill(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bankTransferChargeAPIResponse{
			chargeAPIResponse: chargeAPIResponse{
				StatusCode:        "201",
				TransactionID:     "tx-mandiri-001",
				OrderID:           orderID.String(),
				GrossAmount:       "50000.00",
				PaymentType:       "echannel",
				TransactionStatus: "pending",
			},
			BillKey:    "9900000001",
			BillerCode: "70012",
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      50000,
		PaymentType: "mandiri_bill",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(resp.Instructions))
	}
	if resp.Instructions[0].Value != "9900000001" || resp.Instructions[1].Value != "70012" {
		t.Errorf("unexpected instructions: %+v", resp.Instructions)
	}
}

func TestCharge_GatewayNonSuccessStatus(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, chargeAPIResponse{
			StatusCode:    "400",
			StatusMessage: "One or more parameters in the payload is invalid",
		})
	}
	p, _ := newTestProvider(t, handler)

	_, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
	})
	if err == nil {
		t.Fatal("expected error for non-2xx gateway status, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeBadRequest {
		t.Errorf("expected BadRequest AppError, got: %v", err)
	}
}

func TestCharge_UnsupportedPaymentTypeNeverCallsHTTP(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}
	p, _ := newTestProvider(t, handler)

	_, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "credit_card",
	})
	if err == nil {
		t.Fatal("expected error for unsupported payment type")
	}
	if called {
		t.Error("HTTP handler should not have been called for unsupported payment type")
	}
}

func TestCharge_NetworkError(t *testing.T) {
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {})
	srv.Close() // close immediately to force network error

	_, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
	})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestCharge_ExpiryTimeParsedInWIB(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, chargeAPIResponse{
			StatusCode:        "200",
			TransactionID:     "tx-007",
			OrderID:           orderID.String(),
			GrossAmount:       "10000.00",
			PaymentType:       "qris",
			TransactionStatus: "pending",
			// This is WIB (UTC+7)
			ExpiryTime: "2026-07-02 19:00:00",
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ExpiresAt.IsZero() {
		t.Fatal("ExpiresAt should not be zero")
	}
	// WIB is UTC+7, so 19:00 WIB = 12:00 UTC
	wib, _ := time.LoadLocation("Asia/Jakarta")
	expectedUTC := time.Date(2026, 7, 2, 19, 0, 0, 0, wib).UTC()
	if !resp.ExpiresAt.UTC().Equal(expectedUTC) {
		t.Errorf("ExpiresAt = %v, want %v", resp.ExpiresAt.UTC(), expectedUTC)
	}
}

func TestCharge_InvalidExpiryTimeIsIgnored(t *testing.T) {
	orderID := uuid.New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, chargeAPIResponse{
			StatusCode:        "200",
			TransactionID:     "tx-008",
			OrderID:           orderID.String(),
			GrossAmount:       "10000.00",
			PaymentType:       "qris",
			TransactionStatus: "pending",
			ExpiryTime:        "not-a-date",
		})
	}
	p, _ := newTestProvider(t, handler)

	resp, err := p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     orderID,
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be zero for invalid expiry time format")
	}
}

func TestCharge_AuthorizationHeaderIsSent(t *testing.T) {
	var receivedAuth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		writeJSON(w, chargeAPIResponse{
			StatusCode:        "200",
			TransactionID:     "tx-009",
			OrderID:           uuid.New().String(),
			GrossAmount:       "10000.00",
			PaymentType:       "qris",
			TransactionStatus: "pending",
		})
	}
	p, _ := newTestProvider(t, handler)

	_, _ = p.Charge(context.Background(), paymentgateway.ChargeRequest{
		OrderID:     uuid.New(),
		PaymentID:   uuid.New(),
		Amount:      10000,
		PaymentType: "qris",
	})
	if !strings.HasPrefix(receivedAuth, "Basic ") {
		t.Errorf("Authorization header = %q, want Basic prefix", receivedAuth)
	}
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// ParseNotification — In-memory Cryptographic Signature Verification
// ---------------------------------------------------------------------------

func computeTestSignature(orderID, statusCode, grossAmount, serverKey string) string {
	raw := orderID + statusCode + grossAmount + serverKey
	hash := sha512.Sum512([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func TestParseNotification_Success_Settlement(t *testing.T) {
	serverKey := "test-server-key"
	p, err := NewMidtransAPIProvider(config.MidTransConfig{
		ServerKey: serverKey,
	})
	if err != nil {
		t.Fatalf("NewMidtransAPIProvider: %v", err)
	}

	orderID := uuid.New().String()
	statusCode := "200"
	grossAmount := "100000.00"
	sig := computeTestSignature(orderID, statusCode, grossAmount, serverKey)

	result, err := p.ParseNotification(context.Background(), paymentgateway.NotificationPayload{
		"order_id":           orderID,
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"signature_key":      sig,
		"transaction_id":     "tx-100",
		"transaction_status": "settlement",
		"fraud_status":       "accept",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != paymentgateway.NotificationStatusSettlement {
		t.Errorf("Status = %v, want settlement", result.Status)
	}
	if result.GrossAmount != 100000 {
		t.Errorf("GrossAmount = %v, want 100000", result.GrossAmount)
	}
	if result.GatewayTransactionID != "tx-100" {
		t.Errorf("GatewayTransactionID = %v, want tx-100", result.GatewayTransactionID)
	}
	if result.RawStatus != "settlement" {
		t.Errorf("RawStatus = %v, want settlement", result.RawStatus)
	}
}

func TestParseNotification_Success_Pending(t *testing.T) {
	serverKey := "test-server-key"
	p, err := NewMidtransAPIProvider(config.MidTransConfig{
		ServerKey: serverKey,
	})
	if err != nil {
		t.Fatalf("NewMidtransAPIProvider: %v", err)
	}

	orderID := uuid.New().String()
	statusCode := "201"
	grossAmount := "50000.00"
	sig := computeTestSignature(orderID, statusCode, grossAmount, serverKey)

	result, err := p.ParseNotification(context.Background(), paymentgateway.NotificationPayload{
		"order_id":           orderID,
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"signature_key":      sig,
		"transaction_id":     "tx-101",
		"transaction_status": "pending",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != paymentgateway.NotificationStatusPending {
		t.Errorf("Status = %v, want pending", result.Status)
	}
}

func TestParseNotification_FraudChallenge(t *testing.T) {
	serverKey := "test-server-key"
	p, err := NewMidtransAPIProvider(config.MidTransConfig{
		ServerKey: serverKey,
	})
	if err != nil {
		t.Fatalf("NewMidtransAPIProvider: %v", err)
	}

	orderID := uuid.New().String()
	statusCode := "200"
	grossAmount := "50000.00"
	sig := computeTestSignature(orderID, statusCode, grossAmount, serverKey)

	result, err := p.ParseNotification(context.Background(), paymentgateway.NotificationPayload{
		"order_id":           orderID,
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"signature_key":      sig,
		"transaction_id":     "tx-102",
		"transaction_status": "capture",
		"fraud_status":       "challenge",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != paymentgateway.NotificationStatusChallenge {
		t.Errorf("Status = %v, want challenge", result.Status)
	}
	if result.FraudStatus != "challenge" {
		t.Errorf("FraudStatus = %v, want challenge", result.FraudStatus)
	}
}

func TestParseNotification_MissingOrderID(t *testing.T) {
	p, _ := NewMidtransAPIProvider(config.MidTransConfig{ServerKey: "key"})

	_, err := p.ParseNotification(context.Background(), paymentgateway.NotificationPayload{})
	if err == nil {
		t.Fatal("expected error for missing order_id, got nil")
	}
	if !strings.Contains(err.Error(), "missing order_id") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseNotification_InvalidSignature(t *testing.T) {
	serverKey := "test-server-key"
	p, _ := NewMidtransAPIProvider(config.MidTransConfig{ServerKey: serverKey})

	orderID := uuid.New().String()
	statusCode := "200"
	grossAmount := "100000.00"

	_, err := p.ParseNotification(context.Background(), paymentgateway.NotificationPayload{
		"order_id":           orderID,
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"signature_key":      "invalid-signature-hash",
		"transaction_id":     "tx-invalid",
		"transaction_status": "settlement",
	})
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestParseNotification_MissingSignatureKey(t *testing.T) {
	p, _ := NewMidtransAPIProvider(config.MidTransConfig{ServerKey: "key"})

	_, err := p.ParseNotification(context.Background(), paymentgateway.NotificationPayload{
		"order_id":     uuid.New().String(),
		"status_code":  "200",
		"gross_amount": "100000.00",
	})
	if err == nil {
		t.Fatal("expected error for missing signature_key, got nil")
	}
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CancelTransaction — HTTP-mock based
// ---------------------------------------------------------------------------

func TestCancelTransaction_Success(t *testing.T) {
	gatewayOrderID := uuid.New().String()
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/cancel") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		called = true
		writeJSON(w, cancelAPIResponse{
			StatusCode:    "200",
			StatusMessage: "Success, transaction is canceled",
		})
	}
	p, _ := newTestProvider(t, handler)

	err := p.CancelTransaction(context.Background(), gatewayOrderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected HTTP handler to be called")
	}
}

func TestCancelTransaction_GatewayErrorStatus(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, cancelAPIResponse{
			StatusCode:    "412",
			StatusMessage: "Transaction status cannot be updated",
		})
	}
	p, _ := newTestProvider(t, handler)

	err := p.CancelTransaction(context.Background(), "some-order-id")
	if err == nil {
		t.Fatal("expected error for non-2xx cancel response, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeBadRequest {
		t.Errorf("expected BadRequest AppError for status 412, got: %v", err)
	}
}

func TestCancelTransaction_NetworkError(t *testing.T) {
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {})
	srv.Close()

	err := p.CancelTransaction(context.Background(), "some-order-id")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestCancelTransaction_CorrectURLPath(t *testing.T) {
	gatewayOrderID := "my-order-ref-123"
	var receivedPath string
	handler := func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(w, cancelAPIResponse{StatusCode: "200"})
	}
	p, _ := newTestProvider(t, handler)

	_ = p.CancelTransaction(context.Background(), gatewayOrderID)
	expected := "/v2/" + gatewayOrderID + "/cancel"
	if receivedPath != expected {
		t.Errorf("path = %q, want %q", receivedPath, expected)
	}
}

func TestSupports(t *testing.T) {
	p := &midtransAPIProvider{}

	cases := []struct {
		code string
		want bool
	}{
		{"gopay", true},
		{"GOPAY", true},
		{"shopeepay", true},
		{"qris", true},
		{"qr_code", true},
		{"mandiri", true},
		{"mandiri_bill", true},
		{"bca", true},
		{"bca_va", true},
		{"unsupported", false},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got := p.Supports(tc.code)
			if got != tc.want {
				t.Errorf("Supports(%q) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestAllowedPaymentMethods(t *testing.T) {
	p := &midtransAPIProvider{}
	methods := p.AllowedPaymentMethods()
	if len(methods) != 5 {
		t.Fatalf("expected 5 payment methods, got %d", len(methods))
	}

	expectedCodes := map[string]string{
		"gopay":        "ewallet",
		"shopeepay":    "ewallet",
		"qris":         "qr_code",
		"bca_va":       "bank_transfer",
		"mandiri_bill": "bank_transfer",
	}

	for _, m := range methods {
		expectedType, ok := expectedCodes[m.Code]
		if !ok {
			t.Errorf("unexpected payment method code: %s", m.Code)
			continue
		}
		if m.Type != expectedType {
			t.Errorf("method %s expected type %s, got %s", m.Code, expectedType, m.Type)
		}
		if m.Name == "" {
			t.Errorf("method %s has empty name", m.Code)
		}
	}
}
