package middleware

import (
	"strings"

	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"
	"fixphone/backend/internal/pkg/scope"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type outletRef struct {
	ID       uuid.UUID `gorm:"column:id"`
	IsActive bool      `gorm:"column:is_active"`
}

// Tenant resolves the tenant boundary and the active outlet for the request.
// Owners and admins automatically have access to every outlet in their tenant;
// other roles only see outlets explicitly assigned to them.
func Tenant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity, ok := GetAuth(c)
		if !ok {
			response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan. Silakan login."))
			return
		}

		if identity.TenantID == nil {
			// Platform super admins have no tenant boundary.
			c.Set(scopeKey, &scope.Tenant{IsSuperAdmin: true, Role: identity.Role})
			c.Next()
			return
		}

		tenantID := *identity.TenantID
		var status string
		if err := db.WithContext(c.Request.Context()).
			Raw(`SELECT status FROM tenants WHERE id = ?`, tenantID).Scan(&status).Error; err != nil {
			response.Error(c, apperror.Internal(err))
			return
		}
		switch strings.TrimSpace(status) {
		case "":
			response.Error(c, apperror.Unauthorized("TENANT_NOT_FOUND", "Tenant tidak ditemukan."))
			return
		case "suspended":
			response.Error(c, apperror.Forbidden("TENANT_SUSPENDED", "Akun toko sedang dinonaktifkan."))
			return
		}

		allAccess := identity.Role == "owner" || identity.Role == "admin"
		outlets := make([]outletRef, 0)
		query := `SELECT id, is_active FROM outlets WHERE tenant_id = ?`
		args := []any{tenantID}
		if !allAccess {
			query += ` AND id IN (SELECT outlet_id FROM user_outlets WHERE user_id = ?)`
			args = append(args, identity.UserID)
		}
		query += ` ORDER BY name`
		if err := db.WithContext(c.Request.Context()).Raw(query, args...).Scan(&outlets).Error; err != nil {
			response.Error(c, apperror.Internal(err))
			return
		}

		outletIDs := make([]uuid.UUID, 0, len(outlets))
		activeSet := false
		for _, outlet := range outlets {
			if outlet.IsActive {
				outletIDs = append(outletIDs, outlet.ID)
			}
		}

		tenantScope := &scope.Tenant{
			TenantID:        tenantID,
			OutletIDs:       outletIDs,
			Role:            identity.Role,
			AllOutletAccess: allAccess,
		}

		if rawHeader := strings.TrimSpace(c.GetHeader("X-Outlet-Id")); rawHeader != "" {
			outletID, err := uuid.Parse(rawHeader)
			if err != nil || !tenantScope.HasOutlet(outletID) {
				response.Error(c, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini."))
				return
			}
			tenantScope.ActiveOutletID = &outletID
			activeSet = true
		}
		if !activeSet && len(outletIDs) > 0 {
			first := outletIDs[0]
			tenantScope.ActiveOutletID = &first
		}

		c.Set(scopeKey, tenantScope)
		c.Next()
	}
}
