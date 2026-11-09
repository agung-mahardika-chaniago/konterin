package warranty

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts warranty and claim endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/warranties", can("warranty.view"), h.ListWarranties)
	group.GET("/warranties/:id", can("warranty.view"), h.GetWarranty)
	group.POST("/warranties/:id/claims", can("warranty.claim"), h.CreateClaim)

	group.GET("/claims", can("warranty.view"), h.ListClaims)
	group.GET("/claims/:id", can("warranty.view"), h.GetClaim)
	group.PATCH("/claims/:id", can("warranty.claim"), h.UpdateClaim)
	group.POST("/claims/:id/rework", can("service.create"), can("warranty.claim"), h.CreateRework)
}
