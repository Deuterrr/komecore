package shopdomain

import (
	"time"

	"github.com/google/uuid"
)

type ShopApprovalStatus string

const (
	ShopApprovalStatusPending  ShopApprovalStatus = "pending"
	ShopApprovalStatusApproved ShopApprovalStatus = "approved"
	ShopApprovalStatusRejected ShopApprovalStatus = "rejected"
)

type Shop struct {
	ID uuid.UUID

	Name        string
	Slug        string
	Description *string

	IsActive       bool
	ApprovalStatus ShopApprovalStatus

	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (s *Shop) IsOperable() bool {
	return s.IsActive && s.ApprovalStatus == ShopApprovalStatusApproved
}
