package service

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

// Handler exposes customer and service-order endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the service handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ListCustomers returns a paginated customer list.
func (h *Handler) ListCustomers(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	result, err := h.service.ListCustomers(c.Request.Context(), actor, CustomerFilter{
		Query: c.Query("q"), Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, result.Items, response.PageMeta(page, perPage, result.Total))
}

// CreateCustomer adds a customer.
func (h *Handler) CreateCustomer(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateCustomerRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Name": "name", "Phone": "phone", "Address": "address", "Notes": "notes",
	}); err != nil {
		response.Error(c, err)
		return
	}
	created, err := h.service.CreateCustomer(c.Request.Context(), actor, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// GetCustomer returns customer detail.
func (h *Handler) GetCustomer(c *gin.Context) {
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
	item, err := h.service.GetCustomer(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateCustomer edits a customer.
func (h *Handler) UpdateCustomer(c *gin.Context) {
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
	var body UpdateCustomerRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Name": "name", "Phone": "phone", "Address": "address", "Notes": "notes",
	}); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.service.UpdateCustomer(c.Request.Context(), actor, id, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
}

// DeleteCustomer soft-deletes a customer.
func (h *Handler) DeleteCustomer(c *gin.Context) {
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
	if err := h.service.DeleteCustomer(c.Request.Context(), actor, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Pelanggan berhasil dihapus."})
}

// ListCustomerDevices returns a customer's devices.
func (h *Handler) ListCustomerDevices(c *gin.Context) {
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
	items, err := h.service.ListCustomerDevices(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// CreateCustomerDevice adds a device to a customer.
func (h *Handler) CreateCustomerDevice(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	customerID, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateDeviceRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Brand": "brand", "Model": "model", "IMEI": "imei", "Color": "color", "Notes": "notes",
	}); err != nil {
		response.Error(c, err)
		return
	}
	created, err := h.service.CreateCustomerDevice(c.Request.Context(), actor, customerID, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// ListCustomerServices returns the customer's orders for the active outlet.
func (h *Handler) ListCustomerServices(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	customerID, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	result, err := h.service.ListCustomerServices(c.Request.Context(), actor, customerID, ServiceFilter{
		Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, result.Items, response.PageMeta(page, perPage, result.Total))
}

// CustomerStats returns aggregated customer history.
func (h *Handler) CustomerStats(c *gin.Context) {
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
	stats, err := h.service.GetCustomerStats(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}

// ListServiceOrders returns a paginated order list.
func (h *Handler) ListServiceOrders(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	filter := ServiceFilter{
		Query: c.Query("q"), Page: page, PerPage: perPage,
		FromDate: c.Query("from"), ToDate: c.Query("to"),
	}
	if raw := strings.TrimSpace(c.Query("outlet_id")); raw != "" {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			response.Error(c, apperror.Validation(map[string]string{"outlet_id": "ID outlet tidak valid."}))
			return
		}
		filter.OutletID = id
	}
	if raw := strings.TrimSpace(c.Query("technician_id")); raw != "" {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			response.Error(c, apperror.Validation(map[string]string{"technician_id": "ID teknisi tidak valid."}))
			return
		}
		filter.TechnicianID = &id
	}
	if raw := strings.TrimSpace(c.Query("customer_id")); raw != "" {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			response.Error(c, apperror.Validation(map[string]string{"customer_id": "ID pelanggan tidak valid."}))
			return
		}
		filter.CustomerID = &id
	}
	for _, raw := range append(c.QueryArray("status"), strings.Split(c.Query("status"), ",")...) {
		for _, status := range strings.Split(raw, ",") {
			status = strings.ToLower(strings.TrimSpace(status))
			if status != "" && status != "all" {
				filter.Statuses = append(filter.Statuses, status)
			}
		}
	}
	result, err := h.service.ListServiceOrders(c.Request.Context(), actor, filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, result.Items, response.PageMeta(page, perPage, result.Total))
}

// CreateServiceOrder performs intake.
func (h *Handler) CreateServiceOrder(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateServiceRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Complaint": "complaint", "EstimatedCost": "estimated_cost", "Priority": "priority",
		"Notes": "notes", "InternalNotes": "internal_notes",
	}); err != nil {
		response.Error(c, err)
		return
	}
	created, err := h.service.CreateServiceOrder(c.Request.Context(), actor, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// GetServiceOrder returns full detail.
func (h *Handler) GetServiceOrder(c *gin.Context) {
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
	item, err := h.service.GetServiceOrder(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateServiceOrder edits intake fields.
func (h *Handler) UpdateServiceOrder(c *gin.Context) {
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
	var body UpdateServiceRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Complaint": "complaint", "EstimatedCost": "estimated_cost", "Priority": "priority",
		"Notes": "notes", "InternalNotes": "internal_notes",
	}); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.service.UpdateServiceOrder(c.Request.Context(), actor, id, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
}

// Timeline returns the order status history.
func (h *Handler) Timeline(c *gin.Context) {
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
	entries, err := h.service.Timeline(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, entries)
}

// SetCredential stores a device credential.
func (h *Handler) SetCredential(c *gin.Context) {
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
	var body CredentialRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Type": "type", "Value": "value",
	}); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.service.SetCredential(c.Request.Context(), actor, id, body); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Kredensial perangkat tersimpan."})
}

// DeleteCredential removes a device credential.
func (h *Handler) DeleteCredential(c *gin.Context) {
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
	if err := h.service.DeleteCredential(c.Request.Context(), actor, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Kredensial perangkat dihapus."})
}

// RevealCredential decrypts a credential and records the reason.
func (h *Handler) RevealCredential(c *gin.Context) {
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
	var body RevealCredentialRequest
	if err := validator.Bind(c, &body, map[string]string{"Reason": "reason"}); err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.RevealCredential(c.Request.Context(), actor, id, body.Reason)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, result)
}

// UploadPhoto accepts a multipart service photo.
func (h *Handler) UploadPhoto(c *gin.Context) {
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
	maxBytes := h.service.opts.UploadMaxBytes
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+(1<<20))
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperror.Validation(map[string]string{"file": "File foto wajib diunggah."}))
		return
	}
	isPublic := c.PostForm("is_public") == "true" || c.PostForm("is_public") == "1"
	photo, err := h.service.AddPhoto(c.Request.Context(), actor, id, file, c.PostForm("stage"), isPublic)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, photo)
}

// DeletePhoto removes a service photo.
func (h *Handler) DeletePhoto(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	orderID, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	photoID, err := uuidParam(c, "photoId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.service.DeletePhoto(c.Request.Context(), actor, orderID, photoID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Foto berhasil dihapus."})
}

// Technicians returns active technicians for filters and assignment.
func (h *Handler) Technicians(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var outletID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("outlet_id")); raw != "" {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			response.Error(c, apperror.Validation(map[string]string{"outlet_id": "ID outlet tidak valid."}))
			return
		}
		outletID = &id
	}
	items, err := h.service.Technicians(c.Request.Context(), actor, outletID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
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
