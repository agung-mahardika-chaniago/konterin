package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the auth endpoints.
//
// auth is the bearer-token middleware. loginRateLimit must be applied only to
// the login endpoint.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth gin.HandlerFunc, loginRateLimit gin.HandlerFunc) {
	group := api.Group("/auth")
	group.POST("/login", loginRateLimit, h.Login)
	group.POST("/refresh", h.Refresh)
	group.POST("/logout", auth, h.Logout)
	group.GET("/me", auth, h.Me)
	group.PATCH("/profile", auth, h.UpdateProfile)
	group.POST("/change-password", auth, h.ChangePassword)
}
