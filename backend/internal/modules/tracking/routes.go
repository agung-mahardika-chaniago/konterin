package tracking

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the unauthenticated, rate-limited public tracking API.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, limiter gin.HandlerFunc) {
	group := api.Group("/public", limiter)
	group.GET("/track", h.Track)
	group.GET("/track/:token", h.TrackToken)
	group.GET("/store", h.Store)
}
