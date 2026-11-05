package dashboard

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the dashboard endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("/dashboard", auth, tenant)
	group.GET("/summary", can("dashboard.view"), h.Summary)
	group.GET("/charts", can("dashboard.view"), h.Charts)
	group.GET("/alerts", can("dashboard.view"), h.Alerts)
	group.GET("/technicians", can("dashboard.view"), h.Technicians)
}
