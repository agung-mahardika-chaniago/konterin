package middleware

import (
	"net/http"
	"strings"

	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds conservative browser hardening headers to API
// responses. HSTS is only emitted when the deployment is served over HTTPS
// (production); sending it over plain HTTP would be ignored or harmful.
func SecurityHeaders(enableHSTS bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data: blob:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		c.Header("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
		c.Header("Cache-Control", "no-store")
		if enableHSTS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// CheckOrigin rejects state-changing browser requests from unknown origins.
// Requests without an Origin header (CLI, server-to-server) are allowed; they
// cannot ride an ambient cookie.
func CheckOrigin(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			c.Next()
			return
		}
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				response.Error(c, apperror.Forbidden("ORIGIN_FORBIDDEN", "Asal request tidak diizinkan."))
				return
			}
		}
		c.Next()
	}
}
