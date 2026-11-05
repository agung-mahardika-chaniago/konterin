package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Options configures storage and encryption for the service module.
type Options struct {
	UploadDir      string
	UploadMaxBytes int64
	EncKey         []byte
	AppBaseURL     string
}

// Service implements customer intake and service-order management.
type Service struct {
	db   *gorm.DB
	opts Options
}

// NewService builds the service module.
func NewService(db *gorm.DB, opts Options) *Service {
	if opts.UploadMaxBytes <= 0 {
		opts.UploadMaxBytes = 5 << 20
	}
	return &Service{db: db, opts: opts}
}

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

func (s *Service) outletTimezone(ctx context.Context, tenantID, outletID uuid.UUID) (*time.Location, error) {
	var timezone string
	if err := s.db.WithContext(ctx).Raw(`
		SELECT timezone FROM outlets WHERE id = ? AND tenant_id = ?`,
		outletID, tenantID).Scan(&timezone).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if strings.TrimSpace(timezone) == "" {
		timezone = "Asia/Jakarta"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return loc, nil
}

func (s *Service) outletExists(ctx context.Context, tenantID, outletID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM outlets WHERE id = ? AND tenant_id = ? AND is_active = true)`,
		outletID, tenantID).Scan(&exists).Error
	return exists, err
}

func (s *Service) customerExists(ctx context.Context, tenantID, customerID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM customers
		WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL)`,
		customerID, tenantID).Scan(&exists).Error
	return exists, err
}

func (s *Service) technicianExists(ctx context.Context, tenantID, outletID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM users u
			JOIN roles r ON r.id = u.role_id AND r.slug = 'technician'
			LEFT JOIN user_outlets uo ON uo.user_id = u.id AND uo.outlet_id = ?
			WHERE u.id = ? AND u.tenant_id = ? AND u.deleted_at IS NULL AND u.is_active = true
			  AND (uo.outlet_id IS NOT NULL OR NOT EXISTS (
			      SELECT 1 FROM user_outlets x WHERE x.user_id = u.id
			  ))
		)`, outletID, userID, tenantID).Scan(&exists).Error
	return exists, err
}

// NormalizePhone converts an Indonesian phone number to the 62-prefixed form.
func NormalizePhone(raw string) (display, normalized string, err error) {
	display = strings.TrimSpace(raw)
	if display == "" {
		return "", "", errors.New("Nomor WhatsApp wajib diisi.")
	}
	if len(display) > 30 {
		return "", "", errors.New("Nomor WhatsApp maksimal 30 karakter.")
	}

	var digits strings.Builder
	for _, r := range display {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	value := digits.String()
	switch {
	case strings.HasPrefix(value, "62"):
		value = strings.TrimPrefix(value, "62")
		value = strings.TrimPrefix(value, "0")
		value = "62" + value
	case strings.HasPrefix(value, "0"):
		value = "62" + strings.TrimPrefix(value, "0")
	case strings.HasPrefix(value, "8"):
		value = "62" + value
	default:
		return "", "", errors.New("Nomor WhatsApp harus diawali 08 atau 62.")
	}
	if len(value) < 10 || len(value) > 15 {
		return "", "", errors.New("Nomor WhatsApp tidak valid.")
	}
	return display, value, nil
}

func validateName(raw string, max int) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("Wajib diisi.")
	}
	if len(value) > max {
		return "", errors.New("Terlalu panjang.")
	}
	return value, nil
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

func trimRaw(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseTimePointer(raw *string) (*time.Time, error) {
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}

func (s *Service) technicianOnlyAssigned(ctx context.Context, tenantID, outletID uuid.UUID) bool {
	var assigned bool
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE((
			SELECT technician_sees_only_assigned
			FROM business_settings
			WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
			ORDER BY (outlet_id IS NULL)
			LIMIT 1
		), true)`, tenantID, outletID).Scan(&assigned).Error; err != nil {
		return true
	}
	return assigned
}

var sequencePattern = regexp.MustCompile(`\{SEQ:(\d+)\}`)
