package warranty

import (
	"net/http"
	"strconv"
	"strings"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes warranty and claim endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the warranty handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListWarranties returns tenant-scoped warranties.
func (h *Handler) ListWarranties(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	outletID, appErr := queryUUID(c.Query("outlet_id"), "outlet_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	items, total, err := h.service.ListWarranties(c.Request.Context(), actor, WarrantyFilter{
		Status: strings.TrimSpace(c.Query("status")), Query: c.Query("q"),
		OutletID: outletID, ExpiringInDays: parseInt(c.Query("expiring_in_days"), 0),
		Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, items, response.Meta{
		Page: page, PerPage: perPage, Total: total,
		TotalPages: totalPages(total, perPage),
	})
}

// GetWarranty returns one warranty with claims and invoice.
func (h *Handler) GetWarranty(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	item, svcErr := h.service.GetWarranty(c.Request.Context(), actor, id)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, item)
}

// CreateClaim records a claim against a warranty.
func (h *Handler) CreateClaim(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var input CreateClaimRequest
	if bindErr := c.ShouldBindJSON(&input); bindErr != nil {
		response.Error(c, apperror.Validation(map[string]string{"body": "Payload tidak valid."}))
		return
	}
	claim, svcErr := h.service.CreateClaim(c.Request.Context(), actor, id, input)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Created(c, claim)
}

// ListClaims returns tenant-scoped claims.
func (h *Handler) ListClaims(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	outletID, appErr := queryUUID(c.Query("outlet_id"), "outlet_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	items, total, err := h.service.ListClaims(c.Request.Context(), actor, ClaimFilter{
		Status: strings.TrimSpace(c.Query("status")), Query: c.Query("q"),
		OutletID: outletID, Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, items, response.Meta{
		Page: page, PerPage: perPage, Total: total,
		TotalPages: totalPages(total, perPage),
	})
}

// GetClaim returns one claim.
func (h *Handler) GetClaim(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	item, svcErr := h.service.GetClaim(c.Request.Context(), actor, id)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, item)
}

// UpdateClaim changes claim status and/or resolution.
func (h *Handler) UpdateClaim(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var input UpdateClaimRequest
	if bindErr := c.ShouldBindJSON(&input); bindErr != nil {
		response.Error(c, apperror.Validation(map[string]string{"body": "Payload tidak valid."}))
		return
	}
	item, svcErr := h.service.UpdateClaim(c.Request.Context(), actor, id, input)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, item)
}

// CreateRework creates a linked rework service order for a claim.
func (h *Handler) CreateRework(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	result, svcErr := h.service.CreateRework(c.Request.Context(), actor, id)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Created(c, result)
}

func actorFrom(c *gin.Context) (Actor, error) {
	identity, ok := middleware.GetAuth(c)
	if !ok {
		return Actor{}, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan.")
	}
	tenantScope, ok := middleware.GetScope(c)
	if !ok || tenantScope.IsSuperAdmin || tenantScope.TenantID == uuid.Nil {
		return Actor{}, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return Actor{
		UserID: identity.UserID, TenantID: tenantScope.TenantID,
		Name: identity.Name, Role: identity.Role, Permissions: identity.Permissions,
		AllOutletAccess: tenantScope.AllOutletAccess, OutletIDs: tenantScope.OutletIDs,
		ActiveOutletID: tenantScope.ActiveOutletID,
	}, nil
}

func uuidParam(c *gin.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param(name)))
	if err != nil {
		return uuid.Nil, apperror.BadRequest("ID tidak valid.")
	}
	return id, nil
}

func queryUUID(raw, field string) (*uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, apperror.Validation(map[string]string{field: "Format ID tidak valid."})
	}
	return &id, nil
}

func parseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func normalizePerPage(value int) int {
	if value < 1 {
		return 20
	}
	if value > 100 {
		return 100
	}
	return value
}

func totalPages(total int64, perPage int) int {
	if perPage < 1 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}
