package auth

import (
	"time"

	"fixphone/backend/internal/modules/user"
)

// LoginRequest is the public login body.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=150"`
	Password string `json:"password" validate:"required,min=1,max=72"`
}

// UpdateProfileRequest edits the authenticated user's own profile.
type UpdateProfileRequest struct {
	Name       string  `json:"name" validate:"required,min=1,max=120"`
	Phone      *string `json:"phone" validate:"omitempty,max=30"`
	AvatarPath *string `json:"avatar_path" validate:"omitempty,max=255"`
}

// ChangePasswordRequest changes the authenticated user's password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,max=72"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
}

// Outlet is the outlet shape returned in the session payload.
type Outlet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// Profile is the authenticated user's session profile.
type Profile struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	Phone              *string    `json:"phone,omitempty"`
	AvatarPath         *string    `json:"avatar_path,omitempty"`
	Role               string     `json:"role"`
	TenantID           *string    `json:"tenant_id"`
	Permissions        []string   `json:"permissions"`
	Outlets            []Outlet   `json:"outlets"`
	ActiveOutletID     *string    `json:"active_outlet_id"`
	MustChangePassword bool       `json:"must_change_password"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
}

// Session is the result of login or refresh. The refresh token itself is only
// ever written to the HttpOnly cookie.
type Session struct {
	AccessToken         string
	ExpiresIn           int
	RefreshToken        string
	RefreshTokenExpires time.Time
	User                *Profile
}

// Meta carries request metadata for auditing and token records.
type Meta struct {
	IP        string
	UserAgent string
	RequestID string
}

// LoginInput is the service-level login command.
type LoginInput struct {
	Email    string
	Password string
}

// ProfileInput is the service-level profile update command.
type ProfileInput struct {
	Name       string
	Phone      *string
	AvatarPath *string
}

// ChangePasswordInput is the service-level password change command.
type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

// ProfileResponse wraps a profile for JSON responses.
type ProfileResponse struct {
	User *Profile `json:"user"`
}

// userOutlets converts repository outlets to the API shape.
func userOutlets(outlets []user.Outlet) []Outlet {
	result := make([]Outlet, 0, len(outlets))
	for _, outlet := range outlets {
		result = append(result, Outlet{ID: outlet.ID.String(), Name: outlet.Name, Code: outlet.Code})
	}
	return result
}
