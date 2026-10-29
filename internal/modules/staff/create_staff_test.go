package staff

import (
	"context"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/modules/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStaff_Success(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	svc := NewService(exec, tx, staffR, nil, nil, userR, nil, nil, nil, audit)

	input := CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "floral-logistics",
	}

	err := svc.CreateStaff(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 1, userR.createCalls)
	assert.Equal(t, 1, staffR.createCalls)
	assert.Len(t, audit.events, 1)
	assert.Equal(t, "create_staff_profile", audit.events[0].Action)
}

func TestCreateStaff_ValidationErrors(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	svc := NewService(exec, tx, staffR, nil, nil, userR, nil, nil, nil, audit)

	// Missing name
	err := svc.CreateStaff(context.Background(), CreateStaffInput{
		Name:     "",
		Username: "floral-logistics",
	})
	assert.Error(t, err)
	var badReq *apperror.AppError
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)

	// Missing username
	err = svc.CreateStaff(context.Background(), CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "",
	})
	assert.Error(t, err)
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)
}

func TestCreateStaff_DuplicateUsername(t *testing.T) {
	staffR := &mockStaffRepo{}
	userR := &mockUserRepo{
		user: &user.User{
			Username: "existing-user",
		},
	}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	svc := NewService(exec, tx, staffR, nil, nil, userR, nil, nil, nil, audit)

	input := CreateStaffInput{
		Name:     "Floral Logistics",
		Username: "existing-user",
	}

	err := svc.CreateStaff(context.Background(), input)
	assert.Error(t, err)
	var appErr *apperror.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.StatusCode)
}
