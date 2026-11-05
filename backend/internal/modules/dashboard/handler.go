package dashboard

import (
	"strings"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes dashboard endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the dashboard handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Summary returns today's counters.
func (h *Handler) Summary(c *gin.Context) {
	filter, err := requestFilter(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Summary(c.Request.Context(), filter, c.Query("date"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Charts returns the daily revenue/service series.
func (h *Handler) Charts(c *gin.Context) {
	filter, err := requestFilter(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Charts(c.Request.Context(), filter, c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Alerts returns overdue orders, uncollected devices, and low stock.
func (h *Handler) Alerts(c *gin.Context) {
	filter, err := requestFilter(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Alerts(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Technicians returns per-technician activity.
func (h *Handler) Technicians(c *gin.Context) {
	filter, err := requestFilter(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Technicians(c.Request.Context(), filter, c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func requestFilter(c *gin.Context) (Filter, error) {
	tenantScope, ok := middleware.GetScope(c)
	if !ok || tenantScope.IsSuperAdmin || tenantScope.TenantID == uuid.Nil {
		return Filter{}, apperror.Forbidden("FORBIDDEN", "Dashboard hanya tersedia untuk pengguna tenant.")
	}

	var requested *uuid.UUID
	if raw := strings.TrimSpace(c.Query("outlet_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return Filter{}, apperror.Validation(map[string]string{"outlet_id": "ID outlet tidak valid."})
		}
		if !tenantScope.HasOutlet(id) {
			return Filter{}, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		requested = &id
	}
	if requested == nil {
		requested = tenantScope.ActiveOutletID
	}

	return Filter{TenantID: tenantScope.TenantID, OutletID: requested}, nil
}
