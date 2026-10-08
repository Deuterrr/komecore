package staffhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/common/authctx"
	"komecore/internal/modules/staff/staffdomain"
	"komecore/internal/modules/staff/staffrepo"
	"komecore/internal/modules/staff/staffusecase"
	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userrepo"
	applogger "komecore/pkg/logger"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockExec struct{}

func (m *mockExec) Exec(ctx context.Context, sql string, args ...any) (transaction.Result, error) {
	return transaction.NewResult(1), nil
}
func (m *mockExec) Query(ctx context.Context, sql string, args ...any) (transaction.Rows, error) {
	return nil, nil
}
func (m *mockExec) QueryRow(ctx context.Context, sql string, args ...any) transaction.Row {
	return &mockR{}
}

type mockR struct{}

func (r *mockR) Scan(dest ...any) error { return nil }

type mockTx struct{}

func (m *mockTx) WithinTransaction(ctx context.Context, fn func(exec transaction.Executor) error) error {
	return fn(&mockExec{})
}

type mockAuditor struct{}

func (m *mockAuditor) Log(ctx context.Context, event applogger.AuditEvent) {}

type testStaffRepo struct {
	staff *staffdomain.Staff
}

func (r *testStaffRepo) Create(ctx context.Context, exec transaction.Executor, staff staffdomain.Staff) error {
	return nil
}
func (r *testStaffRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffdomain.Staff, error) {
	return r.staff, nil
}
func (r *testStaffRepo) GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*staffdomain.StaffProfile, error) {
	return nil, nil
}
func (r *testStaffRepo) FindStaff(ctx context.Context, exec transaction.Executor, params staffrepo.FindStaffParams) ([]staffdomain.StaffProfile, int, error) {
	return nil, 0, nil
}
func (r *testStaffRepo) Update(ctx context.Context, exec transaction.Executor, staffID uuid.UUID, name string, logoUrl *string, bannerUrl *string) error {
	return nil
}
func (r *testStaffRepo) Delete(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error {
	return nil
}

type testMembershipRepo struct {
	membership *staffdomain.StaffMembership
	roles      []staffdomain.Role
	accounts   []staffdomain.StaffAccountMember
}

func (r *testMembershipRepo) GetByAccountID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID) (*staffdomain.StaffMembership, error) {
	return r.membership, nil
}
func (r *testMembershipRepo) GetByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) (*staffdomain.StaffMembership, error) {
	return r.membership, nil
}
func (r *testMembershipRepo) ListRolesByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) ([]staffdomain.Role, error) {
	return r.roles, nil
}
func (r *testMembershipRepo) Save(ctx context.Context, exec transaction.Executor, membership staffdomain.StaffMembership) error {
	return nil
}
func (r *testMembershipRepo) ListAccountsByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) ([]staffdomain.StaffAccountMember, error) {
	return r.accounts, nil
}
func (r *testMembershipRepo) DeleteByAccountID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID) error {
	return nil
}
func (r *testMembershipRepo) DeleteByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) error {
	return nil
}
func (r *testMembershipRepo) DeleteByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error {
	return nil
}

type testAccountRepo struct {
	account       *staffusecase.AccountInfo
	accountByUser *staffusecase.AccountInfo
}

func (r *testAccountRepo) GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*staffusecase.AccountInfo, error) {
	return nil, nil
}
func (r *testAccountRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffusecase.AccountInfo, error) {
	return r.account, nil
}
func (r *testAccountRepo) GetByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffusecase.AccountInfo, error) {
	return r.accountByUser, nil
}
func (r *testAccountRepo) CreateStaffAccount(ctx context.Context, exec transaction.Executor, input staffusecase.CreateAccountInput) error {
	return nil
}
func (r *testAccountRepo) DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	return nil
}
func (r *testAccountRepo) RevokeSessionsByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	return nil
}

type testUserRepo struct {
	user *userdomain.User
}

func (r *testUserRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*userdomain.User, error) {
	return r.user, nil
}
func (r *testUserRepo) GetByUsername(ctx context.Context, exec transaction.Executor, username string) (*userdomain.User, error) {
	return nil, nil
}
func (r *testUserRepo) CreateUser(ctx context.Context, exec transaction.Executor, props userrepo.CreateUserProps) error {
	return nil
}
func (r *testUserRepo) SaveProfile(ctx context.Context, exec transaction.Executor, props userrepo.SaveProfileProps) error {
	return nil
}
func (r *testUserRepo) Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error {
	return nil
}

type testUserDeletionService struct{}

func (s *testUserDeletionService) DeleteUserRecord(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	return nil
}


type testRoleRepo struct {
	role *staffdomain.Role
}

func (r *testRoleRepo) GetByCode(ctx context.Context, exec transaction.Executor, code staffdomain.RoleCode) (*staffdomain.Role, error) {
	return r.role, nil
}

func (r *testRoleRepo) GetRolesByAccountAndStaff(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) ([]staffdomain.Role, error) {
	if r.role != nil {
		return []staffdomain.Role{*r.role}, nil
	}
	return nil, nil
}

type testHasher struct{}

func (h *testHasher) Hash(p string) (string, error) { return "hash", nil }
func (h *testHasher) Compare(hash, p string) error  { return nil }

func setupTestHandler(staffID, accountID uuid.UUID) (*staffHandler, *testStaffRepo, *testMembershipRepo) {
	sRepo := &testStaffRepo{
		staff: &staffdomain.Staff{
			ID:     staffID,
			UserID: uuid.New(),
		},
	}
	mRepo := &testMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: accountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
		accounts: []staffdomain.StaffAccountMember{
			{
				AccountID: accountID,
				UserID:    uuid.New(),
				Email:     "jane@example.com",
				Name:      "Jane Doe",
				Username:  "janedoe",
				Role: staffdomain.Role{
					ID:   uuid.New(),
					Code: staffdomain.RoleStaffAdmin,
					Name: "Staff Admin",
				},
				CreatedAt: time.Now(),
			},
		},
	}
	aRepo := &testAccountRepo{
		account: &staffusecase.AccountInfo{
			ID:     uuid.New(),
			UserID: uuid.New(),
		},
	}
	uRepo := &testUserRepo{}
	rRepo := &testRoleRepo{
		role: &staffdomain.Role{ID: uuid.New(), Code: staffdomain.RoleStaff, Name: "Staff"},
	}
	hasher := &testHasher{}

	exec := &mockExec{}
	tx := &mockTx{}
	audit := &mockAuditor{}
	userDeletionSvc := &testUserDeletionService{}

	svc := staffusecase.NewStaffService(
		exec,
		tx,
		sRepo,
		mRepo,
		rRepo,
		uRepo,
		aRepo,
		hasher,
		userDeletionSvc,
		audit,
	)

	handler := NewStaffHandler(svc)
	return handler, sRepo, mRepo
}

func withActorContext(r *http.Request, accountID, staffID uuid.UUID) *http.Request {
	actor := &authctx.Actor{
		AccountID: accountID,
		StaffID:   &staffID,
		Roles: []authctx.Role{
			{Code: authctx.RoleStaffAdmin},
		},
	}
	return r.WithContext(authctx.WithActor(r.Context(), actor))
}

func TestHandler_ListStaffAccounts(t *testing.T) {
	staffID := uuid.New()
	accountID := uuid.New()
	handler, _, _ := setupTestHandler(staffID, accountID)

	req := httptest.NewRequest(http.MethodGet, "/staff/"+staffID.String()+"/accounts", nil)
	req = withActorContext(req, accountID, staffID)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", staffID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	err := handler.ListStaffAccounts(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}

	var resp listStaffAccountsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 1 || resp.StaffID != staffID {
		t.Errorf("unexpected response content: %+v", resp)
	}
}

func TestHandler_UpdateStaff(t *testing.T) {
	staffID := uuid.New()
	accountID := uuid.New()
	handler, _, _ := setupTestHandler(staffID, accountID)

	body, _ := json.Marshal(updateStaffRequest{
		Name: "New Branch Name",
	})
	req := httptest.NewRequest(http.MethodPut, "/staff/"+staffID.String(), bytes.NewReader(body))
	req = withActorContext(req, accountID, staffID)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", staffID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	err := handler.UpdateStaff(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}
}

func TestHandler_DeleteStaff(t *testing.T) {
	staffID := uuid.New()
	accountID := uuid.New()
	handler, _, _ := setupTestHandler(staffID, accountID)

	req := httptest.NewRequest(http.MethodDelete, "/staff/"+staffID.String(), nil)
	req = withActorContext(req, accountID, staffID)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", staffID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	err := handler.DeleteStaff(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}
}

func TestHandler_RemoveStaffAccount(t *testing.T) {
	staffID := uuid.New()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	handler, _, _ := setupTestHandler(staffID, actorAccountID)

	req := httptest.NewRequest(http.MethodDelete, "/staff/"+staffID.String()+"/accounts/"+targetAccountID.String(), nil)
	req = withActorContext(req, actorAccountID, staffID)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", staffID.String())
	rctx.URLParams.Add("accountID", targetAccountID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	err := handler.RemoveStaffAccount(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}
}

func TestHandler_CreateStaff(t *testing.T) {
	staffID := uuid.New()
	accountID := uuid.New()
	handler, _, _ := setupTestHandler(staffID, accountID)

	body, _ := json.Marshal(createStaffRequest{
		Name:     "Floral Logistics",
		Username: "floral-logistics",
	})
	req := httptest.NewRequest(http.MethodPost, "/staff", bytes.NewReader(body))
	req = withActorContext(req, accountID, staffID)

	rec := httptest.NewRecorder()
	err := handler.CreateStaff(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got: %d", rec.Code)
	}
}

func TestHandler_AddStaffAccount(t *testing.T) {
	staffID := uuid.New()
	accountID := uuid.New()
	handler, _, _ := setupTestHandler(staffID, accountID)

	body, _ := json.Marshal(addStaffAccountRequest{
		Email:    "new@example.com",
		Password: "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/staff/"+staffID.String()+"/accounts", bytes.NewReader(body))
	req = withActorContext(req, accountID, staffID)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("staffID", staffID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	err := handler.AddStaffAccount(rec, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got: %d", rec.Code)
	}
}
