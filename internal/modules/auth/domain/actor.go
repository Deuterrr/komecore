package domain

import "github.com/google/uuid"

type RoleCode string

const (
	RoleStaffAdmin RoleCode = "staff_admin"
	RoleStaff      RoleCode = "staff"
	RoleCustomer   RoleCode = "customer"
)

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

type Role struct {
	ID   uuid.UUID
	Code RoleCode
	Name string
}

type Actor struct {
	AccountID  uuid.UUID
	Type       AccountType
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []Role
}

func (a *Actor) HasRole(role RoleCode) bool {
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
