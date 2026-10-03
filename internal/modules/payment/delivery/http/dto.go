package http

import (
	"time"

	"github.com/google/uuid"
)

type updatePaymentMethodActiveRequest struct {
	IsActive bool `json:"is_active"`
}

type paymentInstructionResponse struct {
	ID        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type paymentMethodResponse struct {
	ID            uuid.UUID                   `json:"id"`
	Name          string                      `json:"name"`
	Code          string                      `json:"code"`
	Provider      string                      `json:"provider"`
	Type          string                      `json:"type"`
	IsActive      bool                        `json:"is_active"`
	Description   string                      `json:"description"`
	FeeType       string                      `json:"fee_type"`
	FeeFixed      int64                       `json:"fee_fixed"`
	FeePercentage float64                     `json:"fee_percentage"`
	FeeMax        *int64                      `json:"fee_max,omitempty"`
	Instruction   *paymentInstructionResponse `json:"instruction,omitempty"`
}

type savePaymentInstructionRequest struct {
	Content string `json:"content"`
}

type paymentChannelDataResponse struct {
	ChannelType string     `json:"channel_type"`
	DisplayName string     `json:"display_name"`
	ActionURL   *string    `json:"action_url,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type getPaymentDetailResponse struct {
	PaymentID     string                      `json:"payment_id"`
	Status        string                      `json:"status"`
	Amount        int64                       `json:"amount"`
	ExpiresAt     *time.Time                  `json:"expires_at,omitempty"`
	ChannelType   *string                     `json:"channel_type,omitempty"`
	DisplayName   *string                     `json:"display_name,omitempty"`
	ActionURL     *string                     `json:"action_url,omitempty"`
	ChannelData   *paymentChannelDataResponse `json:"channel_data,omitempty"`
	AccountName   *string                     `json:"account_name,omitempty"`
	AccountNumber *string                     `json:"account_number,omitempty"`
	QRString      *string                     `json:"qr_string,omitempty"`
	Instruction   *string                     `json:"instruction,omitempty"`
}

type checkPaymentStatusResponse struct {
	Status string `json:"status"`
	Synced bool   `json:"synced"`
}
