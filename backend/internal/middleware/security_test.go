package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("without HSTS", func(t *testing.T) {
		r := gin.New()
		r.Use(SecurityHeaders(false))
		r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

		want := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
			"Permissions-Policy":     "camera=(self), microphone=(), geolocation=()",
			"Cache-Control":          "no-store",
		}
		for header, value := range want {
			if got := rec.Header().Get(header); got != value {
				t.Errorf("%s = %q, want %q", header, got, value)
			}
		}
		if got := rec.Header().Get("Content-Security-Policy"); got == "" {
			t.Error("Content-Security-Policy is empty")
		}
		if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
			t.Errorf("Strict-Transport-Security = %q, want empty outside production", got)
		}
	})

	t.Run("with HSTS", func(t *testing.T) {
		r := gin.New()
		r.Use(SecurityHeaders(true))
		r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

		if got, want := rec.Header().Get("Strict-Transport-Security"), "max-age=31536000; includeSubDomains"; got != want {
			t.Errorf("Strict-Transport-Security = %q, want %q", got, want)
		}
	})
}

func TestCheckOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CheckOrigin([]string{"https://app.example.com"}))
	r.POST("/mutate", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	cases := []struct {
		name   string
		origin string
		want   int
	}{
		{"allowed origin", "https://app.example.com", http.StatusNoContent},
		{"foreign origin", "https://evil.example.com", http.StatusForbidden},
		{"server to server without origin", "", http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mutate", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}
