package inventory

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

// Handler exposes sparepart and inventory endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the inventory handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListSpareparts returns a paginated sparepart list.
func (h *Handler) ListSpareparts(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	categoryID, appErr := queryUUID(c.Query("category_id"), "category_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	supplierID, appErr := queryUUID(c.Query("supplier_id"), "supplier_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	result, err := h.service.ListSpareparts(c.Request.Context(), actor, SparepartFilter{
		Query: c.Query("q"), CategoryID: categoryID, SupplierID: supplierID,
		LowStock: parseBool(c.Query("low_stock")), Active: parseOptionalBool(c.Query("is_active")),
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

// CreateSparepart adds a master part.
func (h *Handler) CreateSparepart(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var input CreateSparepartRequest
	if appErr := validator.Bind(c, &input, map[string]string{
		"SKU": "sku", "Barcode": "barcode", "Name": "name",
		"CategoryID": "category_id", "SupplierID": "supplier_id",
		"CompatibleModels": "compatible_models", "Stock": "stock",
		"MinStock": "min_stock", "CostPrice": "cost_price",
		"SellPrice": "sell_price", "Location": "location", "IsActive": "is_active",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	created, err := h.service.CreateSparepart(c.Request.Context(), actor, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// GetSparepart returns one part.
func (h *Handler) GetSparepart(c *gin.Context) {
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
	item, err := h.service.GetSparepart(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// GetSparepartByBarcode resolves a scanner code.
func (h *Handler) GetSparepartByBarcode(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.service.GetSparepartByBarcode(c.Request.Context(), actor, c.Param("code"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateSparepart edits master data.
func (h *Handler) UpdateSparepart(c *gin.Context) {
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
	var input UpdateSparepartRequest
	if appErr := validator.Bind(c, &input, map[string]string{
		"SKU": "sku", "Barcode": "barcode", "Name": "name",
		"CategoryID": "category_id", "SupplierID": "supplier_id",
		"CompatibleModels": "compatible_models", "MinStock": "min_stock",
		"CostPrice": "cost_price", "SellPrice": "sell_price",
		"Location": "location", "IsActive": "is_active",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	updated, err := h.service.UpdateSparepart(c.Request.Context(), actor, id, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
}

// DeleteSparepart soft-deletes a part.
func (h *Handler) DeleteSparepart(c *gin.Context) {
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
	if err := h.service.DeleteSparepart(c.Request.Context(), actor, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Sparepart berhasil dihapus."})
}

// ListMovements returns stock movement history.
func (h *Handler) ListMovements(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	sparepartID, appErr := queryUUID(c.Query("sparepart_id"), "sparepart_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	if raw := c.Param("id"); raw != "" {
		id, paramErr := uuidParam(c, "id")
		if paramErr != nil {
			response.Error(c, paramErr)
			return
		}
		sparepartID = &id
	}
	outletID, appErr := queryUUID(c.Query("outlet_id"), "outlet_id")
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	result, err := h.service.ListMovements(c.Request.Context(), actor, MovementFilter{
		SparepartID: sparepartID, MovementType: c.Query("movement_type"),
		OutletID: outletID, FromDate: c.Query("from"), ToDate: c.Query("to"),
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

// Purchase records received stock.
func (h *Handler) Purchase(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var input PurchaseRequest
	if appErr := validator.Bind(c, &input, map[string]string{
		"SupplierID": "supplier_id", "ReferenceNo": "reference_no",
		"PurchasedAt": "purchased_at", "Items": "items", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	result, err := h.service.Purchase(c.Request.Context(), actor, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// AdjustStock sets physical stock for one part.
func (h *Handler) AdjustStock(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var input AdjustmentRequest
	if appErr := validator.Bind(c, &input, map[string]string{
		"SparepartID": "sparepart_id", "NewStock": "new_stock",
		"Reason": "reason", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	movement, err := h.service.AdjustStock(c.Request.Context(), actor, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, movement)
}

// Alerts lists low-stock parts.
func (h *Handler) Alerts(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	alerts, err := h.service.Alerts(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, alerts)
}

// Valuation returns inventory value at cost.
func (h *Handler) Valuation(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	valuation, err := h.service.Valuation(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, valuation)
}

// TopUsed ranks parts by service usage.
func (h *Handler) TopUsed(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.service.TopUsed(c.Request.Context(), actor,
		c.Query("from"), c.Query("to"), parseInt(c.Query("limit"), 10))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// ListCategories returns sparepart categories.
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

// CreateCategory adds a category.
func (h *Handler) CreateCategory(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var input CreateCategoryRequest
	if appErr := validator.Bind(c, &input, map[string]string{"Name": "name"}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	created, err := h.service.CreateCategory(c.Request.Context(), actor, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// UpdateCategory renames a category.
func (h *Handler) UpdateCategory(c *gin.Context) {
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
	var input UpdateCategoryRequest
	if appErr := validator.Bind(c, &input, map[string]string{"Name": "name"}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	updated, err := h.service.UpdateCategory(c.Request.Context(), actor, id, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
}

// ListSuppliers returns suppliers.
func (h *Handler) ListSuppliers(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.service.ListSuppliers(c.Request.Context(), actor, c.Query("q"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// CreateSupplier adds a supplier.
func (h *Handler) CreateSupplier(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var input CreateSupplierRequest
	if appErr := validator.Bind(c, &input, map[string]string{
		"Name": "name", "ContactName": "contact_name", "Phone": "phone",
		"Address": "address", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	created, err := h.service.CreateSupplier(c.Request.Context(), actor, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// UpdateSupplier edits a supplier.
func (h *Handler) UpdateSupplier(c *gin.Context) {
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
	var input UpdateSupplierRequest
	if appErr := validator.Bind(c, &input, map[string]string{
		"Name": "name", "ContactName": "contact_name", "Phone": "phone",
		"Address": "address", "Notes": "notes",
	}); appErr != nil {
		response.Error(c, appErr)
		return
	}
	updated, err := h.service.UpdateSupplier(c.Request.Context(), actor, id, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
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

func parseBool(raw string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && value
}

func parseOptionalBool(raw string) *bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
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
