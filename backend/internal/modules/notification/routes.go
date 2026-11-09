package notification

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts notification, WhatsApp outbox, and staff-notify routes.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/notifications", h.List)
	group.GET("/notifications/unread-count", h.UnreadCount)
	group.PATCH("/notifications/:id/read", h.MarkRead)
	group.POST("/notifications/read-all", h.MarkAllRead)

	group.GET("/whatsapp-messages", can("settings.manage"), h.ListOutbox)
	group.POST("/whatsapp-messages/:id/retry", can("settings.manage"), h.RetryOutbox)

	group.POST("/service-orders/:id/notify", can("service.update"), h.NotifyServiceOrder)
}
