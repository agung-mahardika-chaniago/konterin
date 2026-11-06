package inventory

import (
	"context"
	"errors"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Service implements sparepart master data and inventory operations.
type Service struct {
	db *gorm.DB
}

// NewService builds the inventory module.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func actorTenant(actor Actor) (uuid.UUID, error) {
	if actor.TenantID == uuid.Nil {
		return uuid.Nil, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return actor.TenantID, nil
}

func actorID(actor Actor) *uuid.UUID {
	id := actor.UserID
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func (a Actor) outletFor(requested *uuid.UUID) (uuid.UUID, error) {
	if requested != nil {
		if !a.HasOutlet(*requested) {
			return uuid.Nil, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		return *requested, nil
	}
	if a.ActiveOutletID == nil {
		return uuid.Nil, apperror.Validation(map[string]string{"outlet_id": "Pilih outlet terlebih dahulu."})
	}
	return *a.ActiveOutletID, nil
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}

func parseOptionalUUID(raw *string, field, message string) (*uuid.UUID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return nil, apperror.Validation(map[string]string{field: message})
	}
	return &id, nil
}

type sparepartRow struct {
	ID               uuid.UUID  `gorm:"column:id"`
	SKU              string     `gorm:"column:sku"`
	Barcode          *string    `gorm:"column:barcode"`
	Name             string     `gorm:"column:name"`
	CategoryID       *uuid.UUID `gorm:"column:category_id"`
	CategoryName     *string    `gorm:"column:category_name"`
	SupplierID       *uuid.UUID `gorm:"column:supplier_id"`
	SupplierName     *string    `gorm:"column:supplier_name"`
	CompatibleModels *string    `gorm:"column:compatible_models"`
	Stock            int        `gorm:"column:stock"`
	MinStock         int        `gorm:"column:min_stock"`
	CostPrice        int64      `gorm:"column:cost_price"`
	SellPrice        int64      `gorm:"column:sell_price"`
	Location         *string    `gorm:"column:location"`
	IsActive         bool       `gorm:"column:is_active"`
	LastStockAt      *time.Time `gorm:"column:last_stock_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

const sparepartSelect = `
	SELECT sp.id, sp.sku, sp.barcode, sp.name, sp.category_id, c.name AS category_name,
	       sp.supplier_id, s.name AS supplier_name, sp.compatible_models,
	       sp.stock, sp.min_stock, sp.cost_price, sp.sell_price, sp.location,
	       sp.is_active, sp.last_stock_at, sp.created_at, sp.updated_at
	FROM spareparts sp
	LEFT JOIN sparepart_categories c ON c.id = sp.category_id AND c.tenant_id = sp.tenant_id
	LEFT JOIN suppliers s ON s.id = sp.supplier_id AND s.tenant_id = sp.tenant_id`

func (s *Service) sparepartDTO(row sparepartRow, canViewCost bool) Sparepart {
	sp := Sparepart{
		ID:               row.ID.String(),
		SKU:              row.SKU,
		Barcode:          row.Barcode,
		Name:             row.Name,
		CompatibleModels: row.CompatibleModels,
		Stock:            row.Stock,
		MinStock:         row.MinStock,
		SellPrice:        row.SellPrice,
		Location:         row.Location,
		IsActive:         row.IsActive,
		LowStock:         row.Stock <= row.MinStock,
		LastStockAt:      row.LastStockAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if row.CategoryID != nil {
		id := row.CategoryID.String()
		sp.CategoryID = &id
	}
	sp.CategoryName = row.CategoryName
	if row.SupplierID != nil {
		id := row.SupplierID.String()
		sp.SupplierID = &id
	}
	sp.SupplierName = row.SupplierName
	if canViewCost {
		sp.CostPrice = row.CostPrice
		sp.StockValue = int64(row.Stock) * row.CostPrice
	}
	return sp
}

// ListSpareparts returns a paginated master-data list.
func (s *Service) ListSpareparts(ctx context.Context, actor Actor, filter SparepartFilter) (*SparepartList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	where := ` WHERE sp.tenant_id = ? AND sp.deleted_at IS NULL`
	args := []any{tenantID}

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		where += ` AND (sp.name ILIKE ? OR sp.sku ILIKE ? OR sp.barcode ILIKE ? OR sp.compatible_models ILIKE ?)`
		args = append(args, like, like, like, like)
	}
	if filter.CategoryID != nil {
		where += ` AND sp.category_id = ?`
		args = append(args, *filter.CategoryID)
	}
	if filter.SupplierID != nil {
		where += ` AND sp.supplier_id = ?`
		args = append(args, *filter.SupplierID)
	}
	if filter.LowStock {
		where += ` AND sp.stock <= sp.min_stock`
	}
	if filter.Active != nil {
		where += ` AND sp.is_active = ?`
		args = append(args, *filter.Active)
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM spareparts sp`+where, args...).Scan(&total).Error; err != nil {
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

	var rows []sparepartRow
	query := sparepartSelect + where + ` ORDER BY sp.name ASC, sp.id ASC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	if err := s.db.WithContext(ctx).Raw(query, listArgs...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	canViewCost := actor.HasPermission("inventory.cost.view")
	items := make([]Sparepart, len(rows))
	for i, row := range rows {
		items[i] = s.sparepartDTO(row, canViewCost)
	}
	return &SparepartList{Items: items, Total: total}, nil
}

// GetSparepart returns one part by ID.
func (s *Service) GetSparepart(ctx context.Context, actor Actor, id uuid.UUID) (*Sparepart, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	return s.getSparepart(ctx, tenantID, id, actor.HasPermission("inventory.cost.view"))
}

func (s *Service) getSparepart(ctx context.Context, tenantID, id uuid.UUID, canViewCost bool) (*Sparepart, error) {
	var row sparepartRow
	err := s.db.WithContext(ctx).Raw(sparepartSelect+`
		WHERE sp.id = ? AND sp.tenant_id = ? AND sp.deleted_at IS NULL`, id, tenantID).Scan(&row).Error
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Sparepart tidak ditemukan.")
	}
	dto := s.sparepartDTO(row, canViewCost)
	return &dto, nil
}

// GetSparepartByBarcode resolves a barcode (or SKU fallback) for scanners.
func (s *Service) GetSparepartByBarcode(ctx context.Context, actor Actor, code string) (*Sparepart, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, apperror.Validation(map[string]string{"code": "Barcode wajib diisi."})
	}
	var row sparepartRow
	err = s.db.WithContext(ctx).Raw(sparepartSelect+`
		WHERE sp.tenant_id = ? AND sp.deleted_at IS NULL
		  AND (sp.barcode = ? OR sp.sku = ?)
		ORDER BY (sp.barcode = ?) DESC
		LIMIT 1`, tenantID, code, code, code).Scan(&row).Error
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Sparepart dengan barcode tersebut tidak ditemukan.")
	}
	dto := s.sparepartDTO(row, actor.HasPermission("inventory.cost.view"))
	return &dto, nil
}

func (s *Service) validateCategory(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	var exists bool
	if err := tx.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM sparepart_categories WHERE id = ? AND tenant_id = ?)`,
		*id, tenantID).Scan(&exists).Error; err != nil {
		return apperror.Internal(err)
	}
	if !exists {
		return apperror.Validation(map[string]string{"category_id": "Kategori tidak ditemukan."})
	}
	return nil
}

func (s *Service) validateSupplier(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	var exists bool
	if err := tx.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM suppliers
		WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL)`,
		*id, tenantID).Scan(&exists).Error; err != nil {
		return apperror.Internal(err)
	}
	if !exists {
		return apperror.Validation(map[string]string{"supplier_id": "Supplier tidak ditemukan."})
	}
	return nil
}

// CreateSparepart inserts a master part and an opening-stock movement.
func (s *Service) CreateSparepart(ctx context.Context, actor Actor, input CreateSparepartRequest) (*Sparepart, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	sku := strings.TrimSpace(input.SKU)
	name := strings.TrimSpace(input.Name)
	if sku == "" {
		return nil, apperror.Validation(map[string]string{"sku": "SKU wajib diisi."})
	}
	if name == "" {
		return nil, apperror.Validation(map[string]string{"name": "Nama sparepart wajib diisi."})
	}
	barcode := trimOptional(input.Barcode)
	categoryID, err := parseOptionalUUID(input.CategoryID, "category_id", "ID kategori tidak valid.")
	if err != nil {
		return nil, err
	}
	supplierID, err := parseOptionalUUID(input.SupplierID, "supplier_id", "ID supplier tidak valid.")
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id := uuid.New()
	var outletID uuid.UUID
	if input.Stock > 0 {
		outletID, err = actor.outletFor(nil)
		if err != nil {
			return nil, err
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.validateCategory(ctx, tx, tenantID, categoryID); err != nil {
			return err
		}
		if err := s.validateSupplier(ctx, tx, tenantID, supplierID); err != nil {
			return err
		}

		var duplicate bool
		if err := tx.WithContext(ctx).Raw(`
			SELECT EXISTS (SELECT 1 FROM spareparts
			WHERE tenant_id = ? AND deleted_at IS NULL AND lower(sku) = lower(?))`,
			tenantID, sku).Scan(&duplicate).Error; err != nil {
			return apperror.Internal(err)
		}
		if duplicate {
			return apperror.Conflict("DUPLICATE_SKU", "SKU sudah dipakai sparepart lain.")
		}
		if barcode != nil {
			if err := tx.WithContext(ctx).Raw(`
				SELECT EXISTS (SELECT 1 FROM spareparts
				WHERE tenant_id = ? AND deleted_at IS NULL AND lower(barcode) = lower(?))`,
				tenantID, *barcode).Scan(&duplicate).Error; err != nil {
				return apperror.Internal(err)
			}
			if duplicate {
				return apperror.Conflict("DUPLICATE_BARCODE", "Barcode sudah dipakai sparepart lain.")
			}
		}

		isActive := true
		if input.IsActive != nil {
			isActive = *input.IsActive
		}
		var lastStockAt *time.Time
		if input.Stock > 0 {
			lastStockAt = &now
		}

		if err := tx.Exec(`
			INSERT INTO spareparts (
				id, tenant_id, category_id, supplier_id, sku, barcode, name,
				compatible_models, stock, min_stock, cost_price, sell_price,
				location, is_active, last_stock_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, tenantID, categoryID, supplierID, sku, barcode, name,
			trimOptional(input.CompatibleModels), input.Stock, input.MinStock,
			input.CostPrice, input.SellPrice, trimOptional(input.Location),
			isActive, lastStockAt, now, now).Error; err != nil {
			if isUniqueViolation(err) {
				return apperror.Conflict("DUPLICATE_SPAREPART", "SKU atau barcode sudah dipakai sparepart lain.")
			}
			return apperror.Internal(err)
		}

		if input.Stock > 0 {
			referenceID := uuid.New()
			if err := tx.Exec(`
				INSERT INTO inventory_movements (
					id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
					stock_before, stock_after, unit_cost, reference_type, reference_id,
					notes, created_by, created_at
				) VALUES (?, ?, ?, ?, 'initial', ?, 0, ?, ?, 'opname', ?, ?, ?, ?)`,
				uuid.New(), tenantID, outletID, id, input.Stock, input.Stock,
				input.CostPrice, referenceID, "Stok awal sparepart", actorID(actor), now).Error; err != nil {
				return apperror.Internal(err)
			}
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "sparepart_create",
			Module:     "sparepart",
			EntityType: "sparepart",
			EntityID:   &id,
			After: map[string]any{
				"sku": sku, "name": name, "stock": input.Stock,
				"cost_price": input.CostPrice, "sell_price": input.SellPrice,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.getSparepart(ctx, tenantID, id, actor.HasPermission("inventory.cost.view"))
}

// UpdateSparepart edits master data. Stock only changes through purchases and
// adjustments; empty-string category/supplier/barcode clears the field.
func (s *Service) UpdateSparepart(ctx context.Context, actor Actor, id uuid.UUID, input UpdateSparepartRequest) (*Sparepart, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row sparepartRow
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, sku, barcode, name, category_id, supplier_id, compatible_models,
			       stock, min_stock, cost_price, sell_price, location, is_active
			FROM spareparts
			WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL
			FOR UPDATE`, id, tenantID).Scan(&row).Error; err != nil {
			return apperror.Internal(err)
		}
		if row.ID == uuid.Nil {
			return apperror.NotFound("Sparepart tidak ditemukan.")
		}

		sku := row.SKU
		if input.SKU != nil {
			sku = strings.TrimSpace(*input.SKU)
			if sku == "" {
				return apperror.Validation(map[string]string{"sku": "SKU wajib diisi."})
			}
		}
		name := row.Name
		if input.Name != nil {
			name = strings.TrimSpace(*input.Name)
			if name == "" {
				return apperror.Validation(map[string]string{"name": "Nama sparepart wajib diisi."})
			}
		}
		barcode := row.Barcode
		if input.Barcode != nil {
			barcode = trimOptional(input.Barcode)
		}
		categoryID := row.CategoryID
		if input.CategoryID != nil {
			categoryID, err = parseOptionalUUID(input.CategoryID, "category_id", "ID kategori tidak valid.")
			if err != nil {
				return err
			}
		}
		supplierID := row.SupplierID
		if input.SupplierID != nil {
			supplierID, err = parseOptionalUUID(input.SupplierID, "supplier_id", "ID supplier tidak valid.")
			if err != nil {
				return err
			}
		}
		if err := s.validateCategory(ctx, tx, tenantID, categoryID); err != nil {
			return err
		}
		if err := s.validateSupplier(ctx, tx, tenantID, supplierID); err != nil {
			return err
		}

		var duplicate bool
		if err := tx.WithContext(ctx).Raw(`
			SELECT EXISTS (SELECT 1 FROM spareparts
			WHERE tenant_id = ? AND deleted_at IS NULL AND id <> ? AND lower(sku) = lower(?))`,
			tenantID, id, sku).Scan(&duplicate).Error; err != nil {
			return apperror.Internal(err)
		}
		if duplicate {
			return apperror.Conflict("DUPLICATE_SKU", "SKU sudah dipakai sparepart lain.")
		}
		if barcode != nil {
			if err := tx.WithContext(ctx).Raw(`
				SELECT EXISTS (SELECT 1 FROM spareparts
				WHERE tenant_id = ? AND deleted_at IS NULL AND id <> ? AND lower(barcode) = lower(?))`,
				tenantID, id, *barcode).Scan(&duplicate).Error; err != nil {
				return apperror.Internal(err)
			}
			if duplicate {
				return apperror.Conflict("DUPLICATE_BARCODE", "Barcode sudah dipakai sparepart lain.")
			}
		}

		compatibleModels := row.CompatibleModels
		if input.CompatibleModels != nil {
			compatibleModels = trimOptional(input.CompatibleModels)
		}
		minStock := row.MinStock
		if input.MinStock != nil {
			minStock = *input.MinStock
		}
		costPrice := row.CostPrice
		if input.CostPrice != nil {
			costPrice = *input.CostPrice
		}
		sellPrice := row.SellPrice
		if input.SellPrice != nil {
			sellPrice = *input.SellPrice
		}
		location := row.Location
		if input.Location != nil {
			location = trimOptional(input.Location)
		}
		isActive := row.IsActive
		if input.IsActive != nil {
			isActive = *input.IsActive
		}

		if err := tx.Exec(`
			UPDATE spareparts SET sku = ?, barcode = ?, name = ?, category_id = ?,
				supplier_id = ?, compatible_models = ?, min_stock = ?, cost_price = ?,
				sell_price = ?, location = ?, is_active = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`,
			sku, barcode, name, categoryID, supplierID, compatibleModels, minStock,
			costPrice, sellPrice, location, isActive, now, id, tenantID).Error; err != nil {
			if isUniqueViolation(err) {
				return apperror.Conflict("DUPLICATE_SPAREPART", "SKU atau barcode sudah dipakai sparepart lain.")
			}
			return apperror.Internal(err)
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "sparepart_update",
			Module:     "sparepart",
			EntityType: "sparepart",
			EntityID:   &id,
			Before: map[string]any{
				"sku": row.SKU, "name": row.Name, "min_stock": row.MinStock,
				"cost_price": row.CostPrice, "sell_price": row.SellPrice,
				"is_active": row.IsActive,
			},
			After: map[string]any{
				"sku": sku, "name": name, "min_stock": minStock,
				"cost_price": costPrice, "sell_price": sellPrice,
				"is_active": isActive,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.getSparepart(ctx, tenantID, id, actor.HasPermission("inventory.cost.view"))
}

// DeleteSparepart soft-deletes a part while keeping movement history.
func (s *Service) DeleteSparepart(ctx context.Context, actor Actor, id uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	now := time.Now().UTC()

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID   uuid.UUID `gorm:"column:id"`
			SKU  string    `gorm:"column:sku"`
			Name string    `gorm:"column:name"`
		}
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, sku, name FROM spareparts
			WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL
			FOR UPDATE`, id, tenantID).Scan(&row).Error; err != nil {
			return apperror.Internal(err)
		}
		if row.ID == uuid.Nil {
			return apperror.NotFound("Sparepart tidak ditemukan.")
		}
		if err := tx.Exec(`
			UPDATE spareparts SET deleted_at = ?, is_active = false, updated_at = ?
			WHERE id = ? AND tenant_id = ?`, now, now, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID:   &tenantID,
			UserID:     actorID(actor),
			ActorName:  actor.Name,
			Action:     "sparepart_delete",
			Module:     "sparepart",
			EntityType: "sparepart",
			EntityID:   &id,
			Before:     map[string]any{"sku": row.SKU, "name": row.Name},
		})
	})
}

func (s *Service) outletName(ctx context.Context, tx *gorm.DB, tenantID, outletID uuid.UUID) (string, error) {
	var name string
	if err := tx.WithContext(ctx).Raw(`
		SELECT name FROM outlets WHERE id = ? AND tenant_id = ?`, outletID, tenantID).Scan(&name).Error; err != nil {
		return "", apperror.Internal(err)
	}
	if strings.TrimSpace(name) == "" {
		return "", apperror.Validation(map[string]string{"outlet_id": "Outlet tidak ditemukan."})
	}
	return name, nil
}

func optionalString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	id := value.String()
	return &id
}
