// Package user implements tenant-scoped user management.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User is the user projection used by the API. PasswordHash is never exposed.
type User struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	TenantID           *uuid.UUID `gorm:"column:tenant_id"`
	RoleID             uuid.UUID  `gorm:"column:role_id"`
	RoleSlug           string     `gorm:"column:role_slug"`
	RoleName           string     `gorm:"column:role_name"`
	Name               string     `gorm:"column:name"`
	Email              string     `gorm:"column:email"`
	Phone              *string    `gorm:"column:phone"`
	PasswordHash       string     `gorm:"column:password_hash"`
	AvatarPath         *string    `gorm:"column:avatar_path"`
	IsActive           bool       `gorm:"column:is_active"`
	FailedLoginCount   int        `gorm:"column:failed_login_count"`
	LockedUntil        *time.Time `gorm:"column:locked_until"`
	LastLoginAt        *time.Time `gorm:"column:last_login_at"`
	PermissionVersion  int        `gorm:"column:permission_version"`
	MustChangePassword bool       `gorm:"column:must_change_password"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

// TableName maps the projection to the users table.
func (User) TableName() string { return "users" }

// Role is a system or custom role.
type Role struct {
	ID       uuid.UUID `gorm:"column:id"`
	Slug     string    `gorm:"column:slug"`
	Name     string    `gorm:"column:name"`
	IsSystem bool      `gorm:"column:is_system"`
}

// TableName maps the projection to the roles table.
func (Role) TableName() string { return "roles" }

// Permission is an RBAC permission definition.
type Permission struct {
	ID          uuid.UUID `gorm:"column:id"`
	Code        string    `gorm:"column:code"`
	Module      string    `gorm:"column:module"`
	Description *string   `gorm:"column:description"`
}

// TableName maps the projection to the permissions table.
func (Permission) TableName() string { return "permissions" }

// Outlet is a branch of a tenant.
type Outlet struct {
	ID        uuid.UUID `gorm:"column:id"`
	TenantID  uuid.UUID `gorm:"column:tenant_id"`
	Code      string    `gorm:"column:code"`
	Name      string    `gorm:"column:name"`
	Address   *string   `gorm:"column:address"`
	Phone     *string   `gorm:"column:phone"`
	Timezone  string    `gorm:"column:timezone"`
	IsActive  bool      `gorm:"column:is_active"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName maps the projection to the outlets table.
func (Outlet) TableName() string { return "outlets" }

// RefreshToken is an opaque refresh-token session record.
type RefreshToken struct {
	ID           uuid.UUID  `gorm:"column:id"`
	UserID       uuid.UUID  `gorm:"column:user_id"`
	TokenHash    string     `gorm:"column:token_hash"`
	FamilyID     uuid.UUID  `gorm:"column:family_id"`
	ExpiresAt    time.Time  `gorm:"column:expires_at"`
	RevokedAt    *time.Time `gorm:"column:revoked_at"`
	ReplacedByID *uuid.UUID `gorm:"column:replaced_by_id"`
	IP           *string    `gorm:"column:ip"`
	UserAgent    *string    `gorm:"column:user_agent"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

// TableName maps the projection to the refresh_tokens table.
func (RefreshToken) TableName() string { return "refresh_tokens" }
