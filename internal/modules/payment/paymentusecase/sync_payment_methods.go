package paymentusecase

import (
	"context"
	"fmt"

	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	"komecore/internal/pagination"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

// SyncPaymentMethodsUsecase synchronizes the system's payment methods
// with the active payment gateway provider.
//
// It registers newly supported methods, updates metadata for existing methods,
// and deactivates methods that are no longer allowed by the provider.
type SyncPaymentMethodsUsecase struct {
	methodRepo paymentrepo.PaymentMethodRepository
	executor   transaction.Executor
	gateway    paymentgateway.Provider
}

func NewSyncPaymentMethodsUsecase(
	methodRepo paymentrepo.PaymentMethodRepository,
	executor transaction.Executor,
	gateway paymentgateway.Provider,
) *SyncPaymentMethodsUsecase {
	return &SyncPaymentMethodsUsecase{
		methodRepo: methodRepo,
		executor:   executor,
		gateway:    gateway,
	}
}

func (u *SyncPaymentMethodsUsecase) Execute(ctx context.Context) error {
	allowedMethods := u.gateway.AllowedPaymentMethods()
	providerName := u.gateway.Name()

	existingMethods, err := u.methodRepo.ListAll(ctx, u.executor, pagination.Sorts{})
	if err != nil {
		return fmt.Errorf("failed to list existing payment methods: %w", err)
	}

	type key struct {
		code     string
		provider string
		mType    string
	}
	existingMap := make(map[key]*paymentdomain.PaymentMethod)
	for i := range existingMethods {
		m := &existingMethods[i]
		existingMap[key{
			code:     m.Code,
			provider: m.Provider,
			mType:    string(m.Type),
		}] = m
	}

	activeKeys := make(map[key]bool)
	for _, am := range allowedMethods {
		k := key{
			code:     am.Code,
			provider: providerName,
			mType:    am.Type,
		}
		activeKeys[k] = true

		existing, ok := existingMap[k]
		if !ok {
			newMethod := paymentdomain.PaymentMethod{
				ID:            uuid.New(),
				Name:          am.Name,
				Code:          am.Code,
				Provider:      providerName,
				Type:          paymentdomain.PaymentMethodType(am.Type),
				IsActive:      true,
				Description:   am.Description,
				FeeType:       paymentdomain.PaymentFeeType(am.FeeType),
				FeeFixed:      am.FeeFixed,
				FeePercentage: am.FeePercentage,
				FeeMax:        am.FeeMax,
				CreatedAt:     appclock.Now(),
			}
			if err := u.methodRepo.Save(ctx, u.executor, newMethod); err != nil {
				return fmt.Errorf("failed to save new payment method %s: %w", am.Code, err)
			}
		} else {
			updatedMethod := *existing
			updatedMethod.Name = am.Name
			updatedMethod.Description = am.Description
			updatedMethod.FeeType = paymentdomain.PaymentFeeType(am.FeeType)
			updatedMethod.FeeFixed = am.FeeFixed
			updatedMethod.FeePercentage = am.FeePercentage
			updatedMethod.FeeMax = am.FeeMax

			if err := u.methodRepo.Save(ctx, u.executor, updatedMethod); err != nil {
				return fmt.Errorf("failed to update payment method %s: %w", am.Code, err)
			}
		}
	}

	for k, method := range existingMap {
		if !activeKeys[k] && method.IsActive {
			method.IsActive = false
			if err := u.methodRepo.Save(ctx, u.executor, *method); err != nil {
				return fmt.Errorf("failed to deactivate obsolete payment method %s: %w", method.Code, err)
			}
		}
	}

	return nil
}
