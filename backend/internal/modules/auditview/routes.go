package auditview

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts audit log endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/audit-logs", can("audit.view"), h.List)
	group.GET("/audit-logs/modules", can("audit.view"), h.Modules)
}
