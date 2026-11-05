package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type documentSettings struct {
	TicketPrefix  string `gorm:"column:ticket_prefix"`
	TicketFormat  string `gorm:"column:ticket_format"`
	InvoicePrefix string `gorm:"column:invoice_prefix"`
	InvoiceFormat string `gorm:"column:invoice_format"`
}

func (s *Service) settingsFor(ctx context.Context, tenantID, outletID uuid.UUID) (documentSettings, error) {
	settings := documentSettings{}
	err := s.db.WithContext(ctx).Raw(`
		SELECT ticket_prefix, ticket_format, invoice_prefix, invoice_format
		FROM business_settings
		WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
		ORDER BY (outlet_id IS NULL)
		LIMIT 1`, tenantID, outletID).Scan(&settings).Error
	if err != nil {
		return settings, err
	}
	if strings.TrimSpace(settings.TicketPrefix) == "" {
		settings.TicketPrefix = "FP"
	}
	if strings.TrimSpace(settings.TicketFormat) == "" {
		settings.TicketFormat = "{PREFIX}-{YYMMDD}-{SEQ:4}"
	}
	if strings.TrimSpace(settings.InvoicePrefix) == "" {
		settings.InvoicePrefix = "INV"
	}
	if strings.TrimSpace(settings.InvoiceFormat) == "" {
		settings.InvoiceFormat = "{PREFIX}-{YYMMDD}-{SEQ:4}"
	}
	return settings, nil
}

// nextDocumentNumber atomically increments a per-day document sequence.
func (s *Service) nextDocumentNumber(
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
	return formatDocumentNumber(format, prefix, now, number), nil
}

func formatDocumentNumber(format, prefix string, now time.Time, number int64) string {
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

// newPublicToken returns the 32-character token stored in service_orders.
func newPublicToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", apperror.Internal(err)
	}
	return hex.EncodeToString(buffer), nil
}
