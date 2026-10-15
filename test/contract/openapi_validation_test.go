package contract_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIValidation_CoreEndpoints(t *testing.T) {
	_, openapiRouter := loadOpenAPIRouter(t)
	server := newTestContractServer(t)
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Health Endpoint (Unversioned Infrastructure)",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Catalog Products Listing (Paginated Response Envelope)",
			method:         http.MethodGet,
			path:           "/api/v1/products?page=1&limit=10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Couriers List (Supported Shipping Providers)",
			method:         http.MethodGet,
			path:           "/api/v1/couriers",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Payment Methods List (Staff / Payment Options)",
			method:         http.MethodGet,
			path:           "/api/v1/payments/methods",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Revenue Analytics (Staff Admin Aggregate Metrics)",
			method:         http.MethodGet,
			path:           "/api/v1/staff/analytics/revenue?range=30d&interval=day",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Order Pipeline Analytics (Staff Admin Order Counts)",
			method:         http.MethodGet,
			path:           "/api/v1/staff/analytics/orders",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Validate Coupon (Bad Request Error Envelope)",
			method:         http.MethodPost,
			path:           "/api/v1/coupons/validate",
			body:           `{"code":""}`, // Missing subtotal and shipping_fee, triggers 400 Bad Request
			expectedStatus: http.StatusBadRequest,
		},
	}

	client := server.Client()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = bytes.NewBufferString(tc.body)
			}

			reqURL := server.URL + tc.path
			req, err := http.NewRequest(tc.method, reqURL, bodyReader)
			require.NoError(t, err)

			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "unexpected HTTP status for %s %s: %s", tc.method, tc.path, string(bodyBytes))

			// Construct matching request for openapi3filter validation (matching relative path)
			validationReq, err := http.NewRequest(tc.method, tc.path, bytes.NewReader(bodyBytes))
			require.NoError(t, err)
			if tc.body != "" {
				validationReq.Header.Set("Content-Type", "application/json")
			}

			validateHTTPResponse(t, openapiRouter, validationReq, resp, bodyBytes)
		})
	}
}
