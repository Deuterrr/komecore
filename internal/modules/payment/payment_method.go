package payment

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// Integration rail classification (virtual accounts, digital wallets, or dynamic QR codes).
type PaymentMethodType string

// Pricing surcharge model applied during checkout total calculation.
type PaymentFeeType string

const (
	// Virtual account banking rails with unique payment numbers (e.g. BCA, Mandiri, BRI).
	TypeBankTransfer PaymentMethodType = "bank_transfer"
	// Mobile wallet app redirects or deep links (e.g. GoPay, OVO, ShopeePay).
	TypeEWallet PaymentMethodType = "ewallet"
	// Dynamic EMVCo QRIS payment barcodes for mobile banking apps.
	TypeQRCode PaymentMethodType = "qr_code"

	// Fixed nominal fee added regardless of basket total.
	FeeTypeFlat PaymentFeeType = "flat"
	// Variable percentage surcharge calculated from total payable amount.
	FeeTypePercentage PaymentFeeType = "percentage"
	// Both a fixed nominal base fee and a variable percentage surcharge.
	FeeTypeMixed PaymentFeeType = "mixed"
)

// Active checkout channel configuration, pricing fee formulas, and customer guidance.
//
// Invariant: Conflict on (code, provider, type) revives soft-deleted records to preserve relational foreign keys.
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
	FeeBps        int64 // Surcharge in basis points (100 bps = 1.00%) for exact integer arithmetic.
	FeeMax        *int64

	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time

	Instruction *PaymentInstruction
}

// Asserts integrity of channel configuration and fee surcharge bounds.
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
		if pM.FeeBps < 0 || pM.FeeBps > 10000 || pM.FeePercentage < 0 || pM.FeePercentage > 1 {
			return ErrInvalidFeePercentage
		}

	case FeeTypeMixed:
		if pM.FeeFixed < 0 {
			return ErrInvalidFeeFixed
		}
		if pM.FeeBps < 0 || pM.FeeBps > 10000 || pM.FeePercentage < 0 || pM.FeePercentage > 1 {
			return ErrInvalidFeePercentage
		}
	}

	return nil
}

// Computes customer surcharge for a basket total using fixed-point integer math.
//
// Invariant: Capped by FeeMax when specified to prevent excessive percentage surcharges on large orders.
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

// Reports whether the rail is supported by domain channels.
func (t PaymentMethodType) IsValid() bool {
	switch t {
	case TypeBankTransfer, TypeEWallet, TypeQRCode:
		return true
	default:
		return false
	}
}

// Reports whether the fee calculation model is recognized.
func (t PaymentFeeType) IsValid() bool {
	switch t {
	case FeeTypeFlat, FeeTypePercentage, FeeTypeMixed:
		return true
	default:
		return false
	}
}
