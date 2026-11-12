// Package auditview exposes the read-only audit log query endpoint.
package auditview

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service queries the append-only audit trail.
type Service struct {
	db *gorm.DB
}

// NewService builds the audit viewer.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// Filter is the audit log query filter set.
type Filter struct {
	TenantID  uuid.UUID
	UserID    *uuid.UUID
	Module    string
	Action    string
	EntityType string
	From      *time.Time
	To        *time.Time
	Page      int
	PerPage   int
}

// Row is one audit entry as returned to the client.
type Row struct {
	ID         int64           `json:"id"`
	ActorName  string          `json:"actor_name"`
	Action     string          `json:"action"`
	Module     string          `json:"module"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id,omitempty"`
	Before     json.RawMessage `json:"before_data,omitempty"`
	After      json.RawMessage `json:"after_data,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// List returns the paginated audit trail for the tenant.
func (s *Service) List(ctx context.Context, filter Filter) ([]Row, int64, error) {
	where := ` WHERE tenant_id = ?`
	args := []any{filter.TenantID}
	if filter.UserID != nil {
		where += ` AND user_id = ?`
		args = append(args, *filter.UserID)
	}
	if strings.TrimSpace(filter.Module) != "" {
		where += ` AND module = ?`
		args = append(args, strings.TrimSpace(filter.Module))
	}
	if strings.TrimSpace(filter.Action) != "" {
		where += ` AND action ILIKE ?`
		args = append(args, "%"+strings.TrimSpace(filter.Action)+"%")
	}
	if strings.TrimSpace(filter.EntityType) != "" {
		where += ` AND entity_type = ?`
		args = append(args, strings.TrimSpace(filter.EntityType))
	}
	if filter.From != nil {
		where += ` AND created_at >= ?`
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		where += ` AND created_at < ?`
		args = append(args, filter.To.AddDate(0, 0, 1))
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM audit_logs`+where, args...).Scan(&total).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	queryArgs := append(append([]any{}, args...), filter.PerPage, offset)
	var rows []Row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, COALESCE(actor_name, '') AS actor_name, action, module, entity_type,
			COALESCE(entity_id::text, '') AS entity_id,
			before_data, after_data, created_at
		FROM audit_logs`+where+`
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}
	return rows, total, nil
}

// Modules lists the distinct modules present in the audit trail.
func (s *Service) Modules(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var modules []string
	if err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT module FROM audit_logs WHERE tenant_id = ? ORDER BY module`,
		tenantID).Scan(&modules).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return modules, nil
}


