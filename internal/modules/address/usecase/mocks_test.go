package usecase_test

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type mockExecutor struct {
	transaction.Executor
}

type mockTransactor struct {
	transaction.Transactor
	err error
}

func (m *mockTransactor) WithinTransaction(
	ctx context.Context,
	fn func(exec transaction.Executor) error,
) error {
	if m.err != nil {
		return m.err
	}
	return fn(&mockExecutor{})
}

type mockCustomerAddressRepo struct {
	repository.CustomerAddressRepository
	addresses       map[uuid.UUID]domain.CustomerAddress
	count           *int
	getByIDError    error
	countError      error
	unsetDefaultErr error
	saveError       error
	deleteError     error
	listError       error

	getByIDCalls      int
	countCalls        int
	unsetDefaultCalls int
	saveCalls         int
	deleteCalls       int
	listCalls         int

	savedAddresses []domain.CustomerAddress
}

func newMockCustomerAddressRepo() *mockCustomerAddressRepo {
	return &mockCustomerAddressRepo{
		addresses: make(map[uuid.UUID]domain.CustomerAddress),
	}
}

func (m *mockCustomerAddressRepo) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	addressID uuid.UUID,
) (*domain.CustomerAddress, error) {
	m.getByIDCalls++
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	addr, exists := m.addresses[addressID]
	if !exists {
		return nil, nil
	}
	return &addr, nil
}

func (m *mockCustomerAddressRepo) CountByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*int, error) {
	m.countCalls++
	if m.countError != nil {
		return nil, m.countError
	}
	if m.count != nil {
		return m.count, nil
	}
	count := len(m.addresses)
	return &count, nil
}

func (m *mockCustomerAddressRepo) UnsetDefaultByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) error {
	m.unsetDefaultCalls++
	if m.unsetDefaultErr != nil {
		return m.unsetDefaultErr
	}
	return nil
}

func (m *mockCustomerAddressRepo) Save(
	ctx context.Context,
	exec transaction.Executor,
	address domain.CustomerAddress,
) error {
	m.saveCalls++
	if m.saveError != nil {
		return m.saveError
	}
	m.savedAddresses = append(m.savedAddresses, address)
	m.addresses[address.ID] = address
	return nil
}

func (m *mockCustomerAddressRepo) Delete(
	ctx context.Context,
	exec transaction.Executor,
	addressID uuid.UUID,
) error {
	m.deleteCalls++
	if m.deleteError != nil {
		return m.deleteError
	}
	delete(m.addresses, addressID)
	return nil
}

func (m *mockCustomerAddressRepo) ListByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) ([]domain.CustomerAddress, error) {
	m.listCalls++
	if m.listError != nil {
		return nil, m.listError
	}
	var res []domain.CustomerAddress
	for _, a := range m.addresses {
		if a.CustomerID == customerID {
			res = append(res, a)
		}
	}
	return res, nil
}
