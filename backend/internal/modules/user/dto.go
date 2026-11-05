package user

import "time"

// Actor is the authenticated principal performing a management action.
type Actor struct {
	UserID   string
	TenantID string
	Role     string
	Name     string
}

// OutletDTO is an outlet summary.
type OutletDTO struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// UserDTO is the API representation of a user.
type UserDTO struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Email              string      `json:"email"`
	Phone              *string     `json:"phone"`
	Role               string      `json:"role"`
	RoleName           string      `json:"role_name"`
	IsActive           bool        `json:"is_active"`
	MustChangePassword bool        `json:"must_change_password"`
	LastLoginAt        *time.Time  `json:"last_login_at"`
	Outlets            []OutletDTO `json:"outlets"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// ListResponse is a paginated user list.
type ListResponse struct {
	Items []UserDTO
	Total int64
}

// CreateRequest creates a tenant user.
type CreateRequest struct {
	Name      string   `json:"name" validate:"required,min=1,max=120"`
	Email     string   `json:"email" validate:"required,email,max=150"`
	Phone     *string  `json:"phone" validate:"omitempty,max=30"`
	Role      string   `json:"role" validate:"required,max=30"`
	Password  string   `json:"password" validate:"required,min=8,max=72"`
	OutletIDs []string `json:"outlet_ids" validate:"max=50,dive,uuid"`
	IsActive  *bool    `json:"is_active"`
}

// UpdateRequest edits a tenant user.
type UpdateRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=1,max=120"`
	Phone    *string `json:"phone" validate:"omitempty,max=30"`
	Role     *string `json:"role" validate:"omitempty,max=30"`
	IsActive *bool   `json:"is_active"`
}

// ReplaceOutletsRequest assigns a user's outlets.
type ReplaceOutletsRequest struct {
	OutletIDs []string `json:"outlet_ids" validate:"max=50,dive,uuid"`
}

// ResetPasswordResponse is returned exactly once.
type ResetPasswordResponse struct {
	TemporaryPassword  string `json:"temporary_password"`
	MustChangePassword bool   `json:"must_change_password"`
}

// RoleDTO is a role with its permission codes.
type RoleDTO struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// PermissionDTO is a permission definition.
type PermissionDTO struct {
	Code        string  `json:"code"`
	Module      string  `json:"module"`
	Description *string `json:"description"`
}

// ListFilter is the handler-level list query.
type ListFilter struct {
	Page    int
	PerPage int
	Query   string
	Role    string
	Status  string
}

// CreateInput is the service-level create command.
type CreateInput struct {
	Name      string
	Email     string
	Phone     *string
	Role      string
	Password  string
	OutletIDs []string
	IsActive  bool
}

// UpdateInput is the service-level update command.
type UpdateInput struct {
	Name     *string
	Phone    *string
	Role     *string
	IsActive *bool
}
