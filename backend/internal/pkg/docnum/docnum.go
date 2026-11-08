// Package docnum generates atomic, per-outlet document numbers.
package docnum

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var sequencePattern = regexp.MustCompile(`\{SEQ:(\d+)\}`)

// Next increments the per-day sequence for docType and formats the value.
//
// The INSERT ... ON CONFLICT statement makes concurrent document creation
// safe: PostgreSQL serializes the update on the unique key and each caller
// receives a distinct number.
func Next(
	ctx context.Context,
	tx *gorm.DB,
	tenantID, outletID uuid.UUID,
	docType, prefix, format string,
	now time.Time,
) (string, error) {
	periodKey := now.Format("20060102")
	var number int64
	if err := tx.WithContext(ctx).Raw(`
		INSERT INTO document_sequences (tenant_id, outlet_id, doc_type, period_key, last_number)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT (tenant_id, outlet_id, doc_type, period_key)
		DO UPDATE SET last_number = document_sequences.last_number + 1, updated_at = now()
		RETURNING last_number`,
		tenantID, outletID, docType, periodKey).Scan(&number).Error; err != nil {
		return "", err
	}
	if number < 1 {
		number = 1
	}
	return Format(format, prefix, now, number), nil
}

// Format expands the supported placeholders in a document number format.
func Format(format, prefix string, now time.Time, number int64) string {
	if strings.TrimSpace(format) == "" {
		format = "{PREFIX}-{YYMMDD}-{SEQ:4}"
	}
	formatted := strings.ReplaceAll(format, "{PREFIX}", prefix)
	formatted = strings.ReplaceAll(formatted, "{YYYY}", now.Format("2006"))
	formatted = strings.ReplaceAll(formatted, "{YYMMDD}", now.Format("060102"))
	formatted = strings.ReplaceAll(formatted, "{MM}", now.Format("01"))
	formatted = strings.ReplaceAll(formatted, "{DD}", now.Format("02"))
	formatted = sequencePattern.ReplaceAllStringFunc(formatted, func(token string) string {
		match := sequencePattern.FindStringSubmatch(token)
		if len(match) != 2 {
			return token
		}
		width, err := strconv.Atoi(match[1])
		if err != nil || width < 1 || width > 10 {
			width = 4
		}
		return fmt.Sprintf("%0*d", width, number)
	})
	if !strings.Contains(format, "{SEQ:") {
		formatted = fmt.Sprintf("%s-%04d", formatted, number)
	}
	return formatted
}
