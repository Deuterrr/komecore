package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type postmanCollection struct {
	Info struct {
		Name string `json:"name"`
	} `json:"info"`
	Item []struct {
		Name string `json:"name"`
		Item []struct {
			Name    string `json:"name"`
			Request struct {
				Method string `json:"method"`
				URL    struct {
					Raw  string   `json:"raw"`
					Path []string `json:"path"`
				} `json:"url"`
				Body struct {
					Mode string `json:"mode"`
					Raw  string `json:"raw"`
				} `json:"body"`
			} `json:"request"`
		} `json:"item"`
	} `json:"item"`
}

type openAPISpec struct {
	OpenAPI string         `yaml:"openapi"`
	Info    map[string]any `yaml:"info"`
	Paths   map[string]any `yaml:"paths"`
	Comp    map[string]any `yaml:"components"`
}

func TestAPIContract_PostmanCollectionCoverage(t *testing.T) {
	// Locate api/postman_collection.json from internal/bootstrap directory
	postmanPath := filepath.Join("..", "..", "api", "postman_collection.json")
	data, err := os.ReadFile(postmanPath)
	require.NoError(t, err, "postman collection must exist")

	var col postmanCollection
	err = json.Unmarshal(data, &col)
	require.NoError(t, err, "postman collection must be valid JSON")

	assert.Equal(t, "Komecore API v1", col.Info.Name)
	assert.GreaterOrEqual(t, len(col.Item), 10, "should have at least 10 domain folders")

	totalRequests := 0
	for _, folder := range col.Item {
		totalRequests += len(folder.Item)
		for _, req := range folder.Item {
			require.NotEmpty(t, req.Request.Method, "method must not be empty for %s", req.Name)
			require.NotEmpty(t, req.Request.URL.Raw, "raw url must not be empty for %s", req.Name)

			// If JSON raw body is provided, it must be valid JSON
			if req.Request.Body.Mode == "raw" && req.Request.Body.Raw != "" {
				var jsonCheck map[string]any
				err := json.Unmarshal([]byte(req.Request.Body.Raw), &jsonCheck)
				assert.NoError(t, err, "body for %s must be valid JSON: %v", req.Name, err)
			}
		}
	}

	assert.GreaterOrEqual(t, totalRequests, 45, "must document at least 45 endpoints")
}

func TestAPIContract_OpenAPISpecValidation(t *testing.T) {
	openapiPath := filepath.Join("..", "..", "api", "openapi.yaml")
	data, err := os.ReadFile(openapiPath)
	require.NoError(t, err, "openapi.yaml must exist")

	var spec openAPISpec
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "openapi.yaml must be valid YAML")

	assert.True(t, strings.HasPrefix(spec.OpenAPI, "3.0"), "must be OpenAPI 3.0.x")
	assert.NotEmpty(t, spec.Paths, "paths must not be empty")

	// Verify key endpoints exist
	expectedPaths := []string{
		"/health",
		"/api/v1/auth/signin",
		"/api/v1/auth/signup",
		"/api/v1/auth/verify",
		"/api/v1/profile",
		"/api/v1/products",
		"/api/v1/shops",
		"/api/v1/carts",
		"/api/v1/carts/checkout",
		"/api/v1/order",
		"/api/v1/orders",
		"/api/v1/shipping/cost",
		"/api/v1/payments/methods",
		"/api/v1/users/me/wishlist",
	}

	for _, ep := range expectedPaths {
		_, ok := spec.Paths[ep]
		assert.True(t, ok, "path %s must be defined in openapi.yaml", ep)
	}

	// Verify schemas exist under components
	schemas, ok := spec.Comp["schemas"].(map[string]any)
	require.True(t, ok, "components.schemas must be defined")

	expectedSchemas := []string{
		"SignUpRequest",
		"SignInEmailRequest",
		"VerifyAccountRequest",
		"SaveProductRequest",
		"AddItemRequest",
		"CheckoutRequest",
		"CreateOrderRequest",
		"EstimateShippingOptionsRequest",
		"OrderResponse",
		"CartResponse",
		"WishlistResponse",
	}

	for _, s := range expectedSchemas {
		_, hasSchema := schemas[s]
		assert.True(t, hasSchema, "schema %s must be defined in openapi.yaml", s)
	}
}

func TestAPIContract_DTOKeySampleValidation(t *testing.T) {
	// Test SignUp DTO payload
	signUpJSON := `{"name":"John Doe","username":"johndoe","email":"john@example.com","password":"Password123!","phone":"+6281234567890"}`
	var signUpParsed struct {
		Name     string  `json:"name"`
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Password string  `json:"password"`
		Phone    *string `json:"phone"`
	}
	require.NoError(t, json.Unmarshal([]byte(signUpJSON), &signUpParsed))
	assert.Equal(t, "John Doe", signUpParsed.Name)
	assert.Equal(t, "johndoe", signUpParsed.Username)

	// Test Checkout DTO payload
	checkoutJSON := `{
		"shops": [
			{
				"shop_id": "c0000000-0000-0000-0000-000000000001",
				"items": [
					{
						"product_id": "b0000000-0000-0000-0000-000000000001",
						"quantity": 2,
						"item_options": {"variant": "5kg"}
					}
				],
				"courier": {
					"code": "jne",
					"service": "REG"
				}
			}
		]
	}`
	var checkoutParsed struct {
		Shops []struct {
			ShopID string `json:"shop_id"`
			Items  []struct {
				ProductID   *string           `json:"product_id"`
				Quantity    int               `json:"quantity"`
				ItemOptions map[string]string `json:"item_options"`
			} `json:"items"`
			Courier struct {
				Code    string `json:"code"`
				Service string `json:"service"`
			} `json:"courier"`
		} `json:"shops"`
	}
	require.NoError(t, json.Unmarshal([]byte(checkoutJSON), &checkoutParsed))
	require.Len(t, checkoutParsed.Shops, 1)
	assert.Equal(t, "jne", checkoutParsed.Shops[0].Courier.Code)

	// Test Create Order DTO payload
	orderJSON := `{
		"address_id": "e0000000-0000-0000-0000-000000000001",
		"selected_payment": {
			"id": "d0000000-0000-0000-0000-000000000001"
		},
		"shops": [
			{
				"shop_id": "c0000000-0000-0000-0000-000000000001",
				"name": "Kome Central Store",
				"selected_courier": {
					"code": "jne",
					"service": "REG"
				},
				"items": [
					{
						"product_id": "b0000000-0000-0000-0000-000000000001",
						"name": "Organic Japonica Rice 5kg",
						"quantity": 2
					}
				]
			}
		]
	}`
	var orderParsed struct {
		AddressID       string `json:"address_id"`
		SelectedPayment struct {
			ID string `json:"id"`
		} `json:"selected_payment"`
		Shops []struct {
			ShopID string `json:"shop_id"`
			Name   string `json:"name"`
		} `json:"shops"`
	}
	require.NoError(t, json.Unmarshal([]byte(orderJSON), &orderParsed))
	_, err := uuid.Parse(orderParsed.SelectedPayment.ID)
	assert.NoError(t, err, "selected_payment.id must be a valid UUID")

	// Test Shipping Cost Estimation DTO payload
	shipCostJSON := `{"shop_id":"c0000000-0000-0000-0000-000000000001","origin":3173,"destination":3174,"weight":1000,"price_filter":"lowest"}`
	var shipCostParsed struct {
		ShopID      string  `json:"shop_id"`
		Origin      int     `json:"origin"`
		Destination int     `json:"destination"`
		Weight      int     `json:"weight"`
		PriceFilter *string `json:"price_filter"`
	}
	require.NoError(t, json.Unmarshal([]byte(shipCostJSON), &shipCostParsed))
	assert.Equal(t, 3173, shipCostParsed.Origin)
	assert.Equal(t, 1000, shipCostParsed.Weight)
}
