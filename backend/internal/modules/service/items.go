package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListItems returns all items (services and spareparts) attached to a service order.
func (s *Service) ListItems(ctx context.Context, actor Actor, orderID uuid.UUID) ([]ServiceOrderItemDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.findOrder(ctx, actor, orderID); err != nil {
		return nil, err
	}
	return s.itemsForOrder(ctx, tenantID, orderID, actor.HasPermission("inventory.cost.view"))
}

// AddItem attaches a service charge or sparepart line to the order and updates inventory.
func (s *Service) AddItem(ctx context.Context, actor Actor, orderID uuid.UUID, input AddServiceItemRequest) (*ServiceOrderItemDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	order, err := s.findOrder(ctx, actor, orderID)
	if err != nil {
		return nil, err
	}

	itemType := strings.ToLower(strings.TrimSpace(input.ItemType))
	if itemType != "service" && itemType != "sparepart" && itemType != "custom" {
		return nil, apperror.Validation(map[string]string{"item_type": "Tipe item harus service, sparepart, atau custom."})
	}
	if input.Qty > 9999 {
		return nil, apperror.Validation(map[string]string{"qty": "Jumlah item maksimal 9999."})
	}
	if itemType == "sparepart" && math.Trunc(input.Qty) != input.Qty {
		return nil, apperror.Validation(map[string]string{"qty": "Jumlah sparepart harus bilangan bulat."})
	}

	itemID := uuid.New()
	now := time.Now().UTC()

	var createdDTO ServiceOrderItemDTO

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check invoice lock: finalized invoices freeze their line items.
		locked, err := invoiceLocked(tx, orderID, tenantID)
		if err != nil {
			return err
		}
		if locked {
			return apperror.Conflict("INVOICE_LOCKED", "Item servis tidak dapat diubah karena invoice sudah difinalisasi.")
		}

		var sparepartID *uuid.UUID
		var unitPrice int64
		var costPrice int64
		description := strings.TrimSpace(input.Description)

		if itemType == "sparepart" {
			if input.SparepartID == nil || strings.TrimSpace(*input.SparepartID) == "" {
				return apperror.Validation(map[string]string{"sparepart_id": "Sparepart wajib dipilih."})
			}
			parsedSpID, parseErr := uuid.Parse(strings.TrimSpace(*input.SparepartID))
			if parseErr != nil {
				return apperror.Validation(map[string]string{"sparepart_id": "ID sparepart tidak valid."})
			}
			sparepartID = &parsedSpID

			type spRow struct {
				ID        uuid.UUID `gorm:"column:id"`
				Name      string    `gorm:"column:name"`
				Stock     int       `gorm:"column:stock"`
				CostPrice int64     `gorm:"column:cost_price"`
				SellPrice int64     `gorm:"column:sell_price"`
				IsActive  bool      `gorm:"column:is_active"`
			}
			var sp spRow
			spErr := tx.Raw(`
				SELECT id, name, stock, cost_price, sell_price, is_active
				FROM spareparts
				WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL
				FOR UPDATE`, parsedSpID, tenantID).Scan(&sp).Error

			if spErr != nil || sp.ID == uuid.Nil {
				return apperror.NotFound("Sparepart tidak ditemukan.")
			}
			if !sp.IsActive {
				return apperror.Conflict("SPAREPART_INACTIVE", "Sparepart ini tidak aktif.")
			}

			qtyNeeded := int(input.Qty)
			if sp.Stock < qtyNeeded {
				return apperror.Conflict("INSUFFICIENT_STOCK",
					fmt.Sprintf("Stok sparepart tidak mencukupi (sisa: %d, diminta: %d).", sp.Stock, qtyNeeded))
			}

			if description == "" {
				description = sp.Name
			}
			if input.UnitPrice != nil {
				unitPrice = *input.UnitPrice
			} else {
				unitPrice = sp.SellPrice
			}
			costPrice = sp.CostPrice

			// Deduct stock
			newStock := sp.Stock - qtyNeeded
			if err := tx.Exec(`
				UPDATE spareparts SET stock = ?, last_stock_at = ?, updated_at = ?
				WHERE id = ? AND tenant_id = ?`,
				newStock, now, now, parsedSpID, tenantID).Error; err != nil {
				return apperror.Internal(err)
			}

			// Record inventory movement
			movementID := uuid.New()
			if err := tx.Exec(`
				INSERT INTO inventory_movements (
					id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
					stock_before, stock_after, unit_cost, reference_type, reference_id,
					notes, created_by, created_at
				) VALUES (?, ?, ?, ?, 'service_out', ?, ?, ?, ?, 'service_order', ?, ?, ?, ?)`,
				movementID, tenantID, order.OutletID, parsedSpID, -qtyNeeded,
				sp.Stock, newStock, costPrice, orderID,
				"Pemakaian untuk tiket servis "+order.TicketNo, actorID(actor), now).Error; err != nil {
				return apperror.Internal(err)
			}
		} else {
			if description == "" {
				return apperror.Validation(map[string]string{"description": "Deskripsi jasa wajib diisi."})
			}
			if input.UnitPrice != nil {
				unitPrice = *input.UnitPrice
			}
			costPrice = 0
		}

		gross := int64(input.Qty * float64(unitPrice))
		if input.DiscountAmount > gross {
			return apperror.Validation(map[string]string{"discount_amount": "Diskon baris tidak boleh melebihi harga item."})
		}
		subtotal := gross - input.DiscountAmount

		// Insert item line
		if err := tx.Exec(`
			INSERT INTO service_order_items (
				id, tenant_id, service_order_id, item_type, sparepart_id,
				description, qty, unit_price, cost_price, discount_amount,
				subtotal, warranty_days, added_by, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			itemID, tenantID, orderID, itemType, sparepartID,
			description, input.Qty, unitPrice, costPrice, input.DiscountAmount,
			subtotal, input.WarrantyDays, actorID(actor), now, now).Error; err != nil {
			return apperror.Internal(err)
		}

		// Recalculate draft invoice and order estimated cost.
		if _, err := recalcOrderTotals(tx, tenantID, orderID, now); err != nil {
			return err
		}

		// Audit log
		if err := audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "service_item_add",
			Module:     "service",
			EntityType: "service_order",
			EntityID:   &orderID,
			After: map[string]any{
				"item_id":     itemID.String(),
				"item_type":   itemType,
				"description": description,
				"subtotal":    subtotal,
			},
		}); err != nil {
			return apperror.Internal(err)
		}

		var spIDStr *string
		if sparepartID != nil {
			s := sparepartID.String()
			spIDStr = &s
		}
		actorName := actor.Name
		createdDTO = ServiceOrderItemDTO{
			ID:             itemID.String(),
			ItemType:       itemType,
			SparepartID:    spIDStr,
			Description:    description,
			Qty:            input.Qty,
			UnitPrice:      unitPrice,
			CostPrice:      costPrice,
			DiscountAmount: input.DiscountAmount,
			Subtotal:       subtotal,
			WarrantyDays:   input.WarrantyDays,
			AddedBy:        &actorName,
			CreatedAt:      now,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &createdDTO, nil
}

// DeleteItem removes a service item line and restocks spareparts if applicable.
func (s *Service) DeleteItem(ctx context.Context, actor Actor, orderID, itemID uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	order, err := s.findOrder(ctx, actor, orderID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check invoice lock: finalized invoices freeze their line items.
		locked, err := invoiceLocked(tx, orderID, tenantID)
		if err != nil {
			return err
		}
		if locked {
			return apperror.Conflict("INVOICE_LOCKED", "Item servis tidak dapat dihapus karena invoice sudah difinalisasi.")
		}

		type itemRow struct {
			ItemType    string     `gorm:"column:item_type"`
			SparepartID *uuid.UUID `gorm:"column:sparepart_id"`
			Qty         float64    `gorm:"column:qty"`
			CostPrice   int64      `gorm:"column:cost_price"`
			Description string     `gorm:"column:description"`
		}
		var it itemRow
		err = tx.Raw(`
			SELECT item_type, sparepart_id, qty, cost_price, description
			FROM service_order_items
			WHERE id = ? AND service_order_id = ? AND tenant_id = ?`,
			itemID, orderID, tenantID).Scan(&it).Error

		if err != nil {
			return apperror.Internal(err)
		}
		if it.ItemType == "" {
			return apperror.NotFound("Item servis tidak ditemukan.")
		}

		// Restock sparepart if needed
		if it.ItemType == "sparepart" && it.SparepartID != nil {
			qtyInt := int(it.Qty)
			if qtyInt > 0 {
				var currentStock int
				if err := tx.Raw(`SELECT stock FROM spareparts WHERE id = ? AND tenant_id = ? FOR UPDATE`,
					*it.SparepartID, tenantID).Scan(&currentStock).Error; err != nil {
					return apperror.Internal(err)
				}
				newStock := currentStock + qtyInt
				if err := tx.Exec(`UPDATE spareparts SET stock = ?, updated_at = ? WHERE id = ? AND tenant_id = ?`,
					newStock, now, *it.SparepartID, tenantID).Error; err != nil {
					return apperror.Internal(err)
				}

				movementID := uuid.New()
				if err := tx.Exec(`
					INSERT INTO inventory_movements (
						id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
						stock_before, stock_after, unit_cost, reference_type, reference_id,
						notes, created_by, created_at
					) VALUES (?, ?, ?, ?, 'service_return', ?, ?, ?, ?, 'service_order', ?, ?, ?, ?)`,
					movementID, tenantID, order.OutletID, *it.SparepartID, qtyInt,
					currentStock, newStock, it.CostPrice, orderID,
					"Pengembalian stok dari pembatalan item "+it.Description, actorID(actor), now).Error; err != nil {
					return apperror.Internal(err)
				}
			}
		}

		// Delete item
		if err := tx.Exec(`DELETE FROM service_order_items WHERE id = ? AND tenant_id = ?`, itemID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		// Recalculate invoice total.
		if _, err := recalcOrderTotals(tx, tenantID, orderID, now); err != nil {
			return err
		}

		// Audit log
		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "service_item_delete",
			Module:     "service",
			EntityType: "service_order",
			EntityID:   &orderID,
			After: map[string]any{
				"item_id":     itemID.String(),
				"description": it.Description,
			},
		})
	})
}

// UpdateItem edits an existing line item and adjusts sparepart stock when the
// quantity changes.
func (s *Service) UpdateItem(ctx context.Context, actor Actor, orderID, itemID uuid.UUID, input UpdateServiceItemRequest) (*ServiceOrderItemDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	order, err := s.findOrder(ctx, actor, orderID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := invoiceLocked(tx, orderID, tenantID)
		if err != nil {
			return err
		}
		if locked {
			return apperror.Conflict("INVOICE_LOCKED", "Item servis tidak dapat diubah karena invoice sudah difinalisasi.")
		}

		type itemRow struct {
			ID             uuid.UUID  `gorm:"column:id"`
			ItemType       string     `gorm:"column:item_type"`
			SparepartID    *uuid.UUID `gorm:"column:sparepart_id"`
			Description    string     `gorm:"column:description"`
			Qty            float64    `gorm:"column:qty"`
			UnitPrice      int64      `gorm:"column:unit_price"`
			CostPrice      int64      `gorm:"column:cost_price"`
			DiscountAmount int64      `gorm:"column:discount_amount"`
			WarrantyDays   int        `gorm:"column:warranty_days"`
		}
		var it itemRow
		if err := tx.Raw(`
			SELECT id, item_type, sparepart_id, description, qty, unit_price,
			       cost_price, discount_amount, warranty_days
			FROM service_order_items
			WHERE id = ? AND service_order_id = ? AND tenant_id = ?
			FOR UPDATE`, itemID, orderID, tenantID).Scan(&it).Error; err != nil {
			return apperror.Internal(err)
		}
		if it.ID == uuid.Nil {
			return apperror.NotFound("Item servis tidak ditemukan.")
		}

		description := it.Description
		if input.Description != nil {
			description = strings.TrimSpace(*input.Description)
			if description == "" || len(description) > 200 {
				return apperror.Validation(map[string]string{"description": "Deskripsi wajib diisi, maksimal 200 karakter."})
			}
		}
		qty := it.Qty
		if input.Qty != nil {
			qty = *input.Qty
		}
		if qty <= 0 || qty > 9999 {
			return apperror.Validation(map[string]string{"qty": "Jumlah item harus lebih dari 0 dan maksimal 9999."})
		}
		if it.ItemType == "sparepart" && math.Trunc(qty) != qty {
			return apperror.Validation(map[string]string{"qty": "Jumlah sparepart harus bilangan bulat."})
		}
		unitPrice := it.UnitPrice
		if input.UnitPrice != nil {
			unitPrice = *input.UnitPrice
		}
		discount := it.DiscountAmount
		if input.DiscountAmount != nil {
			discount = *input.DiscountAmount
		}
		warranty := it.WarrantyDays
		if input.WarrantyDays != nil {
			warranty = *input.WarrantyDays
		}

		gross := int64(qty * float64(unitPrice))
		if discount > gross {
			return apperror.Validation(map[string]string{"discount_amount": "Diskon baris tidak boleh melebihi harga item."})
		}
		subtotal := gross - discount

		// Adjust stock by the quantity delta so an edit never double-deducts.
		if it.ItemType == "sparepart" && it.SparepartID != nil {
			delta := int(qty) - int(it.Qty)
			if delta != 0 {
				type sparepartRow struct {
					ID    uuid.UUID `gorm:"column:id"`
					Stock int       `gorm:"column:stock"`
				}
				var sp sparepartRow
				if err := tx.Raw(`
					SELECT id, stock FROM spareparts
					WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL
					FOR UPDATE`, *it.SparepartID, tenantID).Scan(&sp).Error; err != nil {
					return apperror.Internal(err)
				}
				if sp.ID == uuid.Nil {
					return apperror.NotFound("Sparepart tidak ditemukan.")
				}
				newStock := sp.Stock - delta
				if newStock < 0 {
					return apperror.Conflict("INSUFFICIENT_STOCK",
						fmt.Sprintf("Stok sparepart tidak mencukupi (sisa: %d, tambahan: %d).", sp.Stock, delta))
				}
				if err := tx.Exec(`
					UPDATE spareparts SET stock = ?, last_stock_at = ?, updated_at = ?
					WHERE id = ? AND tenant_id = ?`,
					newStock, now, now, *it.SparepartID, tenantID).Error; err != nil {
					return apperror.Internal(err)
				}

				movementType := "service_out"
				movementQty := -delta
				notes := "Penyesuaian pemakaian sparepart untuk tiket servis " + order.TicketNo
				if delta < 0 {
					movementType = "service_return"
					movementQty = -delta
					notes = "Pengembalian sebagian stok dari perubahan item tiket " + order.TicketNo
				}
				movementID := uuid.New()
				if err := tx.Exec(`
					INSERT INTO inventory_movements (
						id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
						stock_before, stock_after, unit_cost, reference_type, reference_id,
						notes, created_by, created_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'service_order', ?, ?, ?, ?)`,
					movementID, tenantID, order.OutletID, *it.SparepartID, movementType, movementQty,
					sp.Stock, newStock, it.CostPrice, orderID, notes, actorID(actor), now).Error; err != nil {
					return apperror.Internal(err)
				}
			}
		}

		if err := tx.Exec(`
			UPDATE service_order_items
			SET description = ?, qty = ?, unit_price = ?, discount_amount = ?,
			    subtotal = ?, warranty_days = ?, updated_at = ?
			WHERE id = ? AND service_order_id = ? AND tenant_id = ?`,
			description, qty, unitPrice, discount, subtotal, warranty, now,
			itemID, orderID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		if _, err := recalcOrderTotals(tx, tenantID, orderID, now); err != nil {
			return err
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "service_item_update",
			Module:     "service",
			EntityType: "service_order",
			EntityID:   &orderID,
			Before: map[string]any{
				"description": it.Description, "qty": it.Qty,
				"unit_price": it.UnitPrice, "discount_amount": it.DiscountAmount,
				"warranty_days": it.WarrantyDays,
			},
			After: map[string]any{
				"item_id": itemID.String(), "description": description, "qty": qty,
				"unit_price": unitPrice, "discount_amount": discount,
				"warranty_days": warranty, "subtotal": subtotal,
			},
		})
	})
	if err != nil {
		return nil, err
	}

	items, err := s.itemsForOrder(ctx, tenantID, orderID, actor.HasPermission("inventory.cost.view"))
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == itemID.String() {
			return &items[i], nil
		}
	}
	return nil, apperror.NotFound("Item servis tidak ditemukan.")
}

// SearchSpareparts finds active spareparts for line items.
func (s *Service) SearchSpareparts(ctx context.Context, actor Actor, q string, limit int) ([]SparepartOption, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	type spSearchRow struct {
		ID        uuid.UUID `gorm:"column:id"`
		SKU       string    `gorm:"column:sku"`
		Name      string    `gorm:"column:name"`
		Stock     int       `gorm:"column:stock"`
		CostPrice int64     `gorm:"column:cost_price"`
		SellPrice int64     `gorm:"column:sell_price"`
		Location  *string   `gorm:"column:location"`
	}

	var rows []spSearchRow
	query := `
		SELECT id, sku, name, stock, cost_price, sell_price, location
		FROM spareparts
		WHERE tenant_id = ? AND is_active = true AND deleted_at IS NULL`
	args := []any{tenantID}

	if trimmed := strings.TrimSpace(q); trimmed != "" {
		like := "%" + trimmed + "%"
		query += ` AND (name ILIKE ? OR sku ILIKE ? OR compatible_models ILIKE ?)`
		args = append(args, like, like, like)
	}

	query += ` ORDER BY name ASC LIMIT ?`
	args = append(args, limit)

	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	result := make([]SparepartOption, len(rows))
	canViewCost := actor.HasPermission("inventory.cost.view")
	for i, r := range rows {
		loc := ""
		if r.Location != nil {
			loc = *r.Location
		}
		cost := int64(0)
		if canViewCost {
			cost = r.CostPrice
		}
		result[i] = SparepartOption{
			ID:        r.ID.String(),
			SKU:       r.SKU,
			Name:      r.Name,
			Stock:     r.Stock,
			CostPrice: cost,
			SellPrice: r.SellPrice,
			Location:  loc,
		}
	}
	return result, nil
}

func (s *Service) itemsForOrder(ctx context.Context, tenantID, orderID uuid.UUID, canViewCost bool) ([]ServiceOrderItemDTO, error) {
	type itemRow struct {
		ID             uuid.UUID  `gorm:"column:id"`
		ItemType       string     `gorm:"column:item_type"`
		SparepartID    *uuid.UUID `gorm:"column:sparepart_id"`
		Description    string     `gorm:"column:description"`
		Qty            float64    `gorm:"column:qty"`
		UnitPrice      int64      `gorm:"column:unit_price"`
		CostPrice      int64      `gorm:"column:cost_price"`
		DiscountAmount int64      `gorm:"column:discount_amount"`
		Subtotal       int64      `gorm:"column:subtotal"`
		WarrantyDays   int        `gorm:"column:warranty_days"`
		AddedByName    *string    `gorm:"column:added_by_name"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
	}

	var rows []itemRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT i.id, i.item_type, i.sparepart_id, i.description, i.qty,
		       i.unit_price, i.cost_price, i.discount_amount, i.subtotal,
		       i.warranty_days, u.name AS added_by_name, i.created_at
		FROM service_order_items i
		LEFT JOIN users u ON u.id = i.added_by
		WHERE i.service_order_id = ? AND i.tenant_id = ?
		ORDER BY i.created_at ASC`, orderID, tenantID).Scan(&rows).Error

	if err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]ServiceOrderItemDTO, len(rows))
	for i, r := range rows {
		var spStr *string
		if r.SparepartID != nil {
			s := r.SparepartID.String()
			spStr = &s
		}
		costPrice := int64(0)
		if canViewCost {
			costPrice = r.CostPrice
		}
		items[i] = ServiceOrderItemDTO{
			ID:             r.ID.String(),
			ItemType:       r.ItemType,
			SparepartID:    spStr,
			Description:    r.Description,
			Qty:            r.Qty,
			UnitPrice:      r.UnitPrice,
			CostPrice:      costPrice,
			DiscountAmount: r.DiscountAmount,
			Subtotal:       r.Subtotal,
			WarrantyDays:   r.WarrantyDays,
			AddedBy:        r.AddedByName,
			CreatedAt:      r.CreatedAt,
		}
	}
	return items, nil
}

// invoiceLocked reports whether the order invoice has left draft state.
func invoiceLocked(tx *gorm.DB, orderID, tenantID uuid.UUID) (bool, error) {
	var invoice struct {
		PaidAmount int64  `gorm:"column:paid_amount"`
		Status     string `gorm:"column:status"`
	}
	if err := tx.Raw(`
		SELECT paid_amount, status FROM invoices
		WHERE service_order_id = ? AND tenant_id = ?`, orderID, tenantID).
		Scan(&invoice).Error; err != nil {
		return false, apperror.Internal(err)
	}
	if invoice.Status == "" {
		return false, nil
	}
	return invoice.Status != "draft" || invoice.PaidAmount > 0, nil
}

// recalcOrderTotals rebuilds draft invoice subtotals and the order estimate
// from the persisted line items. remaining_amount is a generated column.
func recalcOrderTotals(tx *gorm.DB, tenantID, orderID uuid.UUID, now time.Time) (int64, error) {
	var sums struct {
		Service int64 `gorm:"column:service_total"`
		Part    int64 `gorm:"column:part_total"`
	}
	if err := tx.Raw(`
		SELECT
			COALESCE(SUM(subtotal) FILTER (WHERE item_type IN ('service', 'custom')), 0) AS service_total,
			COALESCE(SUM(subtotal) FILTER (WHERE item_type = 'sparepart'), 0) AS part_total
		FROM service_order_items
		WHERE service_order_id = ? AND tenant_id = ?`, orderID, tenantID).
		Scan(&sums).Error; err != nil {
		return 0, apperror.Internal(err)
	}

	total := sums.Service + sums.Part
	if err := tx.Exec(`
		UPDATE invoices
		SET subtotal_service = ?, subtotal_part = ?, total_amount = ?, updated_at = ?
		WHERE service_order_id = ? AND tenant_id = ? AND status = 'draft'`,
		sums.Service, sums.Part, total, now, orderID, tenantID).Error; err != nil {
		return 0, apperror.Internal(err)
	}
	if err := tx.Exec(`
		UPDATE service_orders SET estimated_cost = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?`,
		total, now, orderID, tenantID).Error; err != nil {
		return 0, apperror.Internal(err)
	}
	return total, nil
}
