package outlet

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts outlet endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)
	group.GET("/outlets", can("outlet.view"), h.List)
	group.POST("/outlets", can("outlet.manage"), h.Create)
	group.PATCH("/outlets/:id", can("outlet.manage"), h.Update)
}
