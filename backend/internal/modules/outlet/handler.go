package outlet

import (
	"strings"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"
	"fixphone/backend/internal/pkg/validator"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes outlet endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the outlet handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// List returns accessible outlets.
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.service.List(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// Create adds an outlet.
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body CreateRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Code": "code", "Name": "name", "Address": "address", "Phone": "phone",
		"Timezone": "timezone", "IsActive": "is_active",
	}); err != nil {
		response.Error(c, err)
		return
	}
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	created, err := h.service.Create(c.Request.Context(), actor, CreateInput{
		Code: body.Code, Name: body.Name, Address: body.Address, Phone: body.Phone,
		Timezone: body.Timezone, IsActive: isActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// Update edits an outlet.
func (h *Handler) Update(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Error(c, apperror.BadRequest("ID outlet tidak valid."))
		return
	}
	var body UpdateRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Code": "code", "Name": "name", "Address": "address", "Phone": "phone",
		"Timezone": "timezone", "IsActive": "is_active",
	}); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.service.Update(c.Request.Context(), actor, id, UpdateInput{
		Code: body.Code, Name: body.Name, Address: body.Address, Phone: body.Phone,
		Timezone: body.Timezone, IsActive: body.IsActive,
	})
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
	tenantID := ""
	if identity.TenantID != nil {
		tenantID = identity.TenantID.String()
	}
	return Actor{
		UserID: identity.UserID.String(), TenantID: tenantID,
		Role: identity.Role, Name: identity.Name,
	}, nil
}
