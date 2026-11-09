package tracking

import (
	"strings"

	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// Handler exposes public tracking endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the tracking handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// Track handles ticket + last-four-phone lookup.
func (h *Handler) Track(c *gin.Context) {
	ticket := strings.TrimSpace(c.Query("ticket"))
	last4 := strings.TrimSpace(c.Query("phone_last4"))
	result, err := h.service.FindByTicket(c.Request.Context(), ticket, last4)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// TrackToken handles the direct QR/deep link.
func (h *Handler) TrackToken(c *gin.Context) {
	result, err := h.service.FindByToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Store returns the public store identity for a valid tracking token.
func (h *Handler) Store(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		response.Error(c, apperror.Validation(map[string]string{
			"token": "Token tracking wajib diisi.",
		}))
		return
	}
	store, err := h.service.StoreByToken(c.Request.Context(), token)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, store)
}
