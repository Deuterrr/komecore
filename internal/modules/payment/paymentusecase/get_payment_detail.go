package paymentusecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	markdown "komecore/internal/shared/markdown"

	"github.com/google/uuid"
)

type GetPaymentDetailUsecase struct {
	executor               transaction.Executor
	orderMgr               OrderPaymentManager
	paymentRepo            paymentrepo.PaymentRepository
	paymentMethodRepo      paymentrepo.PaymentMethodRepository
	paymentInstructionRepo paymentrepo.PaymentInstructionRepository
	paymentChannelDataRepo paymentrepo.PaymentChannelDataRepository
}

func NewGetPaymentDetailUsecase(
	executor transaction.Executor,
	orderMgr OrderPaymentManager,
	paymentRepo paymentrepo.PaymentRepository,
	paymentMethodRepo paymentrepo.PaymentMethodRepository,
	paymentInstructionRepo paymentrepo.PaymentInstructionRepository,
	paymentChannelDataRepo paymentrepo.PaymentChannelDataRepository,
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
	Payment     paymentdomain.Payment
	ChannelData *paymentdomain.PaymentChannelData
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
		return nil, apperrors.NewNotFound("order not found")
	}

	if input.CustomerID != nil &&
		order.CustomerID != *input.CustomerID {
		return nil, apperrors.NewNotFound("order not found")
	}

	payment, err := u.paymentRepo.GetByOrderID(ctx, u.executor, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment: %w", err)
	}
	if payment == nil {
		return nil, apperrors.NewNotFound("payment not found")
	}

	var (
		channelData *paymentdomain.PaymentChannelData
		vaNumber    string
		qrString    string
		redirectURL string
	)

	cd, err := u.paymentChannelDataRepo.GetByPaymentID(ctx, u.executor, payment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment channel data: %w", err)
	}

	channelData = cd
	if cd != nil {
		if cd.AccountNumber != nil {
			vaNumber = *cd.AccountNumber
		} else if cd.ChannelType == paymentdomain.TypeBankTransfer && cd.ActionURL != nil {
			vaNumber = *cd.ActionURL
		}
		if cd.QRString != nil {
			qrString = *cd.QRString
		} else if cd.ChannelType == paymentdomain.TypeQRCode && cd.ActionURL != nil {
			vaNumber = *cd.ActionURL
		}
		if cd.RedirectURL != nil {
			redirectURL = *cd.RedirectURL
		} else if cd.ChannelType == paymentdomain.TypeEWallet && cd.ActionURL != nil {
			redirectURL = *cd.ActionURL
		}
	}

	var renderedInstruction *string
	instruction, err := u.paymentInstructionRepo.GetByPaymentMethodID(ctx, u.executor, payment.MethodID)
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
		if payment.ExpiresAt != nil {
			expiredAtStr = payment.ExpiresAt.Format(time.RFC3339)
		} else {
			expiredAtStr = payment.CreatedAt.Add(24 * time.Hour).Format(time.RFC3339)
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

		content, err := markdown.Render(instruction.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to format payment instruction: %w", err)
		}
		renderedInstruction = &content
	}

	return &GetPaymentDetailResult{
		Payment:     *payment,
		ChannelData: channelData,
		Instruction: renderedInstruction,
	}, nil
}
