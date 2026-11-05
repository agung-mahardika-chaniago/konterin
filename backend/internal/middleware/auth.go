// Package middleware contains cross-cutting HTTP guards for the API.
package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/jwtutil"
	"fixphone/backend/internal/pkg/response"
	"fixphone/backend/internal/pkg/scope"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	authContextKey = "auth_context"
	scopeKey       = "tenant_scope"
)

// AuthContext is the authenticated identity attached to a request.
type AuthContext struct {
	UserID             uuid.UUID
	TenantID           *uuid.UUID
	Name               string
	Email              string
	Role               string
	SessionID          uuid.UUID
	Permissions        map[string]struct{}
	MustChangePassword bool
	PermissionVersion  int
}

// HasPermission reports whether the identity holds a permission code.
func (a *AuthContext) HasPermission(code string) bool {
	if a == nil {
		return false
	}
	_, ok := a.Permissions[code]
	return ok
}

// IdentityLoader loads a fresh identity from storage. It must reject tokens
// whose permission version is stale and accounts that are inactive.
type IdentityLoader interface {
	LoadAuthContext(ctx context.Context, userID uuid.UUID, permissionVersion int) (*AuthContext, error)
}

// Auth verifies the bearer token and attaches the identity.
func Auth(tokens *jwtutil.Manager, loader IdentityLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan. Silakan login."))
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Format token tidak valid."))
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			if errors.Is(err, jwtutil.ErrExpired) {
				response.Error(c, apperror.Unauthorized("TOKEN_EXPIRED", "Sesi telah berakhir. Silakan muat ulang."))
				return
			}
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Token tidak valid."))
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Token tidak valid."))
			return
		}
		sessionID, err := uuid.Parse(claims.SessionID)
		if err != nil {
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Token tidak valid."))
			return
		}

		identity, err := loader.LoadAuthContext(c.Request.Context(), userID, claims.PermissionVersion)
		if err != nil {
			response.Error(c, err)
			return
		}
		identity.SessionID = sessionID
		c.Set(authContextKey, identity)
		c.Next()
	}
}

// GetAuth returns the authenticated identity, if any.
func GetAuth(c *gin.Context) (*AuthContext, bool) {
	value, ok := c.Get(authContextKey)
	if !ok {
		return nil, false
	}
	identity, ok := value.(*AuthContext)
	return identity, ok && identity != nil
}

// GetScope returns the tenant/outlet boundary, if any.
func GetScope(c *gin.Context) (*scope.Tenant, bool) {
	value, ok := c.Get(scopeKey)
	if !ok {
		return nil, false
	}
	tenantScope, ok := value.(*scope.Tenant)
	return tenantScope, ok && tenantScope != nil
}

// RequirePermission enforces default-deny authorization. Accounts that must
// change their password are blocked from all permission-guarded endpoints.
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity, ok := GetAuth(c)
		if !ok {
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan. Silakan login."))
			return
		}
		if identity.MustChangePassword {
			response.Error(c, apperror.Forbidden("PASSWORD_CHANGE_REQUIRED", "Ubah password terlebih dahulu sebelum melanjutkan."))
			return
		}
		if !identity.HasPermission(code) {
			response.Error(c, apperror.Forbidden("FORBIDDEN", "Anda tidak memiliki akses untuk tindakan ini."))
			return
		}
		c.Next()
	}
}

// RateLimit is a per-client, per-route sliding-window limiter. It is intended
// for auth endpoints where a shared external store is not yet available.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	buckets := newLimiterStore(limit, window)
	return func(c *gin.Context) {
		allowed, retryAfter := buckets.allow(c.ClientIP() + "|" + c.FullPath())
		if !allowed {
			c.Header("Retry-After", retryAfter)
			response.Error(c, apperror.RateLimited("Terlalu banyak percobaan. Silakan coba lagi nanti."))
			return
		}
		c.Next()
	}
}
