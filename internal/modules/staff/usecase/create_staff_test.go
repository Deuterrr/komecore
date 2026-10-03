package usecase

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	userDomain "komecore/internal/modules/user/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStaffUsecase_Success(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	uc := NewCreateStaffUsecase(staffR, userR, exec, tx, audit)

	input := CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "floral-logistics",
	}

	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 1, userR.createCalls)
	assert.Equal(t, 1, staffR.createCalls)
	assert.Len(t, audit.events, 1)
	assert.Equal(t, "create_staff_profile", audit.events[0].Action)
}

func TestCreateStaffUsecase_ValidationErrors(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	uc := NewCreateStaffUsecase(staffR, userR, exec, tx, audit)

	// Missing name
	err := uc.Execute(context.Background(), CreateStaffInput{
		Name:     "",
		Username: "floral-logistics",
	})
	assert.Error(t, err)
	var badReq *apperrors.AppError
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)

	// Missing username
	err = uc.Execute(context.Background(), CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "",
	})
	assert.Error(t, err)
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)
}

func TestCreateStaffUsecase_DuplicateUsername(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{
		user: &userDomain.User{
			Username: "existing-user",
		},
	}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	uc := NewCreateStaffUsecase(staffR, userR, exec, tx, audit)

	input := CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "existing-user",
	}

	err := uc.Execute(context.Background(), input)
	assert.Error(t, err)
	var appErr *apperrors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.StatusCode)
}
