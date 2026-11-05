package user

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

// Handler exposes user, role, and permission endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the user handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// List returns a paginated tenant user list.
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := parseInt(c.Query("per_page"), 20)
	result, err := h.service.List(c.Request.Context(), actor, ListFilter{
		Page: page, PerPage: perPage, Query: c.Query("q"),
		Role: c.Query("role"), Status: c.Query("status"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.JSON(c, http.StatusOK, result.Items, response.PageMeta(page, normalizePerPage(perPage), result.Total))
}

// Create adds a tenant user.
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Name": "name", "Email": "email", "Phone": "phone", "Role": "role",
		"Password": "password", "OutletIDs": "outlet_ids", "IsActive": "is_active",
	}); err != nil {
		response.Error(c, err)
		return
	}
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	created, err := h.service.Create(c.Request.Context(), actor, CreateInput{
		Name: body.Name, Email: body.Email, Phone: body.Phone, Role: body.Role,
		Password: body.Password, OutletIDs: body.OutletIDs, IsActive: isActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// Get returns one tenant user.
func (h *Handler) Get(c *gin.Context) {
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
	item, err := h.service.Get(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// Update edits a tenant user.
func (h *Handler) Update(c *gin.Context) {
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
	var body UpdateRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Name": "name", "Phone": "phone", "Role": "role", "IsActive": "is_active",
	}); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.service.Update(c.Request.Context(), actor, id, UpdateInput{
		Name: body.Name, Phone: body.Phone, Role: body.Role, IsActive: body.IsActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
}

// Delete soft-deletes a tenant user.
func (h *Handler) Delete(c *gin.Context) {
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
	if err := h.service.Delete(c.Request.Context(), actor, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Pengguna berhasil dihapus."})
}

// ResetPassword resets a user's password and returns the temporary password.
func (h *Handler) ResetPassword(c *gin.Context) {
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
	result, err := h.service.ResetPassword(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// ReplaceOutlets assigns a user's outlets.
func (h *Handler) ReplaceOutlets(c *gin.Context) {
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
	var body ReplaceOutletsRequest
	if err := validator.Bind(c, &body, map[string]string{"OutletIDs": "outlet_ids"}); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.service.ReplaceOutlets(c.Request.Context(), actor, id, body.OutletIDs)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, updated)
}

// Roles returns tenant roles and their permissions.
func (h *Handler) Roles(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	roles, err := h.service.Roles(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, roles)
}

// Permissions returns tenant permission definitions.
func (h *Handler) Permissions(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	permissions, err := h.service.Permissions(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, permissions)
}

func actorFrom(c *gin.Context) (Actor, error) {
	identity, ok := middleware.GetAuth(c)
	if !ok {
		return Actor{}, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan.")
	}
	tenantID := ""
	if identity.TenantID != nil {
		tenantID = identity.TenantID.String()
	}
	return Actor{
		UserID: identity.UserID.String(), TenantID: tenantID,
		Role: identity.Role, Name: identity.Name,
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
