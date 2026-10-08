package staffusecase

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/user/userdomain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStaffUsecase_Success(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	uc := NewStaffService(exec, tx, staffR, nil, nil, userR, nil, nil, nil, audit)

	input := CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "floral-logistics",
	}

	err := uc.CreateStaff(context.Background(), input)
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

	uc := NewStaffService(exec, tx, staffR, nil, nil, userR, nil, nil, nil, audit)

	// Missing name
	err := uc.CreateStaff(context.Background(), CreateStaffInput{
		Name:     "",
		Username: "floral-logistics",
	})
	assert.Error(t, err)
	var badReq *apperrors.AppError
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)

	// Missing username
	err = uc.CreateStaff(context.Background(), CreateStaffInput{
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
		user: &userdomain.User{
			Username: "existing-user",
		},
	}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	uc := NewStaffService(exec, tx, staffR, nil, nil, userR, nil, nil, nil, audit)

	input := CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "existing-user",
	}

	err := uc.CreateStaff(context.Background(), input)
	assert.Error(t, err)
	var appErr *apperrors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.StatusCode)
}
