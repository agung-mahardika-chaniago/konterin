package inventory

import (
	"context"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListCategories returns categories with their active sparepart counts.
func (s *Service) ListCategories(ctx context.Context, actor Actor) ([]Category, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	type row struct {
		ID             uuid.UUID `gorm:"column:id"`
		Name           string    `gorm:"column:name"`
		SparepartCount int64     `gorm:"column:sparepart_count"`
		CreatedAt      time.Time `gorm:"column:created_at"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT c.id, c.name, count(sp.id) AS sparepart_count, c.created_at
		FROM sparepart_categories c
		LEFT JOIN spareparts sp ON sp.category_id = c.id
		  AND sp.tenant_id = c.tenant_id AND sp.deleted_at IS NULL
		WHERE c.tenant_id = ?
		GROUP BY c.id, c.name, c.created_at
		ORDER BY c.name ASC`, tenantID).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]Category, len(rows))
	for i, row := range rows {
		out[i] = Category{
			ID: row.ID.String(), Name: row.Name,
			SparepartCount: row.SparepartCount, CreatedAt: row.CreatedAt,
		}
	}
	return out, nil
}

// CreateCategory adds a sparepart category.
func (s *Service) CreateCategory(ctx context.Context, actor Actor, input CreateCategoryRequest) (*Category, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperror.Validation(map[string]string{"name": "Nama kategori wajib diisi."})
	}
	now := time.Now().UTC()
	id := uuid.New()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var duplicate bool
		if err := tx.WithContext(ctx).Raw(`
			SELECT EXISTS (SELECT 1 FROM sparepart_categories
			WHERE tenant_id = ? AND lower(name) = lower(?))`, tenantID, name).Scan(&duplicate).Error; err != nil {
			return apperror.Internal(err)
		}
		if duplicate {
			return apperror.Conflict("DUPLICATE_CATEGORY", "Nama kategori sudah dipakai.")
		}
		if err := tx.Exec(`
			INSERT INTO sparepart_categories (id, tenant_id, name, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, id, tenantID, name, now, now).Error; err != nil {
			if isUniqueViolation(err) {
				return apperror.Conflict("DUPLICATE_CATEGORY", "Nama kategori sudah dipakai.")
			}
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "sparepart_category_create", Module: "sparepart",
			EntityType: "sparepart_category", EntityID: &id,
			After: map[string]any{"name": name},
		})
	})
	if err != nil {
		return nil, err
	}
	return &Category{ID: id.String(), Name: name, CreatedAt: now}, nil
}

// UpdateCategory renames a category.
func (s *Service) UpdateCategory(ctx context.Context, actor Actor, id uuid.UUID, input UpdateCategoryRequest) (*Category, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if input.Name == nil {
		return nil, apperror.Validation(map[string]string{"name": "Nama kategori wajib diisi."})
	}
	name := strings.TrimSpace(*input.Name)
	if name == "" {
		return nil, apperror.Validation(map[string]string{"name": "Nama kategori wajib diisi."})
	}
	now := time.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before struct {
			ID   uuid.UUID `gorm:"column:id"`
			Name string    `gorm:"column:name"`
		}
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, name FROM sparepart_categories
			WHERE id = ? AND tenant_id = ? FOR UPDATE`, id, tenantID).Scan(&before).Error; err != nil {
			return apperror.Internal(err)
		}
		if before.ID == uuid.Nil {
			return apperror.NotFound("Kategori tidak ditemukan.")
		}
		var duplicate bool
		if err := tx.WithContext(ctx).Raw(`
			SELECT EXISTS (SELECT 1 FROM sparepart_categories
			WHERE tenant_id = ? AND id <> ? AND lower(name) = lower(?))`,
			tenantID, id, name).Scan(&duplicate).Error; err != nil {
			return apperror.Internal(err)
		}
		if duplicate {
			return apperror.Conflict("DUPLICATE_CATEGORY", "Nama kategori sudah dipakai.")
		}
		if err := tx.Exec(`
			UPDATE sparepart_categories SET name = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`, name, now, id, tenantID).Error; err != nil {
			if isUniqueViolation(err) {
				return apperror.Conflict("DUPLICATE_CATEGORY", "Nama kategori sudah dipakai.")
			}
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "sparepart_category_update", Module: "sparepart",
			EntityType: "sparepart_category", EntityID: &id,
			Before: map[string]any{"name": before.Name}, After: map[string]any{"name": name},
		})
	})
	if err != nil {
		return nil, err
	}

	var count int64
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM spareparts
		WHERE tenant_id = ? AND category_id = ? AND deleted_at IS NULL`, tenantID, id).Scan(&count).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return &Category{ID: id.String(), Name: name, SparepartCount: count}, nil
}

// ListSuppliers returns suppliers, optionally filtered by a search query.
func (s *Service) ListSuppliers(ctx context.Context, actor Actor, q string) ([]Supplier, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	where := ` WHERE tenant_id = ? AND deleted_at IS NULL`
	args := []any{tenantID}
	if q = strings.TrimSpace(q); q != "" {
		like := "%" + q + "%"
		where += ` AND (name ILIKE ? OR contact_name ILIKE ? OR phone ILIKE ?)`
		args = append(args, like, like, like)
	}
	type row struct {
		ID          uuid.UUID `gorm:"column:id"`
		Name        string    `gorm:"column:name"`
		ContactName *string   `gorm:"column:contact_name"`
		Phone       *string   `gorm:"column:phone"`
		Address     *string   `gorm:"column:address"`
		Notes       *string   `gorm:"column:notes"`
		CreatedAt   time.Time `gorm:"column:created_at"`
		UpdatedAt   time.Time `gorm:"column:updated_at"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, name, contact_name, phone, address, notes, created_at, updated_at
		FROM suppliers`+where+` ORDER BY name ASC`, args...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]Supplier, len(rows))
	for i, row := range rows {
		out[i] = Supplier{
			ID: row.ID.String(), Name: row.Name, ContactName: row.ContactName,
			Phone: row.Phone, Address: row.Address, Notes: row.Notes,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

// CreateSupplier adds a supplier.
func (s *Service) CreateSupplier(ctx context.Context, actor Actor, input CreateSupplierRequest) (*Supplier, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperror.Validation(map[string]string{"name": "Nama supplier wajib diisi."})
	}
	now := time.Now().UTC()
	id := uuid.New()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO suppliers (id, tenant_id, name, contact_name, phone, address, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, tenantID, name, trimOptional(input.ContactName), trimOptional(input.Phone),
			trimOptional(input.Address), trimOptional(input.Notes), now, now).Error; err != nil {
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "supplier_create", Module: "sparepart",
			EntityType: "supplier", EntityID: &id, After: map[string]any{"name": name},
		})
	})
	if err != nil {
		return nil, err
	}
	return &Supplier{
		ID: id.String(), Name: name, ContactName: trimOptional(input.ContactName),
		Phone: trimOptional(input.Phone), Address: trimOptional(input.Address),
		Notes: trimOptional(input.Notes), CreatedAt: now, UpdatedAt: now,
	}, nil
}

// UpdateSupplier edits a supplier.
func (s *Service) UpdateSupplier(ctx context.Context, actor Actor, id uuid.UUID, input UpdateSupplierRequest) (*Supplier, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	var out Supplier
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before struct {
			ID          uuid.UUID `gorm:"column:id"`
			Name        string    `gorm:"column:name"`
			ContactName *string   `gorm:"column:contact_name"`
			Phone       *string   `gorm:"column:phone"`
			Address     *string   `gorm:"column:address"`
			Notes       *string   `gorm:"column:notes"`
			CreatedAt   time.Time `gorm:"column:created_at"`
		}
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, name, contact_name, phone, address, notes, created_at
			FROM suppliers
			WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL
			FOR UPDATE`, id, tenantID).Scan(&before).Error; err != nil {
			return apperror.Internal(err)
		}
		if before.ID == uuid.Nil {
			return apperror.NotFound("Supplier tidak ditemukan.")
		}
		name := before.Name
		if input.Name != nil {
			name = strings.TrimSpace(*input.Name)
			if name == "" {
				return apperror.Validation(map[string]string{"name": "Nama supplier wajib diisi."})
			}
		}
		contactName := before.ContactName
		if input.ContactName != nil {
			contactName = trimOptional(input.ContactName)
		}
		phone := before.Phone
		if input.Phone != nil {
			phone = trimOptional(input.Phone)
		}
		address := before.Address
		if input.Address != nil {
			address = trimOptional(input.Address)
		}
		notes := before.Notes
		if input.Notes != nil {
			notes = trimOptional(input.Notes)
		}
		if err := tx.Exec(`
			UPDATE suppliers SET name = ?, contact_name = ?, phone = ?, address = ?,
				notes = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`,
			name, contactName, phone, address, notes, now, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		out = Supplier{
			ID: id.String(), Name: name, ContactName: contactName, Phone: phone,
			Address: address, Notes: notes, CreatedAt: before.CreatedAt, UpdatedAt: now,
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "supplier_update", Module: "sparepart",
			EntityType: "supplier", EntityID: &id,
			Before: map[string]any{"name": before.Name}, After: map[string]any{"name": name},
		})
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
