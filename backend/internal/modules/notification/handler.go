package notification

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

// Handler exposes notification and WhatsApp outbox endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the notification handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// List returns the current user's notification inbox.
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	items, total, unread, err := h.service.ListNotifications(c.Request.Context(), actor, NotificationFilter{
		Unread: strings.EqualFold(c.Query("unread"), "true") || c.Query("unread") == "1",
		Page:   page, PerPage: perPage,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("X-Unread-Count", strconv.FormatInt(unread, 10))
	response.JSON(c, http.StatusOK, items, response.Meta{
		Page: page, PerPage: perPage, Total: total,
		TotalPages: totalPages(total, perPage),
	})
}

// UnreadCount returns the unread badge count.
func (h *Handler) UnreadCount(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	count, err := h.service.UnreadCount(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"count": count})
}

// MarkRead marks one notification read.
func (h *Handler) MarkRead(c *gin.Context) {
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
	if err := h.service.MarkRead(c.Request.Context(), actor, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"id": id.String(), "is_read": true})
}

// MarkAllRead marks every visible notification read.
func (h *Handler) MarkAllRead(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.service.MarkAllRead(c.Request.Context(), actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"unread": 0})
}

// ListOutbox returns WhatsApp outbox rows.
func (h *Handler) ListOutbox(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	page := parseInt(c.Query("page"), 1)
	perPage := normalizePerPage(parseInt(c.Query("per_page"), 20))
	items, total, err := h.service.ListOutbox(c.Request.Context(), actor, OutboxFilter{
		Status: strings.TrimSpace(c.Query("status")), Query: c.Query("q"),
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

// RetryOutbox re-queues a failed WhatsApp message.
func (h *Handler) RetryOutbox(c *gin.Context) {
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
	item, err := h.service.RetryOutbox(c.Request.Context(), actor, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// NotifyServiceOrder queues a WhatsApp message for one service order.
func (h *Handler) NotifyServiceOrder(c *gin.Context) {
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
	var input NotifyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, apperror.Validation(map[string]string{"body": "Payload tidak valid."}))
		return
	}
	result, err := h.service.NotifyServiceOrder(c.Request.Context(), actor, id, input)
	if err != nil {
		response.Error(c, err)
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
	if perPage < 1 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}
