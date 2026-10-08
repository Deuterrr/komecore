package authctx

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrStaffRequired          = errors.New("staff account required")
	ErrInsufficientRole       = errors.New("insufficient role for operation")
)

// AccountType identifies the high-level category of account.
type AccountType string

const (
	AccountTypeCustomer AccountType = "customer"
	AccountTypeStaff    AccountType = "staff"
)

// TokenType indicates access vs refresh token.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// RoleCode represents staff/customer roles.
type RoleCode string

const (
	RoleStaffAdmin RoleCode = "staff_admin"
	RoleStaff      RoleCode = "staff"
	RoleCustomer   RoleCode = "customer"
)

// Standard permissions across domains.
const (
	PermissionShopView          = "shop:view"
	PermissionShopUpdate        = "shop:update"
	PermissionProductCreate     = "product:create"
	PermissionProductUpdate     = "product:update"
	PermissionProductDelete     = "product:delete"
	PermissionInventoryManage   = "inventory:manage"
	PermissionOrderRead         = "order:read"
	PermissionOrderUpdateStatus = "order:update_status"
	PermissionCourierManage     = "courier:manage"
	PermissionAddressManage     = "address:manage"
)

// Role represents an assigned role with ID, code and name.
type Role struct {
	ID   uuid.UUID
	Code RoleCode
	Name string
}

// Actor represents the active authenticated identity with its resolved permissions and roles.
type Actor struct {
	AccountID  uuid.UUID
	Type       AccountType
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []Role
}

func (a *Actor) HasRole(role RoleCode) bool {
	if a == nil {
		return false
	}
	for _, r := range a.Roles {
		if r.Code == role {
			return true
		}
	}
	return false
}

func (a *Actor) IsSuperAdmin() bool {
	return a.HasRole(RoleStaffAdmin)
}

func (a *Actor) HasPermission(shopID uuid.UUID, permission string) bool {
	if a == nil {
		return false
	}
	if a.IsSuperAdmin() {
		return true
	}
	if a.Type == AccountTypeStaff {
		return true
	}
	return false
}

func (a *Actor) GetAssignedShopIDs() []uuid.UUID {
	return nil
}

// AuthContext represents the authenticated identity during a request lifecycle.
type AuthContext struct {
	UserID    uuid.UUID
	SessionID uuid.UUID

	StaffID    *uuid.UUID
	CustomerID *uuid.UUID

	AccountType AccountType
	TokenType   TokenType

	IsAuthenticated bool
	Roles           []string
}

type (
	authContextKey struct{}
	actorContextKey struct{}
)

// WithAuthContext stores an AuthContext in ctx.
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, authCtx)
}

// GetAuthContext retrieves the AuthContext from ctx.
func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey{}).(*AuthContext)
	return authCtx, ok
}

// WithActor stores an Actor in ctx.
func WithActor(ctx context.Context, actor *Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// GetActor retrieves the Actor from ctx.
func GetActor(ctx context.Context) (*Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(*Actor)
	return actor, ok
}

// ActorFromAuthContext translates an AuthContext into a rich Actor struct.
func ActorFromAuthContext(authCtx *AuthContext) *Actor {
	if authCtx == nil {
		return nil
	}

	accType := authCtx.AccountType
	if accType == "" {
		if authCtx.StaffID != nil {
			accType = AccountTypeStaff
		} else {
			accType = AccountTypeCustomer
		}
	}

	var roles []Role
	for _, r := range authCtx.Roles {
		roles = append(roles, Role{
			Code: RoleCode(r),
			Name: r,
		})
	}

	return &Actor{
		AccountID:  authCtx.UserID,
		Type:       accType,
		StaffID:    authCtx.StaffID,
		CustomerID: authCtx.CustomerID,
		Roles:      roles,
	}
}
