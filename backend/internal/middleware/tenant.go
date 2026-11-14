package middleware

import (
	"log/slog"
	"strings"

	"fixphone/backend/internal/database"
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
//
// It also starts a request-scoped transaction on the application pool and sets
// app.tenant_id for PostgreSQL row level security. The transaction is committed
// for successful responses and rolled back for error responses or panics.
// resolver should be a privileged pool because resolving the tenant happens
// before a tenant context exists. In single-role deployments resolver and app
// may be the same handle.
func Tenant(resolver, app *gorm.DB, log *slog.Logger) gin.HandlerFunc {
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
		if err := resolver.WithContext(c.Request.Context()).
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
		if err := resolver.WithContext(c.Request.Context()).Raw(query, args...).Scan(&outlets).Error; err != nil {
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

		tx := app.WithContext(c.Request.Context()).Begin()
		if tx.Error != nil {
			response.Error(c, apperror.Internal(tx.Error))
			return
		}
		if err := tx.Exec(`SELECT set_config('app.tenant_id', ?, true)`, tenantID.String()).Error; err != nil {
			_ = tx.Rollback().Error
			response.Error(c, apperror.Internal(err))
			return
		}
		// Bound every query to a sane statement timeout so a stuck query cannot
		// hold a pooled connection forever.
		if err := tx.Exec(`SET LOCAL statement_timeout = '15s'`).Error; err != nil {
			_ = tx.Rollback().Error
			response.Error(c, apperror.Internal(err))
			return
		}
		ctx := database.ContextWithTx(c.Request.Context(), tx)
		c.Request = c.Request.WithContext(ctx)
		c.Set(scopeKey, tenantScope)

		finished := false
		defer func() {
			if finished {
				return
			}
			_ = tx.Rollback().Error
			if recovered := recover(); recovered != nil {
				panic(recovered)
			}
		}()

		c.Next()
		finished = true

		if c.Writer.Status() >= 400 {
			if err := tx.Rollback().Error; err != nil && log != nil {
				log.Warn("tenant_tx_rollback_failed", "tenant_id", tenantID, "error", err)
			}
			return
		}
		if err := tx.Commit().Error; err != nil {
			if log != nil {
				log.Error("tenant_tx_commit_failed", "tenant_id", tenantID, "error", err)
			}
			if !c.Writer.Written() {
				response.Error(c, apperror.Internal(err))
			}
		}
	}
}
