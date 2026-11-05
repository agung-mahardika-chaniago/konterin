package user

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts tenant user, role, and permission endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/users", can("user.view"), h.List)
	group.POST("/users", can("user.manage"), h.Create)
	group.GET("/users/:id", can("user.view"), h.Get)
	group.PATCH("/users/:id", can("user.manage"), h.Update)
	group.DELETE("/users/:id", can("user.manage"), h.Delete)
	group.POST("/users/:id/reset-password", can("user.manage"), h.ResetPassword)
	group.PUT("/users/:id/outlets", can("user.manage"), h.ReplaceOutlets)

	group.GET("/roles", can("user.view"), h.Roles)
	group.GET("/permissions", can("user.view"), h.Permissions)
}
