package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Allowed transition map as specified by BR-02 and section 7.1.
var allowedTransitions = map[string]map[string]bool{
	"received": {
		"waiting_diagnosis": true,
		"cancelled":         true,
	},
	"waiting_diagnosis": {
		"waiting_approval": true,
		"waiting_part":     true,
		"in_progress":      true,
		"cancelled":        true,
	},
	"waiting_approval": {
		"waiting_part": true,
		"in_progress":  true,
		"cancelled":    true,
	},
	"waiting_part": {
		"in_progress": true,
		"cancelled":   true,
	},
	"in_progress": {
		"waiting_part": true,
		"testing":      true,
		"cancelled":    true,
	},
	"testing": {
		"in_progress": true,
		"completed":   true,
		"cancelled":   true,
	},
	"completed": {
		"ready_for_pickup": true,
		"in_progress":      true,
		"cancelled":        true,
	},
	"ready_for_pickup": {
		"picked_up": true,
		"completed": true,
		"cancelled": true,
	},
	"picked_up": {},
	"cancelled": {},
}

// ChangeStatus transitions a service order through the state machine.
func (s *Service) ChangeStatus(ctx context.Context, actor Actor, orderID uuid.UUID, input ChangeStatusRequest) (*ServiceOrderDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	targetStatus := strings.ToLower(strings.TrimSpace(input.Status))
	if _, ok := allowedStatuses[targetStatus]; !ok {
		return nil, apperror.Validation(map[string]string{"status": "Status tidak valid."})
	}

	order, err := s.findOrder(ctx, actor, orderID)
	if err != nil {
		return nil, err
	}

	currentStatus := order.Status
	if currentStatus == targetStatus {
		return s.GetServiceOrder(ctx, actor, orderID)
	}

	// Cancellation has its own permission so technicians and cashiers cannot
	// abort an order accidentally (docs/07 RBAC matrix).
	if targetStatus == "cancelled" && !actor.HasPermission("service.cancel") {
		return nil, apperror.Forbidden("FORBIDDEN", "Anda tidak memiliki izin untuk membatalkan servis.")
	}
	// Cashiers may only complete hand-over transitions.
	if actor.Role == "cashier" && targetStatus != "ready_for_pickup" && targetStatus != "picked_up" {
		return nil, apperror.Forbidden("FORBIDDEN", "Kasir hanya dapat mengubah status menjadi siap diambil atau sudah diambil.")
	}

	// Validate allowed transition
	validTargets := allowedTransitions[currentStatus]
	if validTargets == nil || !validTargets[targetStatus] {
		return nil, apperror.Conflict("INVALID_STATUS_TRANSITION",
			fmt.Sprintf("Transisi status dari '%s' ke '%s' tidak diizinkan.", currentStatus, targetStatus))
	}

	// Never cancel an order that already has money recorded against it.
	if targetStatus == "cancelled" {
		var invoice struct {
			PaidAmount int64  `gorm:"column:paid_amount"`
			Status     string `gorm:"column:status"`
		}
		if err := s.db.WithContext(ctx).Raw(`
			SELECT paid_amount, status FROM invoices
			WHERE service_order_id = ? AND tenant_id = ?`, orderID, tenantID).
			Scan(&invoice).Error; err != nil {
			return nil, apperror.Internal(err)
		}
		if invoice.PaidAmount > 0 || invoice.Status == "paid" || invoice.Status == "partial" {
			return nil, apperror.Conflict("INVOICE_LOCKED",
				"Servis tidak dapat dibatalkan karena invoice sudah memiliki pembayaran.")
		}
	}

	// Validate cancel reason
	reason := ""
	if input.CancelReason != nil {
		reason = strings.TrimSpace(*input.CancelReason)
	} else if input.Notes != nil {
		reason = strings.TrimSpace(*input.Notes)
	}
	if targetStatus == "cancelled" && reason == "" {
		return nil, apperror.Validation(map[string]string{
			"cancel_reason": "Alasan pembatalan wajib diisi.",
		})
	}
	overrideReason := ""
	if input.OverrideReason != nil {
		overrideReason = strings.TrimSpace(*input.OverrideReason)
	}

	now := time.Now().UTC()

	var pickupOverride bool
	var pickupInvoiceTotal, pickupInvoicePaid int64
	var pickupInvoiceNo string

	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		// A hand-over is only allowed when the invoice is settled. Owners may
		// override the rule with a written reason that is stored in the audit
		// log and the status history.
		if targetStatus == "picked_up" {
			var invoice struct {
				InvoiceNo   string `gorm:"column:invoice_no"`
				TotalAmount int64  `gorm:"column:total_amount"`
				PaidAmount  int64  `gorm:"column:paid_amount"`
			}
			if err := tx.Raw(`
				SELECT invoice_no, total_amount, paid_amount
				FROM invoices
				WHERE service_order_id = ? AND tenant_id = ?
				FOR UPDATE`, orderID, tenantID).Scan(&invoice).Error; err != nil {
				return apperror.Internal(err)
			}
			if invoice.InvoiceNo != "" && invoice.TotalAmount > invoice.PaidAmount {
				if !actor.HasPermission("service.override_unpaid_pickup") {
					return apperror.Conflict("UNPAID_PICKUP",
						"Servis belum lunas. Selesaikan pembayaran terlebih dahulu atau minta Owner melakukan override.")
				}
				if overrideReason == "" {
					return apperror.Validation(map[string]string{
						"override_reason": "Alasan override pickup belum lunas wajib diisi.",
					})
				}
				pickupOverride = true
				pickupInvoiceTotal = invoice.TotalAmount
				pickupInvoicePaid = invoice.PaidAmount
				pickupInvoiceNo = invoice.InvoiceNo
			}
		}

		// Update order status and timestamp side effects
		query := `UPDATE service_orders SET status = ?, updated_by = ?, updated_at = ?`
		args := []any{targetStatus, actorID(actor), now}

		switch targetStatus {
		case "in_progress":
			query += `, started_at = COALESCE(started_at, ?)`
			args = append(args, now)
			if currentStatus == "completed" {
				query += `, completed_at = NULL`
			}
		case "completed":
			query += `, completed_at = COALESCE(completed_at, ?)`
			args = append(args, now)
		case "picked_up":
			purgeAt := now.Add(30 * 24 * time.Hour)
			tokenExpiry := now.AddDate(0, 0, 90)
			query += `, picked_up_at = ?, credential_purge_at = ?, completed_at = COALESCE(completed_at, ?), public_token_expires_at = ?`
			args = append(args, now, purgeAt, now, tokenExpiry)
		case "cancelled":
			query += `, cancelled_at = ?, cancel_reason = ?`
			args = append(args, now, reason)

			// Restock any spareparts used in this order
			type itemRow struct {
				SparepartID uuid.UUID `gorm:"column:sparepart_id"`
				Qty         float64   `gorm:"column:qty"`
				CostPrice   int64     `gorm:"column:cost_price"`
			}
			var items []itemRow
			if findErr := tx.Raw(`
				SELECT sparepart_id, qty, cost_price
				FROM service_order_items
				WHERE service_order_id = ? AND tenant_id = ? AND item_type = 'sparepart' AND sparepart_id IS NOT NULL`,
				orderID, tenantID).Scan(&items).Error; findErr != nil {
				return apperror.Internal(findErr)
			}

			for _, it := range items {
				qtyInt := int(it.Qty)
				if qtyInt <= 0 {
					continue
				}
				var currentStock int
				if err := tx.Raw(`SELECT stock FROM spareparts WHERE id = ? AND tenant_id = ? FOR UPDATE`,
					it.SparepartID, tenantID).Scan(&currentStock).Error; err != nil {
					return apperror.Internal(err)
				}

				newStock := currentStock + qtyInt
				if err := tx.Exec(`UPDATE spareparts SET stock = ?, updated_at = ? WHERE id = ? AND tenant_id = ?`,
					newStock, now, it.SparepartID, tenantID).Error; err != nil {
					return apperror.Internal(err)
				}

				movementID := uuid.New()
				if err := tx.Exec(`
					INSERT INTO inventory_movements (
						id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
						stock_before, stock_after, unit_cost, reference_type, reference_id,
						notes, created_by, created_at
					) VALUES (?, ?, ?, ?, 'service_return', ?, ?, ?, ?, 'service_order', ?, ?, ?, ?)`,
					movementID, tenantID, order.OutletID, it.SparepartID, qtyInt,
					currentStock, newStock, it.CostPrice, orderID,
					"Pengembalian stok karena servis dibatalkan: "+reason, actorID(actor), now).Error; err != nil {
					return apperror.Internal(err)
				}
			}

			// Void any invoice that has not received money yet (draft or issued
			// unpaid), as required by BR-10.
			if err := tx.Exec(`
				UPDATE invoices SET status = 'void', updated_at = ?
				WHERE service_order_id = ? AND tenant_id = ? AND status IN ('draft', 'unpaid')`,
				now, orderID, tenantID).Error; err != nil {
				return apperror.Internal(err)
			}
		}

		query += ` WHERE id = ? AND tenant_id = ?`
		args = append(args, orderID, tenantID)

		if err := tx.Exec(query, args...).Error; err != nil {
			return apperror.Internal(err)
		}

		// Insert into service_status_history
		historyID := uuid.New()
		historyNotes := input.Notes
		if targetStatus == "cancelled" && (historyNotes == nil || *historyNotes == "") {
			historyNotes = &reason
		}
		if pickupOverride {
			overrideNote := "Override pickup belum lunas: " + overrideReason
			if historyNotes != nil && strings.TrimSpace(*historyNotes) != "" {
				overrideNote = strings.TrimSpace(*historyNotes) + " | " + overrideNote
			}
			historyNotes = &overrideNote
		}
		if err := tx.Exec(`
			INSERT INTO service_status_history (
				id, tenant_id, service_order_id, from_status, to_status, notes, changed_by, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			historyID, tenantID, orderID, currentStatus, targetStatus, historyNotes, actorID(actor), now).Error; err != nil {
			return apperror.Internal(err)
		}

		// Audit log
		if err := audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "service_status_change",
			Module:     "service",
			EntityType: "service_order",
			EntityID:   &orderID,
			After:      map[string]any{"from": currentStatus, "to": targetStatus, "notes": historyNotes},
		}); err != nil {
			return err
		}
		if pickupOverride {
			if err := audit.Write(ctx, tx, audit.Entry{
				TenantID:   &tenantID,
				UserID:     actorID(actor),
				ActorName:  actor.Name,
				Action:     "service_pickup_override",
				Module:     "service",
				EntityType: "service_order",
				EntityID:   &orderID,
				After: map[string]any{
					"invoice_no":       pickupInvoiceNo,
					"total_amount":     pickupInvoiceTotal,
					"paid_amount":      pickupInvoicePaid,
					"remaining_amount": pickupInvoiceTotal - pickupInvoicePaid,
					"reason":           overrideReason,
					"approved_by":      actor.Name,
				},
			}); err != nil {
				return err
			}
		}

		if targetStatus == "picked_up" && s.pickupHook != nil {
			if err := s.pickupHook(ctx, tx, tenantID, order.OutletID, orderID, order.CustomerID, now); err != nil {
				return err
			}
		}
		if s.statusHook != nil {
			if err := s.statusHook(ctx, tx, StatusEvent{
				TenantID: tenantID, OutletID: order.OutletID, OrderID: orderID,
				CustomerID: order.CustomerID, TicketNo: order.TicketNo,
				FromStatus: currentStatus, ToStatus: targetStatus,
				ActorID: actorID(actor), PickedUpAt: now,
			}); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetServiceOrder(ctx, actor, orderID)
}

// AssignTechnician assigns a technician and optional priority to a service order.
func (s *Service) AssignTechnician(ctx context.Context, actor Actor, orderID uuid.UUID, input AssignTechnicianRequest) (*ServiceOrderDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	order, err := s.findOrder(ctx, actor, orderID)
	if err != nil {
		return nil, err
	}

	var techID *uuid.UUID
	if input.TechnicianID != nil && strings.TrimSpace(*input.TechnicianID) != "" {
		parsed, parseErr := uuid.Parse(strings.TrimSpace(*input.TechnicianID))
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"technician_id": "ID teknisi tidak valid."})
		}
		// Verify the user is an active technician assigned to this outlet.
		exists, checkErr := s.technicianExists(ctx, tenantID, order.OutletID, parsed)
		if checkErr != nil {
			return nil, apperror.Internal(checkErr)
		}
		if !exists {
			return nil, apperror.NotFound("Teknisi tidak ditemukan, tidak aktif, atau tidak bertugas di outlet ini.")
		}
		techID = &parsed
	}

	now := time.Now().UTC()
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		query := `UPDATE service_orders SET assigned_technician_id = ?, updated_by = ?, updated_at = ?`
		args := []any{techID, actorID(actor), now}

		if input.Priority != nil && (*input.Priority == "normal" || *input.Priority == "express") {
			query += `, priority = ?`
			args = append(args, *input.Priority)
		}

		query += ` WHERE id = ? AND tenant_id = ?`
		args = append(args, orderID, tenantID)

		if err := tx.Exec(query, args...).Error; err != nil {
			return apperror.Internal(err)
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "service_assign",
			Module:     "service",
			EntityType: "service_order",
			EntityID:   &orderID,
			After: map[string]any{
				"assigned_technician_id": techID,
				"priority":               input.Priority,
			},
		})
	})

	if err != nil {
		return nil, err
	}

	return s.GetServiceOrder(ctx, actor, orderID)
}
