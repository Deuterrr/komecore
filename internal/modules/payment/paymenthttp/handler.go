package paymenthttp

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	"komecore/internal/modules/payment"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	service *payment.Service
	webhook *payment.WebhookProcessor
}

func NewPaymentHandler(
	service *payment.Service,
	webhook *payment.WebhookProcessor,
) *PaymentHandler {
	return &PaymentHandler{
		service: service,
		webhook: webhook,
	}
}

func (h *PaymentHandler) UpdatePaymentMethodActive(w http.ResponseWriter, r *http.Request) error {
	idStr := chi.URLParam(r, "methodID")
	methodID, err := uuid.Parse(idStr)
	if err != nil {
		return apperror.NewBadRequest("invalid payment method ID")
	}

	var req updatePaymentMethodActiveRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	input := payment.SavePaymentMethodInput{
		ID:       methodID,
		IsActive: req.IsActive,
	}
	err = h.service.SavePaymentMethod(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "payment method successfully updated",
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *PaymentHandler) ListPaymentMethod(w http.ResponseWriter, r *http.Request) error {
	sortParam := httpx.Query(r, "sort")
	input := payment.ListPaymentMethodInput{
		Sort: sortParam,
	}

	payMethods, err := h.service.ListPaymentMethods(r.Context(), input)
	if err != nil {
		return err
	}

	paymentMthds := make([]paymentMethodResponse, 0, len(payMethods))
	for _, p := range payMethods {
		pM := paymentMethodResponse{
			ID:            p.ID,
			Name:          p.Name,
			Code:          p.Code,
			Provider:      p.Provider,
			Type:          string(p.Type),
			IsActive:      p.IsActive,
			Description:   p.Description,
			FeeType:       string(p.FeeType),
			FeeFixed:      p.FeeFixed,
			FeePercentage: p.FeePercentage,
			FeeMax:        p.FeeMax,
		}

		if p.Instruction != nil {
			pM.Instruction = &paymentInstructionResponse{
				ID:        p.Instruction.ID,
				Content:   p.Instruction.Content,
				CreatedAt: p.Instruction.CreatedAt,
			}
		}

		paymentMthds = append(paymentMthds, pM)
	}

	response := map[string][]paymentMethodResponse{
		"methods": paymentMthds,
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *PaymentHandler) HandleMidtransWebhook(w http.ResponseWriter, r *http.Request) error {
	var payload map[string]any
	if err := httpx.DecodeJSON(r, &payload); err != nil {
		return apperror.NewBadRequest("invalid payload")
	}

	if err := h.webhook.ProcessWebhook(r.Context(), payload); err != nil {
		return err
	}

	response := map[string]string{
		"message": "webhook processed successfully",
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *PaymentHandler) SavePaymentInstruction(w http.ResponseWriter, r *http.Request) error {
	methodIDStr := chi.URLParam(r, "methodID")
	methodID, err := uuid.Parse(methodIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid payment method ID")
	}

	var req savePaymentInstructionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}
	if req.Content == "" {
		return apperror.NewBadRequest("content cannot be empty")
	}

	input := payment.SavePaymentInstructionInput{
		PaymentMethodID: methodID,
		Content:         req.Content,
	}
	err = h.service.SavePaymentInstruction(r.Context(), input)
	if err != nil {
		return err
	}

	response := map[string]string{
		"message": "payment instruction successfully saved",
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *PaymentHandler) GetMyOrderPayment(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	result, err := h.service.GetPaymentDetail(r.Context(), payment.GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})
	if err != nil {
		return err
	}

	resp := getPaymentDetailResponse{
		PaymentID: result.Payment.ID.String(),
		Status:    string(result.Payment.Status),
		Amount:    result.Payment.Amount,
		ExpiresAt: result.Payment.ExpiresAt,
	}

	if result.ChannelData != nil {
		channelTypeStr := string(result.ChannelData.ChannelType)
		resp.ChannelType = &channelTypeStr
		resp.DisplayName = &result.ChannelData.DisplayName
		resp.ActionURL = result.ChannelData.ActionURL

		resp.ChannelData = &paymentChannelDataResponse{
			ChannelType: channelTypeStr,
			DisplayName: result.ChannelData.DisplayName,
			ActionURL:   result.ChannelData.ActionURL,
			ExpiresAt:   result.ChannelData.ExpiresAt,
		}

		if result.ChannelData.AccountNumber != nil {
			resp.AccountName = &result.ChannelData.DisplayName
			resp.AccountNumber = result.ChannelData.AccountNumber
		} else if result.ChannelData.ChannelType == payment.TypeBankTransfer {
			resp.AccountName = &result.ChannelData.DisplayName
			resp.AccountNumber = result.ChannelData.ActionURL
		}

		if result.ChannelData.QRString != nil {
			resp.QRString = result.ChannelData.QRString
		} else if result.ChannelData.ChannelType == payment.TypeQRCode {
			resp.QRString = result.ChannelData.ActionURL
		}
	}

	resp.Instruction = result.Instruction
	httpx.WriteJSON(w, http.StatusOK, resp)
	return nil
}

// CheckMyOrderPaymentStatus is the customer-triggered payment sync endpoint.
//
// When a customer's payment appears stuck as 'pending' after they have paid,
// calling this endpoint immediately queries Midtrans for the current status
// and resolves the payment — without waiting for the background reconciler.
func (h *PaymentHandler) CheckMyOrderPaymentStatus(w http.ResponseWriter, r *http.Request) error {
	_, customerID, err := httpx.RequireCustomer(r)
	if err != nil {
		return err
	}

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return apperror.NewBadRequest("invalid order id")
	}

	input := payment.CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	}
	result, err := h.service.CheckPaymentStatus(r.Context(), input)
	if err != nil {
		return err
	}

	httpx.WriteJSON(w, http.StatusOK, checkPaymentStatusResponse{
		Status: string(result.Status),
		Synced: result.Synced,
	})
	return nil
}
