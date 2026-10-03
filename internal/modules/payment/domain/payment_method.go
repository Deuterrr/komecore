package domain

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type PaymentMethodType string
type PaymentFeeType string

const (
	TypeBankTransfer PaymentMethodType = "bank_transfer"
	TypeEWallet      PaymentMethodType = "ewallet"
	TypeQRCode       PaymentMethodType = "qr_code"

	FeeTypeFlat       PaymentFeeType = "flat"
	FeeTypePercentage PaymentFeeType = "percentage"
	FeeTypeMixed      PaymentFeeType = "mixed"
)

type PaymentMethod struct {
	ID uuid.UUID

	Name        string
	Code        string
	Provider    string
	Type        PaymentMethodType
	IsActive    bool
	Description string

	FeeType       PaymentFeeType
	FeeFixed      int64
	FeePercentage float64
	FeeBps        int64 // Fee in basis points (100 bps = 1.00%)
	FeeMax        *int64

	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time

	Instruction *PaymentInstruction
}

func (pM *PaymentMethod) Validate() error {
	if pM.Name == "" {
		return ErrInvalidName
	}

	if pM.Code == "" {
		return ErrInvalidCode
	}

	if pM.Provider == "" {
		return ErrInvalidProvider
	}

	if !pM.Type.IsValid() {
		return ErrInvalidType
	}

	if !pM.FeeType.IsValid() {
		return ErrInvalidFeeType
	}

	if pM.FeeMax != nil && *pM.FeeMax < 0 {
		return ErrInvalidFeeMax
	}

	switch pM.FeeType {
	case FeeTypeFlat:
		if pM.FeeFixed < 0 {
			return ErrInvalidFeeFixed
		}

	case FeeTypePercentage:
		if pM.FeeBps < 0 || pM.FeeBps > 10000 {
			if pM.FeePercentage < 0 || pM.FeePercentage > 1 {
				return ErrInvalidFeePercentage
			}
		}

	case FeeTypeMixed:
		if pM.FeeFixed < 0 {
			return ErrInvalidFeeFixed
		}
		if pM.FeeBps < 0 || pM.FeeBps > 10000 {
			if pM.FeePercentage < 0 || pM.FeePercentage > 1 {
				return ErrInvalidFeePercentage
			}
		}
	}

	return nil
}

func (pM PaymentMethod) CalculateFee(amount int64) int64 {
	feeBps := pM.FeeBps
	if feeBps == 0 && pM.FeePercentage > 0 {
		feeBps = int64(math.Round(pM.FeePercentage * 10000))
	}

	calcPctFee := func() int64 {
		if feeBps <= 0 {
			return 0
		}
		// Fixed-point integer arithmetic with half-up rounding:
		fee := (amount*feeBps + 5000) / 10000
		if pM.FeeMax != nil && *pM.FeeMax > 0 && fee > *pM.FeeMax {
			return *pM.FeeMax
		}
		return fee
	}

	switch pM.FeeType {
	case FeeTypeFlat:
		return pM.FeeFixed

	case FeeTypePercentage:
		return calcPctFee()

	case FeeTypeMixed:
		return pM.FeeFixed + calcPctFee()

	default:
		return 0
	}
}

func (t PaymentMethodType) IsValid() bool {
	switch t {
	case TypeBankTransfer, TypeEWallet, TypeQRCode:
		return true
	default:
		return false
	}
}

func (t PaymentFeeType) IsValid() bool {
	switch t {
	case FeeTypeFlat, FeeTypePercentage, FeeTypeMixed:
		return true
	default:
		return false
	}
}
