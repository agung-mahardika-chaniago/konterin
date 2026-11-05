// Package validator binds JSON bodies and converts validation failures into
// the standard Indonesian field-error envelope.
package validator

import (
	"errors"
	"net/http"
	"strings"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Bind decodes and validates a JSON body. labels maps struct field names to the
// user-facing field path (for example "Email": "email").
func Bind(c *gin.Context, dst any, labels map[string]string) *apperror.Error {
	if err := c.ShouldBindJSON(dst); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			fields := make(map[string]string, len(validationErrors))
			for _, fieldErr := range validationErrors {
				name := fieldErr.Field()
				if mapped, ok := labels[name]; ok {
					name = mapped
				}
				fields[name] = message(fieldErr)
			}
			return apperror.Validation(fields)
		}
		return apperror.New(http.StatusBadRequest, "BAD_REQUEST", "Body request tidak dapat diproses.").Wrap(err)
	}
	return nil
}

func message(field validator.FieldError) string {
	switch field.Tag() {
	case "required":
		return "Wajib diisi."
	case "email":
		return "Format email tidak valid."
	case "min":
		return "Nilai terlalu pendek (minimal " + field.Param() + ")."
	case "max":
		return "Nilai terlalu panjang (maksimal " + field.Param() + ")."
	case "gt":
		return "Nilai harus lebih dari " + field.Param() + "."
	case "gte":
		return "Nilai minimal " + field.Param() + "."
	case "oneof":
		return "Nilai harus salah satu dari: " + strings.ReplaceAll(field.Param(), " ", ", ") + "."
	case "uuid":
		return "Format ID tidak valid."
	default:
		return "Nilai tidak valid."
	}
}
