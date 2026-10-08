package domain

import "komecore/internal/common/authctx"

type RoleCode = authctx.RoleCode

const (
	RoleStaffAdmin = authctx.RoleStaffAdmin
	RoleStaff      = authctx.RoleStaff
	RoleCustomer   = authctx.RoleCustomer
)

const (
	PermissionShopView          = authctx.PermissionShopView
	PermissionShopUpdate        = authctx.PermissionShopUpdate
	PermissionProductCreate     = authctx.PermissionProductCreate
	PermissionProductUpdate     = authctx.PermissionProductUpdate
	PermissionProductDelete     = authctx.PermissionProductDelete
	PermissionInventoryManage   = authctx.PermissionInventoryManage
	PermissionOrderRead         = authctx.PermissionOrderRead
	PermissionOrderUpdateStatus = authctx.PermissionOrderUpdateStatus
	PermissionCourierManage     = authctx.PermissionCourierManage
	PermissionAddressManage     = authctx.PermissionAddressManage
)

type Role = authctx.Role
type Actor = authctx.Actor
