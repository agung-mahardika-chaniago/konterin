package invoice

import (
	"net/http"
	"strconv"
	"strings"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"
	"fixphone/backend/internal/pkg/validator"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes invoice endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the invoice handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListInvoices returns a paginated invoice list for the active outlet.
func (h *Handler) ListInvoices(c *gin.Context) {
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
	statuses := parseStatuses(c.Query("status"))
	result, err := h.service.ListInvoices(c.Request.Context(), actor, InvoiceFilter{
		Query: c.Query("q"), Statuses: statuses, OutletID: valueOrNil(outletID),
		FromDate: c.Query("from"), ToDate: c.Query("to"),
		Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, result.Items, response.Meta{
		Page: page, PerPage: perPage, Total: result.Total,
		TotalPages: totalPages(result.Total, perPage),
	})
}

// GetInvoice returns the full invoice payload.
func (h *Handler) GetInvoice(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	detail, err := h.service.GetInvoice(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, detail)
}

// GetInvoiceByServiceOrder resolves the invoice linked to a service order.
func (h *Handler) GetInvoiceByServiceOrder(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	orderID, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	detail, err := h.service.GetInvoiceByServiceOrder(c.Request.Context(), actor, orderID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateInvoice applies discount and notes while the invoice is a draft.
func (h *Handler) UpdateInvoice(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	var body UpdateInvoiceRequest
	if appErr := validator.Bind(c, &body, map[string]string{
		"DiscountType": "discount_type", "DiscountValue": "discount_value",
		"Notes": "notes", "DueAt": "due_at",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	detail, err := h.service.UpdateInvoice(c.Request.Context(), actor, id, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, detail)
}

// IssueInvoice finalises a draft invoice.
func (h *Handler) IssueInvoice(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	detail, err := h.service.IssueInvoice(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, detail)
}

// VoidInvoice cancels an invoice without posted payments.
func (h *Handler) VoidInvoice(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	var body VoidInvoiceRequest
	if appErr := validator.Bind(c, &body, map[string]string{"Reason": "reason"}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	detail, err := h.service.VoidInvoice(c.Request.Context(), actor, id, body.Reason)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, detail)
}

// PrintInvoice returns the print payload used by the A4/thermal print views.
func (h *Handler) PrintInvoice(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	detail, err := h.service.GetInvoice(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"template": "invoice",
		"paper":    "A4",
		"invoice":  detail,
	})
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
		UserID:          identity.UserID,
		TenantID:        tenantScope.TenantID,
		Name:            identity.Name,
		Role:            identity.Role,
		Permissions:     identity.Permissions,
		AllOutletAccess: tenantScope.AllOutletAccess,
		OutletIDs:       tenantScope.OutletIDs,
		ActiveOutletID:  tenantScope.ActiveOutletID,
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
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperror.Validation(map[string]string{field: "ID tidak valid."})
	}
	return &id, nil
}

func valueOrNil(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

func parseStatuses(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	statuses := make([]string, 0, len(parts))
	for _, part := range parts {
		status := strings.ToLower(strings.TrimSpace(part))
		if status != "" {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

func parseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
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
	if total <= 0 || perPage <= 0 {
		return 1
	}
	pages := int((total + int64(perPage) - 1) / int64(perPage))
	if pages < 1 {
		return 1
	}
	return pages
}
