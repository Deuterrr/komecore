package paymentusecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment"
	"komecore/internal/strtpl"

	"github.com/google/uuid"
)

type GetPaymentDetailUsecase struct {
	executor               transaction.Executor
	orderMgr               OrderPaymentManager
	paymentRepo            payment.Repository
	paymentMethodRepo      payment.MethodRepository
	paymentInstructionRepo payment.InstructionRepository
	paymentChannelDataRepo payment.ChannelDataRepository
}

func NewGetPaymentDetailUsecase(
	executor transaction.Executor,
	orderMgr OrderPaymentManager,
	paymentRepo payment.Repository,
	paymentMethodRepo payment.MethodRepository,
	paymentInstructionRepo payment.InstructionRepository,
	paymentChannelDataRepo payment.ChannelDataRepository,
) *GetPaymentDetailUsecase {
	return &GetPaymentDetailUsecase{
		executor:               executor,
		orderMgr:               orderMgr,
		paymentRepo:            paymentRepo,
		paymentMethodRepo:      paymentMethodRepo,
		paymentInstructionRepo: paymentInstructionRepo,
		paymentChannelDataRepo: paymentChannelDataRepo,
	}
}

type GetPaymentDetailInput struct {
	OrderID uuid.UUID

	// Optional customer ownership enforcement
	CustomerID *uuid.UUID
}

type GetPaymentDetailResult struct {
	Payment     payment.Payment
	ChannelData *payment.PaymentChannelData
	Instruction *string
}

func (u *GetPaymentDetailUsecase) Execute(
	ctx context.Context,
	input GetPaymentDetailInput,
) (*GetPaymentDetailResult, error) {
	order, err := u.orderMgr.GetOrderForPayment(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve order: %w", err)
	}
	if order == nil {
		return nil, apperror.NewNotFound("order not found")
	}

	if input.CustomerID != nil &&
		order.CustomerID != *input.CustomerID {
		return nil, apperror.NewNotFound("order not found")
	}

	pmt, err := u.paymentRepo.GetByOrderID(ctx, u.executor, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment: %w", err)
	}
	if pmt == nil {
		return nil, apperror.NewNotFound("payment not found")
	}

	var (
		channelData *payment.PaymentChannelData
		vaNumber    string
		qrString    string
		redirectURL string
	)

	cd, err := u.paymentChannelDataRepo.GetByPaymentID(ctx, u.executor, pmt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment channel data: %w", err)
	}

	channelData = cd
	if cd != nil {
		if cd.AccountNumber != nil {
			vaNumber = *cd.AccountNumber
		} else if cd.ChannelType == payment.TypeBankTransfer && cd.ActionURL != nil {
			vaNumber = *cd.ActionURL
		}
		if cd.QRString != nil {
			qrString = *cd.QRString
		} else if cd.ChannelType == payment.TypeQRCode && cd.ActionURL != nil {
			vaNumber = *cd.ActionURL
		}
		if cd.RedirectURL != nil {
			redirectURL = *cd.RedirectURL
		} else if cd.ChannelType == payment.TypeEWallet && cd.ActionURL != nil {
			redirectURL = *cd.ActionURL
		}
	}

	var renderedInstruction *string
	instruction, err := u.paymentInstructionRepo.GetByPaymentMethodID(ctx, u.executor, pmt.MethodID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment instruction: %w", err)
	}

	if instruction != nil {
		invoiceNumber, err := u.orderMgr.GetInvoiceNumber(ctx, u.executor, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve invoice: %w", err)
		}
		if invoiceNumber == "" {
			invoiceNumber = order.Number
		}

		expiredAtStr := ""
		if pmt.ExpiresAt != nil {
			expiredAtStr = pmt.ExpiresAt.Format(time.RFC3339)
		} else {
			expiredAtStr = pmt.CreatedAt.Add(24 * time.Hour).Format(time.RFC3339)
		}

		vars := map[string]string{
			"invoice_number": invoiceNumber,
			"amount":         strconv.FormatInt(order.Total, 10),
			"expired_at":     expiredAtStr,
			"va_number":      vaNumber,
			"bill_key":       vaNumber,
			"qr_string":      qrString,
			"redirect_url":   redirectURL,
		}

		content, err := strtpl.Render(instruction.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to format payment instruction: %w", err)
		}
		renderedInstruction = &content
	}

	return &GetPaymentDetailResult{
		Payment:     *pmt,
		ChannelData: channelData,
		Instruction: renderedInstruction,
	}, nil
}
