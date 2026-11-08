package payment

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

// Handler exposes payment endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the payment handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListPayments returns a paginated payment list.
func (h *Handler) ListPayments(c *gin.Context) {
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
	result, err := h.service.ListPayments(c.Request.Context(), actor, PaymentFilter{
		Query: c.Query("q"), Statuses: parseStatuses(c.Query("status")),
		Method: c.Query("method"), OutletID: valueOrNil(outletID),
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

// ListInvoicePayments returns payments for one invoice.
func (h *Handler) ListInvoicePayments(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	invoiceID, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	items, err := h.service.ListInvoicePayments(c.Request.Context(), actor, invoiceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// CreatePayment posts a cashier payment.
func (h *Handler) CreatePayment(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	invoiceID, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	var body CreatePaymentRequest
	if appErr := validator.Bind(c, &body, map[string]string{
		"Method": "method", "Amount": "amount", "PaidAt": "paid_at",
		"ReferenceNo": "reference_no", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	result, err := h.service.CreatePayment(c.Request.Context(), actor, invoiceID,
		c.GetHeader("Idempotency-Key"), body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// VoidPayment cancels a posted payment.
func (h *Handler) VoidPayment(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	paymentID, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	var body VoidPaymentRequest
	if appErr := validator.Bind(c, &body, map[string]string{"Reason": "reason"}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	result, err := h.service.VoidPayment(c.Request.Context(), actor, paymentID, body.Reason)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// DailySummary returns the cashier's daily totals.
func (h *Handler) DailySummary(c *gin.Context) {
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
	summary, err := h.service.DailySummary(c.Request.Context(), actor, c.Query("date"), outletID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, summary)
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
