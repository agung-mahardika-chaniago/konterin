// Package audit provides append-only audit logging helpers.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entry describes one audit event.
type Entry struct {
	TenantID   *uuid.UUID
	UserID     *uuid.UUID
	ActorName  string
	Action     string
	Module     string
	EntityType string
	EntityID   *uuid.UUID
	Before     any
	After      any
	IP         string
	UserAgent  string
	RequestID  string
}

// Write inserts an audit event using the supplied DB or transaction handle.
func Write(ctx context.Context, db *gorm.DB, entry Entry) error {
	if db == nil {
		return nil
	}
	ip := nullableString(entry.IP, 64)
	userAgent := nullableString(entry.UserAgent, 255)
	requestID := nullableString(entry.RequestID, 64)
	actor := nullableString(entry.ActorName, 120)

	var before, after any
	if entry.Before != nil {
		if raw, err := json.Marshal(entry.Before); err == nil {
			before = string(raw)
		}
	}
	if entry.After != nil {
		if raw, err := json.Marshal(entry.After); err == nil {
			after = string(raw)
		}
	}

	err := db.WithContext(ctx).Exec(`
		INSERT INTO audit_logs (
			tenant_id, user_id, actor_name, action, module, entity_type, entity_id,
			before_data, after_data, ip, user_agent, request_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.TenantID, entry.UserID, actor, entry.Action, entry.Module,
		entry.EntityType, entry.EntityID, before, after, ip, userAgent, requestID,
	).Error
	if err != nil {
		slog.Warn("audit_write_failed", "action", entry.Action, "error", err)
	}
	return err
}

// BestEffort writes an event and never fails the caller.
func BestEffort(ctx context.Context, db *gorm.DB, entry Entry) {
	_ = Write(ctx, db, entry)
}

func nullableString(value string, max int) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) > max {
		value = value[:max]
	}
	return value
}

// Now returns the UTC clock used for audit records in tests.
func Now() time.Time { return time.Now().UTC() }
