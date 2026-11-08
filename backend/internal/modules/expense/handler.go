package expense

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

// Handler exposes expense endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the expense handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListExpenses returns a paginated expense list.
func (h *Handler) ListExpenses(c *gin.Context) {
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
	categoryID, appErr := queryUUID(c.Query("category_id"), "category_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	result, err := h.service.ListExpenses(c.Request.Context(), actor, ExpenseFilter{
		Query: c.Query("q"), CategoryID: valueOrNil(categoryID),
		OutletID: valueOrNil(outletID), FromDate: c.Query("from"), ToDate: c.Query("to"),
		Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"items": result.Items, "total_amount": result.TotalAmount,
	}, response.Meta{
		Page: page, PerPage: perPage, Total: result.Total,
		TotalPages: totalPages(result.Total, perPage),
	})
}

// CreateExpense records a new expense.
func (h *Handler) CreateExpense(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateExpenseRequest
	if appErr := validator.Bind(c, &body, map[string]string{
		"CategoryID": "category_id", "Description": "description",
		"Amount": "amount", "ExpenseDate": "expense_date",
		"Method": "method", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	item, err := h.service.CreateExpense(c.Request.Context(), actor, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}

// UpdateExpense edits an existing expense.
func (h *Handler) UpdateExpense(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	expenseID, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	var body UpdateExpenseRequest
	if appErr := validator.Bind(c, &body, map[string]string{
		"CategoryID": "category_id", "Description": "description",
		"Amount": "amount", "ExpenseDate": "expense_date",
		"Method": "method", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	item, err := h.service.UpdateExpense(c.Request.Context(), actor, expenseID, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// DeleteExpense soft-deletes an expense.
func (h *Handler) DeleteExpense(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	expenseID, appErr := uuidParam(c, "id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	if err := h.service.DeleteExpense(c.Request.Context(), actor, expenseID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Pengeluaran berhasil dihapus."})
}

// ListCategories returns expense categories.
func (h *Handler) ListCategories(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.service.ListCategories(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// CreateCategory adds a custom expense category.
func (h *Handler) CreateCategory(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateCategoryRequest
	if appErr := validator.Bind(c, &body, map[string]string{"Name": "name"}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	category, err := h.service.CreateCategory(c.Request.Context(), actor, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, category)
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
