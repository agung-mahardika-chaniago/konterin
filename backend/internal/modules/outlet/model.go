// Package outlet implements tenant outlet management.
package outlet

import (
	"time"

	"github.com/google/uuid"
)

// Outlet is a tenant branch.
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

// Actor is the authenticated principal.
type Actor struct {
	UserID   string
	TenantID string
	Role     string
	Name     string
}

// DTO is the API representation of an outlet.
type DTO struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   *string   `json:"address"`
	Phone     *string   `json:"phone"`
	Timezone  string    `json:"timezone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRequest creates an outlet.
type CreateRequest struct {
	Code     string  `json:"code" validate:"required,min=2,max=20"`
	Name     string  `json:"name" validate:"required,min=1,max=120"`
	Address  *string `json:"address" validate:"omitempty,max=500"`
	Phone    *string `json:"phone" validate:"omitempty,max=30"`
	Timezone string  `json:"timezone" validate:"omitempty,max=50"`
	IsActive *bool   `json:"is_active"`
}

// UpdateRequest edits an outlet.
type UpdateRequest struct {
	Code     *string `json:"code" validate:"omitempty,min=2,max=20"`
	Name     *string `json:"name" validate:"omitempty,min=1,max=120"`
	Address  *string `json:"address" validate:"omitempty,max=500"`
	Phone    *string `json:"phone" validate:"omitempty,max=30"`
	Timezone *string `json:"timezone" validate:"omitempty,max=50"`
	IsActive *bool   `json:"is_active"`
}

// CreateInput is the service-level create command.
type CreateInput struct {
	Code     string
	Name     string
	Address  *string
	Phone    *string
	Timezone string
	IsActive bool
}

// UpdateInput is the service-level update command.
type UpdateInput struct {
	Code     *string
	Name     *string
	Address  *string
	Phone    *string
	Timezone *string
	IsActive *bool
}
