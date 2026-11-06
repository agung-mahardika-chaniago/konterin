package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetDiagnostic fetches the diagnosis record for a service order.
func (s *Service) GetDiagnostic(ctx context.Context, actor Actor, orderID uuid.UUID) (*DiagnosticDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	row, err := s.findOrder(ctx, actor, orderID)
	if err != nil {
		return nil, err
	}
	return s.diagnosticForOrder(ctx, tenantID, row.ID)
}

// UpsertDiagnostic creates or updates a service order diagnosis.
func (s *Service) UpsertDiagnostic(ctx context.Context, actor Actor, orderID uuid.UUID, input UpsertDiagnosticRequest) (*DiagnosticDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	_, err = s.findOrder(ctx, actor, orderID)
	if err != nil {
		return nil, err
	}

	var estDone *time.Time
	if input.EstimatedDoneAt != nil && strings.TrimSpace(*input.EstimatedDoneAt) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(*input.EstimatedDoneAt))
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"estimated_done_at": "Format tanggal harus RFC3339."})
		}
		estDone = &parsed
	}

	now := time.Now().UTC()
	var diagID uuid.UUID

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingID uuid.UUID
		queryErr := tx.Raw(`SELECT id FROM service_diagnostics WHERE service_order_id = ? AND tenant_id = ?`,
			orderID, tenantID).Scan(&existingID).Error

		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return apperror.Internal(queryErr)
		}

		if existingID != uuid.Nil {
			diagID = existingID
			updateErr := tx.Exec(`
				UPDATE service_diagnostics
				SET diagnosis = ?, findings = ?, action_taken = ?, test_result = ?,
				    technician_notes = ?, diagnosed_by = ?, diagnosed_at = ?,
				    estimated_done_at = ?, updated_at = ?
				WHERE id = ? AND tenant_id = ?`,
				input.Diagnosis, input.Findings, input.ActionTaken, input.TestResult,
				input.TechnicianNotes, actorID(actor), now, estDone, now, diagID, tenantID).Error
			if updateErr != nil {
				return apperror.Internal(updateErr)
			}
		} else {
			diagID = uuid.New()
			insertErr := tx.Exec(`
				INSERT INTO service_diagnostics (
					id, tenant_id, service_order_id, diagnosis, findings, action_taken,
					test_result, technician_notes, diagnosed_by, diagnosed_at,
					estimated_done_at, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				diagID, tenantID, orderID, input.Diagnosis, input.Findings, input.ActionTaken,
				input.TestResult, input.TechnicianNotes, actorID(actor), now, estDone, now, now).Error
			if insertErr != nil {
				return apperror.Internal(insertErr)
			}
		}

		// Also sync estimated_done_at to service_orders if provided
		if estDone != nil {
			if err := tx.Exec(`UPDATE service_orders SET estimated_done_at = ?, updated_at = ? WHERE id = ? AND tenant_id = ?`,
				estDone, now, orderID, tenantID).Error; err != nil {
				return apperror.Internal(err)
			}
		}

		// Audit log
		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "diagnostic_upsert",
			Module:     "service",
			EntityType: "service_order",
			EntityID:   &orderID,
			After:      map[string]any{"diagnostic_id": diagID.String()},
		})
	})

	if err != nil {
		return nil, err
	}

	return s.diagnosticForOrder(ctx, tenantID, orderID)
}

func (s *Service) diagnosticForOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*DiagnosticDTO, error) {
	type diagRow struct {
		ID              uuid.UUID  `gorm:"column:id"`
		Diagnosis       *string    `gorm:"column:diagnosis"`
		Findings        *string    `gorm:"column:findings"`
		ActionTaken     *string    `gorm:"column:action_taken"`
		TestResult      *string    `gorm:"column:test_result"`
		TechnicianNotes *string    `gorm:"column:technician_notes"`
		DiagnosedByID   *uuid.UUID `gorm:"column:diagnosed_by"`
		DiagnosedBy     *string    `gorm:"column:diagnosed_by_name"`
		DiagnosedAt     *time.Time `gorm:"column:diagnosed_at"`
		EstimatedDoneAt *time.Time `gorm:"column:estimated_done_at"`
		UpdatedAt       time.Time  `gorm:"column:updated_at"`
	}

	var row diagRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT d.id, d.diagnosis, d.findings, d.action_taken, d.test_result,
		       d.technician_notes, d.diagnosed_by, u.name AS diagnosed_by_name,
		       d.diagnosed_at, d.estimated_done_at, d.updated_at
		FROM service_diagnostics d
		LEFT JOIN users u ON u.id = d.diagnosed_by
		WHERE d.service_order_id = ? AND d.tenant_id = ?
		LIMIT 1`, orderID, tenantID).Scan(&row).Error

	if err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}

	dto := &DiagnosticDTO{
		ID:              row.ID.String(),
		Diagnosis:       row.Diagnosis,
		Findings:        row.Findings,
		ActionTaken:     row.ActionTaken,
		TestResult:      row.TestResult,
		TechnicianNotes: row.TechnicianNotes,
		DiagnosedBy:     row.DiagnosedBy,
		DiagnosedAt:     row.DiagnosedAt,
		EstimatedDoneAt: row.EstimatedDoneAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.DiagnosedByID != nil {
		idStr := row.DiagnosedByID.String()
		dto.DiagnosedByID = &idStr
	}
	return dto, nil
}
