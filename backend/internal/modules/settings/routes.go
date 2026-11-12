package settings

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts settings and backup endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("/settings", auth, tenant)

	group.GET("", can("settings.view"), h.Get)
	group.PATCH("", can("settings.manage"), h.Update)
	group.GET("/backups", can("settings.manage"), h.ListBackups)
	group.POST("/backups", can("settings.manage"), h.CreateBackup)
	group.GET("/backups/:id/download", can("settings.manage"), h.DownloadBackup)
}
