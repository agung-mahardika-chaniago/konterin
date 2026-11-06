package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var jakartaLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}()

func parseTimestamp(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, err
	}
	utc := value.UTC()
	return &utc, nil
}

func parseDateBoundary(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		utc := value.UTC()
		return &utc, nil
	}
	value, err := time.ParseInLocation("2006-01-02", raw, jakartaLocation)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		value = value.AddDate(0, 0, 1)
	}
	utc := value.UTC()
	return &utc, nil
}

func outletScope(actor Actor, alias string, args []any) (string, []any, bool) {
	if actor.AllOutletAccess {
		return "", args, true
	}
	if len(actor.OutletIDs) == 0 {
		return "", args, false
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(actor.OutletIDs)), ",")
	clause := " AND " + alias + ".outlet_id IN (" + placeholders + ")"
	for _, id := range actor.OutletIDs {
		args = append(args, id)
	}
	return clause, args, true
}

type movementRow struct {
	ID            uuid.UUID  `gorm:"column:id"`
	SparepartID   uuid.UUID  `gorm:"column:sparepart_id"`
	SparepartName string     `gorm:"column:sparepart_name"`
	SKU           string     `gorm:"column:sku"`
	OutletID      uuid.UUID  `gorm:"column:outlet_id"`
	OutletName    string     `gorm:"column:outlet_name"`
	MovementType  string     `gorm:"column:movement_type"`
	Qty           int        `gorm:"column:qty"`
	StockBefore   int        `gorm:"column:stock_before"`
	StockAfter    int        `gorm:"column:stock_after"`
	UnitCost      int64      `gorm:"column:unit_cost"`
	ReferenceType string     `gorm:"column:reference_type"`
	ReferenceID   *uuid.UUID `gorm:"column:reference_id"`
	Notes         *string    `gorm:"column:notes"`
	CreatedBy     *string    `gorm:"column:created_by"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func movementDTO(row movementRow, canViewCost bool) Movement {
	dto := Movement{
		ID:            row.ID.String(),
		SparepartID:   row.SparepartID.String(),
		SparepartName: row.SparepartName,
		SKU:           row.SKU,
		OutletID:      row.OutletID.String(),
		OutletName:    row.OutletName,
		MovementType:  row.MovementType,
		Qty:           row.Qty,
		StockBefore:   row.StockBefore,
		StockAfter:    row.StockAfter,
		ReferenceType: row.ReferenceType,
		ReferenceID:   optionalString(row.ReferenceID),
		Notes:         row.Notes,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt,
	}
	if canViewCost {
		dto.UnitCost = row.UnitCost
	}
	return dto
}

// ListMovements returns paginated stock movement history.
func (s *Service) ListMovements(ctx context.Context, actor Actor, filter MovementFilter) (*MovementList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}

	where := ` WHERE im.tenant_id = ?`
	args := []any{tenantID}

	if filter.SparepartID != nil {
		where += ` AND im.sparepart_id = ?`
		args = append(args, *filter.SparepartID)
	}
	if mt := strings.TrimSpace(filter.MovementType); mt != "" {
		where += ` AND im.movement_type = ?`
		args = append(args, mt)
	}
	if filter.OutletID != nil {
		if !actor.AllOutletAccess && !actor.HasOutlet(*filter.OutletID) {
			return nil, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		where += ` AND im.outlet_id = ?`
		args = append(args, *filter.OutletID)
	} else {
		clause, scopedArgs, ok := outletScope(actor, "im", args)
		if !ok {
			return &MovementList{Items: []Movement{}, Total: 0}, nil
		}
		where += clause
		args = scopedArgs
	}

	from, err := parseDateBoundary(filter.FromDate, false)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"from": "Format tanggal awal harus YYYY-MM-DD."})
	}
	to, err := parseDateBoundary(filter.ToDate, true)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"to": "Format tanggal akhir harus YYYY-MM-DD."})
	}
	if from != nil {
		where += ` AND im.created_at >= ?`
		args = append(args, *from)
	}
	if to != nil {
		where += ` AND im.created_at < ?`
		args = append(args, *to)
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM inventory_movements im`+where, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	page, perPage := filter.Page, filter.PerPage
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	var rows []movementRow
	query := `
		SELECT im.id, im.sparepart_id, sp.name AS sparepart_name, sp.sku,
		       im.outlet_id, o.name AS outlet_name, im.movement_type, im.qty,
		       im.stock_before, im.stock_after, im.unit_cost, im.reference_type,
		       im.reference_id, im.notes, u.name AS created_by, im.created_at
		FROM inventory_movements im
		JOIN spareparts sp ON sp.id = im.sparepart_id
		JOIN outlets o ON o.id = im.outlet_id
		LEFT JOIN users u ON u.id = im.created_by` + where + `
		ORDER BY im.created_at DESC, im.id DESC
		LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	if err := s.db.WithContext(ctx).Raw(query, listArgs...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	canViewCost := actor.HasPermission("inventory.cost.view")
	items := make([]Movement, len(rows))
	for i, row := range rows {
		items[i] = movementDTO(row, canViewCost)
	}
	return &MovementList{Items: items, Total: total}, nil
}

// Purchase records received stock and updates the latest cost price.
func (s *Service) Purchase(ctx context.Context, actor Actor, input PurchaseRequest) (*PurchaseResult, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if len(input.Items) == 0 {
		return nil, apperror.Validation(map[string]string{"items": "Minimal satu item pembelian."})
	}
	outletID, err := actor.outletFor(nil)
	if err != nil {
		return nil, err
	}
	supplierID, err := parseOptionalUUID(input.SupplierID, "supplier_id", "ID supplier tidak valid.")
	if err != nil {
		return nil, err
	}
	purchasedAt, err := parseTimestamp(input.PurchasedAt)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"purchased_at": "Format tanggal pembelian harus RFC3339."})
	}
	now := time.Now().UTC()
	if purchasedAt == nil {
		purchasedAt = &now
	}

	type line struct {
		id       uuid.UUID
		qty      int
		unitCost int64
	}
	lines := make([]line, len(input.Items))
	seen := make(map[uuid.UUID]struct{}, len(input.Items))
	for i, item := range input.Items {
		id, parseErr := uuid.Parse(strings.TrimSpace(item.SparepartID))
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{
				fmt.Sprintf("items.%d.sparepart_id", i): "ID sparepart tidak valid.",
			})
		}
		if item.Qty <= 0 {
			return nil, apperror.Validation(map[string]string{
				fmt.Sprintf("items.%d.qty", i): "Jumlah harus lebih dari 0.",
			})
		}
		if item.UnitCost < 0 {
			return nil, apperror.Validation(map[string]string{
				fmt.Sprintf("items.%d.unit_cost", i): "Harga satuan tidak boleh negatif.",
			})
		}
		if _, exists := seen[id]; exists {
			return nil, apperror.Validation(map[string]string{
				fmt.Sprintf("items.%d.sparepart_id", i): "Sparepart duplikat dalam satu pembelian.",
			})
		}
		seen[id] = struct{}{}
		lines[i] = line{id: id, qty: item.Qty, unitCost: item.UnitCost}
	}

	referenceID := uuid.New()
	canViewCost := actor.HasPermission("inventory.cost.view")
	result := &PurchaseResult{ReferenceID: referenceID.String(), PurchasedAt: *purchasedAt, Items: []Movement{}}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		outletName, err := s.outletName(ctx, tx, tenantID, outletID)
		if err != nil {
			return err
		}
		if err := s.validateSupplier(ctx, tx, tenantID, supplierID); err != nil {
			return err
		}

		ids := make([]uuid.UUID, len(lines))
		for i, line := range lines {
			ids[i] = line.id
		}
		type stockRow struct {
			ID        uuid.UUID `gorm:"column:id"`
			Name      string    `gorm:"column:name"`
			SKU       string    `gorm:"column:sku"`
			Stock     int       `gorm:"column:stock"`
			CostPrice int64     `gorm:"column:cost_price"`
			IsActive  bool      `gorm:"column:is_active"`
		}
		var rows []stockRow
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, name, sku, stock, cost_price, is_active
			FROM spareparts
			WHERE tenant_id = ? AND id IN ? AND deleted_at IS NULL
			ORDER BY id
			FOR UPDATE`, tenantID, ids).Scan(&rows).Error; err != nil {
			return apperror.Internal(err)
		}
		byID := make(map[uuid.UUID]stockRow, len(rows))
		for _, row := range rows {
			byID[row.ID] = row
		}

		referenceNo := trimOptional(input.ReferenceNo)
		notes := trimOptional(input.Notes)
		var totalCost int64

		for _, line := range lines {
			row, exists := byID[line.id]
			if !exists {
				return apperror.NotFound("Sparepart tidak ditemukan.")
			}
			if !row.IsActive {
				return apperror.Conflict("SPAREPART_INACTIVE", "Sparepart "+row.Name+" tidak aktif.")
			}
			newStock := row.Stock + line.qty
			if err := tx.Exec(`
				UPDATE spareparts SET stock = ?, cost_price = ?, last_stock_at = ?, updated_at = ?
				WHERE id = ? AND tenant_id = ?`,
				newStock, line.unitCost, *purchasedAt, now, line.id, tenantID).Error; err != nil {
				return apperror.Internal(err)
			}

			movementID := uuid.New()
			movementNotes := "Pembelian stok"
			if referenceNo != nil {
				movementNotes += " " + *referenceNo
			}
			if notes != nil {
				movementNotes += " - " + *notes
			}
			if err := tx.Exec(`
				INSERT INTO inventory_movements (
					id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
					stock_before, stock_after, unit_cost, reference_type, reference_id,
					notes, created_by, created_at
				) VALUES (?, ?, ?, ?, 'purchase_in', ?, ?, ?, ?, 'purchase', ?, ?, ?, ?)`,
				movementID, tenantID, outletID, line.id, line.qty, row.Stock, newStock,
				line.unitCost, referenceID, movementNotes, actorID(actor), *purchasedAt).Error; err != nil {
				return apperror.Internal(err)
			}

			totalCost += int64(line.qty) * line.unitCost
			result.Items = append(result.Items, movementDTO(movementRow{
				ID:            movementID,
				SparepartID:   line.id,
				SparepartName: row.Name,
				SKU:           row.SKU,
				OutletID:      outletID,
				OutletName:    outletName,
				MovementType:  "purchase_in",
				Qty:           line.qty,
				StockBefore:   row.Stock,
				StockAfter:    newStock,
				UnitCost:      line.unitCost,
				ReferenceType: "purchase",
				ReferenceID:   &referenceID,
				Notes:         &movementNotes,
				CreatedBy:     &actor.Name,
				CreatedAt:     *purchasedAt,
			}, canViewCost))
		}
		if canViewCost {
			result.TotalCost = totalCost
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "inventory_purchase",
			Module:     "inventory",
			EntityType: "purchase",
			EntityID:   &referenceID,
			After: map[string]any{
				"reference_no": referenceNo, "items": len(lines),
				"total_cost": totalCost, "outlet_id": outletID.String(),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// AdjustStock sets physical stock for one part and writes an opname movement.
func (s *Service) AdjustStock(ctx context.Context, actor Actor, input AdjustmentRequest) (*Movement, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(strings.TrimSpace(input.SparepartID))
	if err != nil {
		return nil, apperror.Validation(map[string]string{"sparepart_id": "ID sparepart tidak valid."})
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, apperror.Validation(map[string]string{"reason": "Alasan penyesuaian wajib diisi."})
	}
	outletID, err := actor.outletFor(nil)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	var dto Movement
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		outletName, err := s.outletName(ctx, tx, tenantID, outletID)
		if err != nil {
			return err
		}
		var row struct {
			ID        uuid.UUID `gorm:"column:id"`
			Name      string    `gorm:"column:name"`
			SKU       string    `gorm:"column:sku"`
			Stock     int       `gorm:"column:stock"`
			CostPrice int64     `gorm:"column:cost_price"`
		}
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, name, sku, stock, cost_price FROM spareparts
			WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL
			FOR UPDATE`, id, tenantID).Scan(&row).Error; err != nil {
			return apperror.Internal(err)
		}
		if row.ID == uuid.Nil {
			return apperror.NotFound("Sparepart tidak ditemukan.")
		}
		delta := input.NewStock - row.Stock
		if delta == 0 {
			return apperror.Conflict("STOCK_UNCHANGED", "Stok baru sama dengan stok saat ini.")
		}

		if err := tx.Exec(`
			UPDATE spareparts SET stock = ?, last_stock_at = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`,
			input.NewStock, now, now, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		movementType := "adjustment_in"
		if delta < 0 {
			movementType = "adjustment_out"
		}
		referenceID := uuid.New()
		notes := reason
		if extra := trimOptional(input.Notes); extra != nil {
			notes += " - " + *extra
		}
		movementID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO inventory_movements (
				id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
				stock_before, stock_after, unit_cost, reference_type, reference_id,
				notes, created_by, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'opname', ?, ?, ?, ?)`,
			movementID, tenantID, outletID, id, movementType, delta, row.Stock,
			input.NewStock, row.CostPrice, referenceID, notes, actorID(actor), now).Error; err != nil {
			return apperror.Internal(err)
		}

		dto = movementDTO(movementRow{
			ID:            movementID,
			SparepartID:   id,
			SparepartName: row.Name,
			SKU:           row.SKU,
			OutletID:      outletID,
			OutletName:    outletName,
			MovementType:  movementType,
			Qty:           delta,
			StockBefore:   row.Stock,
			StockAfter:    input.NewStock,
			UnitCost:      row.CostPrice,
			ReferenceType: "opname",
			ReferenceID:   &referenceID,
			Notes:         &notes,
			CreatedBy:     &actor.Name,
			CreatedAt:     now,
		}, actor.HasPermission("inventory.cost.view"))

		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "inventory_adjustment",
			Module:     "inventory",
			EntityType: "sparepart",
			EntityID:   &id,
			Before:     map[string]any{"stock": row.Stock},
			After: map[string]any{
				"stock": input.NewStock, "delta": delta, "reason": reason,
				"outlet_id": outletID.String(),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

// Alerts lists active parts at or below their minimum stock.
func (s *Service) Alerts(ctx context.Context, actor Actor) ([]StockAlert, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	type row struct {
		ID           uuid.UUID `gorm:"column:id"`
		SKU          string    `gorm:"column:sku"`
		Name         string    `gorm:"column:name"`
		CategoryName *string   `gorm:"column:category_name"`
		Stock        int       `gorm:"column:stock"`
		MinStock     int       `gorm:"column:min_stock"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT sp.id, sp.sku, sp.name, c.name AS category_name, sp.stock, sp.min_stock
		FROM spareparts sp
		LEFT JOIN sparepart_categories c ON c.id = sp.category_id AND c.tenant_id = sp.tenant_id
		WHERE sp.tenant_id = ? AND sp.deleted_at IS NULL AND sp.is_active = true
		  AND sp.stock <= sp.min_stock
		ORDER BY (sp.min_stock - sp.stock) DESC, sp.name ASC`, tenantID).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	alerts := make([]StockAlert, len(rows))
	for i, row := range rows {
		alerts[i] = StockAlert{
			ID: row.ID.String(), SKU: row.SKU, Name: row.Name,
			CategoryName: row.CategoryName, Stock: row.Stock, MinStock: row.MinStock,
			Shortage: row.MinStock - row.Stock,
		}
	}
	return alerts, nil
}

// Valuation summarises active inventory at cost.
func (s *Service) Valuation(ctx context.Context, actor Actor) (*Valuation, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var out struct {
		TotalItems int64 `gorm:"column:total_items"`
		TotalStock int64 `gorm:"column:total_stock"`
		TotalValue int64 `gorm:"column:total_value"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*) AS total_items,
		       COALESCE(SUM(stock), 0) AS total_stock,
		       COALESCE(SUM(stock::bigint * cost_price), 0) AS total_value
		FROM spareparts
		WHERE tenant_id = ? AND deleted_at IS NULL AND is_active = true`, tenantID).Scan(&out).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return &Valuation{
		TotalItems: out.TotalItems, TotalStock: out.TotalStock, TotalValue: out.TotalValue,
	}, nil
}

// TopUsed ranks parts by service usage in a date range.
func (s *Service) TopUsed(ctx context.Context, actor Actor, fromRaw, toRaw string, limit int) ([]TopUsed, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	now := time.Now().UTC()
	from, err := parseDateBoundary(fromRaw, false)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"from": "Format tanggal awal harus YYYY-MM-DD."})
	}
	if from == nil {
		defaultFrom := now.AddDate(0, 0, -30)
		from = &defaultFrom
	}
	to, err := parseDateBoundary(toRaw, true)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"to": "Format tanggal akhir harus YYYY-MM-DD."})
	}
	if to == nil {
		to = &now
	}

	where := ` WHERE im.tenant_id = ? AND im.movement_type = 'service_out'
		AND im.created_at >= ? AND im.created_at < ?`
	args := []any{tenantID, *from, *to}
	clause, scopedArgs, ok := outletScope(actor, "im", args)
	if !ok {
		return []TopUsed{}, nil
	}
	where += clause
	args = scopedArgs
	args = append(args, limit)

	type row struct {
		SparepartID   uuid.UUID `gorm:"column:sparepart_id"`
		SKU           string    `gorm:"column:sku"`
		Name          string    `gorm:"column:name"`
		UsedQty       int64     `gorm:"column:used_qty"`
		MovementCount int64     `gorm:"column:movement_count"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT sp.id AS sparepart_id, sp.sku, sp.name,
		       COALESCE(SUM(-im.qty), 0) AS used_qty,
		       count(*) AS movement_count
		FROM inventory_movements im
		JOIN spareparts sp ON sp.id = im.sparepart_id
		`+where+`
		GROUP BY sp.id, sp.sku, sp.name
		ORDER BY used_qty DESC, sp.name ASC
		LIMIT ?`, args...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]TopUsed, len(rows))
	for i, row := range rows {
		out[i] = TopUsed{
			SparepartID: row.SparepartID.String(), SKU: row.SKU, Name: row.Name,
			UsedQty: row.UsedQty, MovementCount: row.MovementCount,
		}
	}
	return out, nil
}
