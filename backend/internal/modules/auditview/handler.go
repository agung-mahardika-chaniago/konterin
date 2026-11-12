package auditview

import (
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes the read-only audit log endpoint.
type Handler struct {
	service *Service
}

// NewHandler builds the audit viewer handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// List returns the paginated audit trail with filters.
func (h *Handler) List(c *gin.Context) {
	scope, ok := middleware.GetScope(c)
	if !ok || scope.IsSuperAdmin || scope.TenantID == uuid.Nil {
		response.Error(c, apperror.Forbidden("FORBIDDEN", "Audit log hanya tersedia untuk pengguna tenant."))
		return
	}

	filter := Filter{TenantID: scope.TenantID, Page: 1, PerPage: 20}
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		if value, err := parseInt(raw); err == nil && value > 0 {
			filter.Page = value
		}
	}
	if raw := strings.TrimSpace(c.Query("per_page")); raw != "" {
		if value, err := parseInt(raw); err == nil && value > 0 {
			filter.PerPage = value
			if filter.PerPage > 100 {
				filter.PerPage = 100
			}
		}
	}
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			filter.UserID = &id
		} else {
			response.Error(c, apperror.Validation(map[string]string{"user_id": "ID pengguna tidak valid."}))
			return
		}
	}
	if raw := strings.TrimSpace(c.Query("module")); raw != "" {
		filter.Module = raw
	}
	if raw := strings.TrimSpace(c.Query("action")); raw != "" {
		filter.Action = raw
	}
	if raw := strings.TrimSpace(c.Query("entity_type")); raw != "" {
		filter.EntityType = raw
	}
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		day, err := time.Parse("2006-01-02", raw)
		if err != nil {
			response.Error(c, apperror.Validation(map[string]string{"from": "Format tanggal harus YYYY-MM-DD."}))
			return
		}
		filter.From = &day
	}
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		day, err := time.Parse("2006-01-02", raw)
		if err != nil {
			response.Error(c, apperror.Validation(map[string]string{"to": "Format tanggal harus YYYY-MM-DD."}))
			return
		}
		filter.To = &day
	}

	rows, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, 200, gin.H{"items": rows}, response.PageMeta(filter.Page, filter.PerPage, total))
}

// Modules returns the distinct audit modules for the filter dropdown.
func (h *Handler) Modules(c *gin.Context) {
	scope, ok := middleware.GetScope(c)
	if !ok || scope.IsSuperAdmin || scope.TenantID == uuid.Nil {
		response.Error(c, apperror.Forbidden("FORBIDDEN", "Audit log hanya tersedia untuk pengguna tenant."))
		return
	}
	modules, err := h.service.Modules(c.Request.Context(), scope.TenantID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, modules)
}

func parseInt(raw string) (int, error) {
	var value int
	_, err := fmt.Sscan(raw, &value)
	return value, err
}
