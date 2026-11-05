// Package response writes the standard FixPhone API envelope.
package response

import (
	"log/slog"
	"net/http"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
)

// RequestIDKey is the gin context key populated by the request-id middleware.
const RequestIDKey = "request_id"

// Meta is the standard pagination metadata.
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Success writes a 200 envelope.
func Success(c *gin.Context, data any) {
	JSON(c, http.StatusOK, data, nil)
}

// Created writes a 201 envelope.
func Created(c *gin.Context, data any) {
	JSON(c, http.StatusCreated, data, nil)
}

// JSON writes an envelope with optional metadata.
func JSON(c *gin.Context, status int, data any, meta any) {
	body := gin.H{"success": true, "data": data}
	if meta != nil {
		body["meta"] = meta
	}
	c.JSON(status, body)
}

// NoContent writes a 204 response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error maps a domain error to the standard error envelope.
func Error(c *gin.Context, err error) {
	appErr, ok := apperror.As(err)
	if !ok || appErr.Status >= http.StatusInternalServerError {
		slog.Error("api_error",
			"request_id", c.GetString(RequestIDKey),
			"path", c.FullPath(),
			"error", err,
		)
		if !ok {
			appErr = apperror.Internal(err)
		}
	}

	errorBody := gin.H{
		"code":    appErr.Code,
		"message": appErr.Message,
	}
	if appErr.Details != nil {
		errorBody["details"] = appErr.Details
	}
	c.AbortWithStatusJSON(appErr.Status, gin.H{
		"success":    false,
		"error":      errorBody,
		"request_id": c.GetString(RequestIDKey),
	})
}

// RequestID returns the request id stored by middleware.
func RequestID(c *gin.Context) string {
	return c.GetString(RequestIDKey)
}

// PageMeta calculates pagination metadata.
func PageMeta(page, perPage int, total int64) Meta {
	totalPages := 0
	if perPage > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}
	return Meta{Page: page, PerPage: perPage, Total: total, TotalPages: totalPages}
}
