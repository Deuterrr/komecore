package paymentusecase

import (
	"context"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	query "komecore/internal/shared/query"
)

type ListPaymentMethodUsecase struct {
	paymentMethodRepo paymentrepo.PaymentMethodRepository
	executor          transaction.Executor
}

func NewListPaymentMethodUsecase(
	paymentMethodRepo paymentrepo.PaymentMethodRepository,
	executor transaction.Executor,
) *ListPaymentMethodUsecase {
	return &ListPaymentMethodUsecase{
		paymentMethodRepo: paymentMethodRepo,
		executor:          executor,
	}
}

type ListPaymentMethodInput struct {
	Sort string
}

func (u *ListPaymentMethodUsecase) ListAll(
	ctx context.Context,
	input ListPaymentMethodInput,
) ([]paymentdomain.PaymentMethod, error) {
	var pmSortKeys = map[string]query.SortKey{
		"latest": paymentrepo.PaymentMethodSortLatest,
		"name":   paymentrepo.PaymentMethodSortName,
		"code":   paymentrepo.PaymentMethodSortCode,
		"type":   paymentrepo.PaymentMethodSortType,
	}

	var sorts query.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir query.SortDirection = query.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = query.SortAsc
				}
			}

			sortKey, exists := pmSortKeys[key]
			if exists {
				sorts = append(sorts, query.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = query.Sorts{
			{
				By:        paymentrepo.PaymentMethodSortLatest,
				Direction: query.SortDesc,
			},
		}
	}

	methods, err := u.paymentMethodRepo.ListAll(ctx, u.executor, sorts)
	if err != nil {
		return nil, fmt.Errorf("failed to load payment methods: %w", err)
	}

	return methods, nil
}
