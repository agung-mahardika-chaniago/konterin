package outlet

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,19}$`)

// ErrNotFound indicates a missing outlet.
var ErrNotFound = errors.New("outlet tidak ditemukan")

// Repository is the outlet data access layer.
type Repository struct {
	db *gorm.DB
}

// NewRepository builds an outlet repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// List returns accessible active outlets.
func (r *Repository) List(ctx context.Context, tenantID, userID interface{}, allAccess bool) ([]Outlet, error) {
	items := make([]Outlet, 0, 8)
	query := `SELECT id, tenant_id, code, name, address, phone, timezone, is_active, created_at, updated_at
		FROM outlets WHERE tenant_id = ?`
	args := []any{tenantID}
	if !allAccess {
		query += ` AND id IN (SELECT outlet_id FROM user_outlets WHERE user_id = ?)`
		args = append(args, userID)
	}
	query += ` ORDER BY is_active DESC, name`
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// Find loads an outlet within a tenant.
func (r *Repository) Find(ctx context.Context, tenantID, id interface{}) (*Outlet, error) {
	var item Outlet
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id, tenant_id, code, name, address, phone, timezone, is_active, created_at, updated_at
		FROM outlets WHERE id = ? AND tenant_id = ?`, id, tenantID).Scan(&item).Error; err != nil {
		return nil, err
	}
	if item.ID == uuid.Nil {
		return nil, ErrNotFound
	}
	return &item, nil
}

// CountActive returns the active outlet count for a tenant.
func (r *Repository) CountActive(ctx context.Context, tenantID interface{}) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM outlets WHERE tenant_id = ? AND is_active = true`, tenantID).Scan(&count).Error
	return count, err
}

// Service implements outlet management.
type Service struct {
	db   *gorm.DB
	repo *Repository
}

// NewService builds the outlet service.
func NewService(db *gorm.DB, repo *Repository) *Service {
	return &Service{db: db, repo: repo}
}

// List returns outlets accessible to the actor.
func (s *Service) List(ctx context.Context, actor Actor) ([]DTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(actor.UserID)
	if err != nil {
		return nil, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak valid.")
	}
	allAccess := actor.Role == "owner" || actor.Role == "admin"
	items, err := s.repo.List(ctx, tenantID, userID, allAccess)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]DTO, 0, len(items))
	for _, item := range items {
		out = append(out, toDTO(item))
	}
	return out, nil
}

// Create adds an outlet to the actor's tenant.
func (s *Service) Create(ctx context.Context, actor Actor, input CreateInput) (*DTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if !codePattern.MatchString(code) {
		return nil, apperror.Validation(map[string]string{"code": "Kode outlet harus 2-20 karakter huruf besar/angka/tanda hubung."})
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 120 {
		return nil, apperror.Validation(map[string]string{"name": "Nama outlet wajib diisi, maksimal 120 karakter."})
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, apperror.Validation(map[string]string{"timezone": "Timezone tidak valid."})
	}
	phone := trimOptional(input.Phone)
	if phone != nil && len(*phone) > 30 {
		return nil, apperror.Validation(map[string]string{"phone": "Nomor telepon maksimal 30 karakter."})
	}
	address := trimOptional(input.Address)
	if address != nil && len(*address) > 500 {
		return nil, apperror.Validation(map[string]string{"address": "Alamat maksimal 500 karakter."})
	}

	var exists bool
	if err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM outlets WHERE tenant_id = ? AND code = ?)`,
		tenantID, code).Scan(&exists).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if exists {
		return nil, apperror.Conflict("CONFLICT", "Kode outlet sudah digunakan.")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO outlets (id, tenant_id, code, name, address, phone, timezone, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, code, name, address, phone, timezone, input.IsActive).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("CONFLICT", "Kode outlet sudah digunakan.")
		}
		return nil, apperror.Internal(err)
	}

	item, err := s.repo.Find(ctx, tenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "outlet.create", Module: "outlet", EntityType: "outlet", EntityID: &id,
		After: map[string]any{"code": code, "name": name, "is_active": input.IsActive},
	})
	dto := toDTO(*item)
	return &dto, nil
}

// Update edits an outlet in the actor's tenant.
func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, input UpdateInput) (*DTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.Find(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.NotFound("Outlet tidak ditemukan.")
		}
		return nil, apperror.Internal(err)
	}

	code := current.Code
	if input.Code != nil {
		code = strings.ToUpper(strings.TrimSpace(*input.Code))
		if !codePattern.MatchString(code) {
			return nil, apperror.Validation(map[string]string{"code": "Kode outlet harus 2-20 karakter huruf besar/angka/tanda hubung."})
		}
	}
	name := current.Name
	if input.Name != nil {
		name = strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 120 {
			return nil, apperror.Validation(map[string]string{"name": "Nama outlet wajib diisi, maksimal 120 karakter."})
		}
	}
	address := current.Address
	if input.Address != nil {
		address = trimOptional(input.Address)
		if address != nil && len(*address) > 500 {
			return nil, apperror.Validation(map[string]string{"address": "Alamat maksimal 500 karakter."})
		}
	}
	phone := current.Phone
	if input.Phone != nil {
		phone = trimOptional(input.Phone)
		if phone != nil && len(*phone) > 30 {
			return nil, apperror.Validation(map[string]string{"phone": "Nomor telepon maksimal 30 karakter."})
		}
	}
	timezone := current.Timezone
	if input.Timezone != nil {
		timezone = strings.TrimSpace(*input.Timezone)
		if timezone == "" {
			return nil, apperror.Validation(map[string]string{"timezone": "Timezone wajib diisi."})
		}
		if _, err := time.LoadLocation(timezone); err != nil {
			return nil, apperror.Validation(map[string]string{"timezone": "Timezone tidak valid."})
		}
	}
	isActive := current.IsActive
	if input.IsActive != nil && *input.IsActive != current.IsActive {
		isActive = *input.IsActive
		if !isActive {
			count, err := s.repo.CountActive(ctx, tenantID)
			if err != nil {
				return nil, apperror.Internal(err)
			}
			if count <= 1 {
				return nil, apperror.Conflict("LAST_OUTLET", "Outlet aktif terakhir tidak dapat dinonaktifkan.")
			}
		}
	}

	if err := s.db.WithContext(ctx).Exec(`
		UPDATE outlets
		SET code = ?, name = ?, address = ?, phone = ?, timezone = ?, is_active = ?, updated_at = now()
		WHERE id = ? AND tenant_id = ?`,
		code, name, address, phone, timezone, isActive, current.ID, tenantID).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("CONFLICT", "Kode outlet sudah digunakan.")
		}
		return nil, apperror.Internal(err)
	}

	item, err := s.repo.Find(ctx, tenantID, current.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "outlet.update", Module: "outlet", EntityType: "outlet", EntityID: &current.ID,
		Before: map[string]any{"code": current.Code, "name": current.Name, "is_active": current.IsActive},
		After:  map[string]any{"code": code, "name": name, "is_active": isActive},
	})
	dto := toDTO(*item)
	return &dto, nil
}

func toDTO(item Outlet) DTO {
	return DTO{
		ID: item.ID.String(), Code: item.Code, Name: item.Name, Address: item.Address,
		Phone: item.Phone, Timezone: item.Timezone, IsActive: item.IsActive,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func tenantUUID(actor Actor) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(actor.TenantID))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return id, nil
}

func actorUUID(actor Actor) *uuid.UUID {
	id, err := uuid.Parse(actor.UserID)
	if err != nil {
		return nil
	}
	return &id
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
