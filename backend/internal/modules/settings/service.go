package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service implements settings and backup operations.
type Service struct {
	db        *gorm.DB
	backupDir string
	backupDSN string
}

// NewService builds the settings module.
func NewService(db *gorm.DB, backupDir, backupDSN string) *Service {
	return &Service{db: db, backupDir: backupDir, backupDSN: backupDSN}
}

// Actor mirrors the minimal caller identity needed by this module.
type Actor struct {
	UserID   uuid.UUID
	Name     string
	TenantID uuid.UUID
}

func tenantScope(actor Actor) (uuid.UUID, error) {
	if actor.TenantID == uuid.Nil {
		return uuid.Nil, apperror.Forbidden("FORBIDDEN", "Pengaturan hanya tersedia untuk pengguna tenant.")
	}
	return actor.TenantID, nil
}

// userIDPtr returns a pointer to the actor user id, or nil for anonymous calls.
func (a Actor) userIDPtr() *uuid.UUID {
	if a.UserID == uuid.Nil {
		return nil
	}
	id := a.UserID
	return &id
}

// settingsRow maps the business_settings tenant-level row.
type settingsRow struct {
	StoreName                  string    `gorm:"column:store_name"`
	LegalName                  *string   `gorm:"column:legal_name"`
	LogoPath                   *string   `gorm:"column:logo_path"`
	Address                    *string   `gorm:"column:address"`
	Phone                      *string   `gorm:"column:phone"`
	Email                      *string   `gorm:"column:email"`
	TaxID                      *string   `gorm:"column:tax_id"`
	Timezone                   string    `gorm:"column:timezone"`
	TicketPrefix               string    `gorm:"column:ticket_prefix"`
	TicketFormat               string    `gorm:"column:ticket_format"`
	InvoicePrefix              string    `gorm:"column:invoice_prefix"`
	InvoiceFormat              string    `gorm:"column:invoice_format"`
	InvoiceTerms               *string   `gorm:"column:invoice_terms"`
	InvoiceFooter              *string   `gorm:"column:invoice_footer"`
	DefaultServiceWarrantyDays int       `gorm:"column:default_service_warranty_days"`
	DefaultPartWarrantyDays    int       `gorm:"column:default_part_warranty_days"`
	ReceiptPaperWidth          int       `gorm:"column:receipt_paper_width"`
	TechnicianSeesOnlyAssigned bool      `gorm:"column:technician_sees_only_assigned"`
	PublicTrackingEnabled      bool      `gorm:"column:public_tracking_enabled"`
	WAProvider                 string    `gorm:"column:wa_provider"`
	WATemplates                []byte    `gorm:"column:wa_templates"`
	UpdatedAt                  time.Time `gorm:"column:updated_at"`
}

const settingsColumns = `
	store_name, legal_name, logo_path, address, phone, email, tax_id, timezone,
	ticket_prefix, ticket_format, invoice_prefix, invoice_format,
	invoice_terms, invoice_footer, default_service_warranty_days,
	default_part_warranty_days, receipt_paper_width,
	technician_sees_only_assigned, public_tracking_enabled,
	wa_provider, wa_templates, updated_at`

func (r settingsRow) toSettings() (*Settings, error) {
	templates := WATemplates{}
	if len(r.WATemplates) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(r.WATemplates, &raw); err == nil {
			templates = templatesFromMap(raw)
		}
	}
	return &Settings{
		StoreName: r.StoreName, LegalName: r.LegalName, LogoPath: r.LogoPath,
		Address: r.Address, Phone: r.Phone, Email: r.Email, TaxID: r.TaxID,
		Timezone:     r.Timezone,
		TicketPrefix: r.TicketPrefix, TicketFormat: r.TicketFormat,
		InvoicePrefix: r.InvoicePrefix, InvoiceFormat: r.InvoiceFormat,
		InvoiceTerms: r.InvoiceTerms, InvoiceFooter: r.InvoiceFooter,
		DefaultServiceWarrantyDays: r.DefaultServiceWarrantyDays,
		DefaultPartWarrantyDays:    r.DefaultPartWarrantyDays,
		ReceiptPaperWidth:          r.ReceiptPaperWidth,
		TechnicianSeesOnlyAssigned: r.TechnicianSeesOnlyAssigned,
		PublicTrackingEnabled:      r.PublicTrackingEnabled,
		WAProvider:                 r.WAProvider, WATemplates: templates,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

func templatesFromMap(raw map[string]any) WATemplates {
	get := func(key string) string {
		if value, ok := raw[key]; ok {
			switch v := value.(type) {
			case string:
				return v
			case map[string]any:
				if body, ok := v["body"].(string); ok {
					return body
				}
			}
		}
		return ""
	}
	return WATemplates{
		StatusUpdate:  get("status_update"),
		ReadyPickup:   get("ready_for_pickup"),
		Invoice:       get("invoice"),
		Reminder:      get("reminder"),
		Warranty:      get("warranty"),
		ClaimReceived: get("claim_received"),
	}
}

// Get returns the tenant-level business settings.
func (s *Service) Get(ctx context.Context, actor Actor) (*Settings, error) {
	tenantID, err := tenantScope(actor)
	if err != nil {
		return nil, err
	}
	return s.readRow(ctx, s.db, tenantID)
}

// readRow loads the tenant settings row through the given DB handle. The
// transaction callers pass their own handle so the freshly written values are
// visible before commit.
func (s *Service) readRow(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) (*Settings, error) {
	var row settingsRow
	if err := db.WithContext(ctx).Raw(`
		SELECT `+settingsColumns+`
		FROM business_settings
		WHERE tenant_id = ? AND outlet_id IS NULL`,
		tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if strings.TrimSpace(row.StoreName) == "" {
		return nil, apperror.NotFound("Pengaturan toko belum diinisialisasi untuk tenant ini.")
	}
	return row.toSettings()
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// Update partially updates the tenant-level business settings.
func (s *Service) Update(ctx context.Context, actor Actor, input UpdateSettingsRequest) (*Settings, error) {
	tenantID, err := tenantScope(actor)
	if err != nil {
		return nil, err
	}
	if input.StoreName != nil {
		if strings.TrimSpace(*input.StoreName) == "" {
			return nil, apperror.Validation(map[string]string{"store_name": "Nama toko wajib diisi."})
		}
	}
	if input.Timezone != nil {
		tz := strings.TrimSpace(*input.Timezone)
		if tz != "" {
			if _, loadErr := time.LoadLocation(tz); loadErr != nil {
				return nil, apperror.Validation(map[string]string{"timezone": "Zona waktu tidak dikenal."})
			}
		}
	}
	for field, value := range map[string]*string{
		"ticket_prefix":  input.TicketPrefix,
		"invoice_prefix": input.InvoicePrefix,
	} {
		if value != nil && strings.TrimSpace(*value) == "" {
			return nil, apperror.Validation(map[string]string{field: "Prefiks tidak boleh kosong."})
		}
	}

	var templatesJSON []byte
	if input.WATemplates != nil {
		templatesJSON, err = json.Marshal(map[string]string{
			"status_update":    input.WATemplates.StatusUpdate,
			"ready_for_pickup": input.WATemplates.ReadyPickup,
			"invoice":          input.WATemplates.Invoice,
			"reminder":         input.WATemplates.Reminder,
			"warranty":         input.WATemplates.Warranty,
			"claim_received":   input.WATemplates.ClaimReceived,
		})
		if err != nil {
			return nil, apperror.Internal(err)
		}
	}

	var updated *Settings
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		var current settingsRow
		if err := tx.Raw(`
			SELECT `+settingsColumns+`
			FROM business_settings
			WHERE tenant_id = ? AND outlet_id IS NULL
			FOR UPDATE`, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if strings.TrimSpace(current.StoreName) == "" {
			return apperror.NotFound("Pengaturan toko belum diinisialisasi untuk tenant ini.")
		}

		sets := []string{}
		args := []any{}
		apply := func(column string, value any) {
			sets = append(sets, column+" = ?")
			args = append(args, value)
		}

		if input.StoreName != nil {
			apply("store_name", strings.TrimSpace(*input.StoreName))
		}
		if input.LegalName != nil {
			apply("legal_name", trimPtr(input.LegalName))
		}
		if input.Address != nil {
			apply("address", trimPtr(input.Address))
		}
		if input.Phone != nil {
			apply("phone", trimPtr(input.Phone))
		}
		if input.Email != nil {
			apply("email", trimPtr(input.Email))
		}
		if input.TaxID != nil {
			apply("tax_id", trimPtr(input.TaxID))
		}
		if input.Timezone != nil {
			apply("timezone", strings.TrimSpace(*input.Timezone))
		}
		if input.TicketPrefix != nil {
			apply("ticket_prefix", strings.TrimSpace(*input.TicketPrefix))
		}
		if input.TicketFormat != nil {
			apply("ticket_format", strings.TrimSpace(*input.TicketFormat))
		}
		if input.InvoicePrefix != nil {
			apply("invoice_prefix", strings.TrimSpace(*input.InvoicePrefix))
		}
		if input.InvoiceFormat != nil {
			apply("invoice_format", strings.TrimSpace(*input.InvoiceFormat))
		}
		if input.InvoiceTerms != nil {
			apply("invoice_terms", trimPtr(input.InvoiceTerms))
		}
		if input.InvoiceFooter != nil {
			apply("invoice_footer", trimPtr(input.InvoiceFooter))
		}
		if input.DefaultServiceWarrantyDays != nil {
			apply("default_service_warranty_days", *input.DefaultServiceWarrantyDays)
		}
		if input.DefaultPartWarrantyDays != nil {
			apply("default_part_warranty_days", *input.DefaultPartWarrantyDays)
		}
		if input.ReceiptPaperWidth != nil {
			apply("receipt_paper_width", *input.ReceiptPaperWidth)
		}
		if input.TechnicianSeesOnlyAssigned != nil {
			apply("technician_sees_only_assigned", *input.TechnicianSeesOnlyAssigned)
		}
		if input.PublicTrackingEnabled != nil {
			apply("public_tracking_enabled", *input.PublicTrackingEnabled)
		}
		if input.WAProvider != nil {
			apply("wa_provider", strings.TrimSpace(*input.WAProvider))
		}
		if templatesJSON != nil {
			apply("wa_templates", templatesJSON)
		}

		if len(sets) == 0 {
			result, getErr := s.readRow(ctx, tx, tenantID)
			updated = result
			return getErr
		}
		sets = append(sets, "updated_at = now()")
		args = append(args, tenantID)

		if err := tx.Exec(`
			UPDATE business_settings SET `+strings.Join(sets, ", ")+`
			WHERE tenant_id = ? AND outlet_id IS NULL`, args...).Error; err != nil {
			return apperror.Internal(err)
		}

		if err := audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actor.userIDPtr(), ActorName: actor.Name,
			Action: "settings_update", Module: "settings",
			EntityType: "business_settings", EntityID: nil,
			After: map[string]any{"fields": sets},
		}); err != nil {
			return apperror.Internal(err)
		}

		result, getErr := s.readRow(ctx, tx, tenantID)
		updated = result
		return getErr
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// ListBackups returns recent manual backups for the tenant.
func (s *Service) ListBackups(ctx context.Context, actor Actor) ([]Backup, error) {
	tenantID, err := tenantScope(actor)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID         uuid.UUID `gorm:"column:id"`
		FileName   string    `gorm:"column:file_name"`
		SizeBytes  int64     `gorm:"column:size_bytes"`
		Status     string    `gorm:"column:status"`
		TableCount int       `gorm:"column:table_count"`
		RowCount   int64     `gorm:"column:row_count"`
		Notes      *string   `gorm:"column:notes"`
		CreatedAt  time.Time `gorm:"column:created_at"`
		CreatedBy  *string   `gorm:"column:created_by"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT b.id, b.file_name, b.size_bytes, b.status, b.table_count,
			b.row_count, b.notes, b.created_at, u.name AS created_by
		FROM backups b
		LEFT JOIN users u ON u.id = b.created_by
		WHERE b.tenant_id = ?
		ORDER BY b.created_at DESC
		LIMIT 50`, tenantID).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]Backup, 0, len(rows))
	for _, row := range rows {
		items = append(items, Backup{
			ID: row.ID.String(), FileName: row.FileName, SizeBytes: row.SizeBytes,
			Status: row.Status, TableCount: row.TableCount, RowCount: row.RowCount,
			Notes: row.Notes, CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy,
		})
	}
	return items, nil
}

// backupTenantTables lists tenant-owned tables to dump, excluding platform
// tables (tenants, roles, permissions, users of other tenants). Tables marked
// with a filter have no tenant_id column and scope themselves through a join.
func backupTenantTables() []backupTable {
	tenantScoped := []string{
		"customers", "devices", "service_orders", "service_order_items",
		"service_diagnostics", "service_status_history", "service_photos",
		"sparepart_categories", "suppliers", "spareparts", "inventory_movements",
		"invoices", "invoice_items", "payments",
		"warranties", "warranty_claims", "expense_categories", "expenses",
		"notifications", "whatsapp_messages", "audit_logs", "backups",
	}
	tables := make([]backupTable, 0, len(tenantScoped)+2)
	for _, name := range tenantScoped {
		tables = append(tables, backupTable{name: name})
	}
	return append(tables,
		backupTable{
			name:   "notification_reads",
			filter: "notification_id IN (SELECT id FROM notifications WHERE tenant_id = '%s')",
		},
		backupTable{
			name:   "user_outlets",
			filter: "user_id IN (SELECT id FROM users WHERE tenant_id = '%s')",
		},
	)
}

// CreateBackup dumps tenant rows into a compressed file and records metadata.
func (s *Service) CreateBackup(ctx context.Context, actor Actor) (*Backup, error) {
	tenantID, err := tenantScope(actor)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.backupDSN) == "" {
		return nil, apperror.Internal(errors.New("BACKUP_DSN tidak dikonfigurasi"))
	}
	if err := ensureDir(s.backupDir); err != nil {
		return nil, apperror.Internal(err)
	}

	fileName := fmt.Sprintf("backup-%s-%s.sql.gz", tenantID, time.Now().UTC().Format("20060102-150405"))
	backupID, idErr := uuid.NewV7()
	if idErr != nil {
		return nil, apperror.Internal(idErr)
	}

	dump := NewTenantDump(s.backupDSN, s.backupDir, fileName)
	stats, dumpErr := dump.Run(ctx, tenantID, backupTenantTables())
	if dumpErr != nil {
		audit.BestEffort(ctx, s.db, audit.Entry{
			TenantID: &tenantID, UserID: actor.userIDPtr(), ActorName: actor.Name,
			Action: "backup_create_failed", Module: "settings",
			EntityType: "backup", EntityID: &backupID,
		})
		return nil, apperror.Internal(dumpErr)
	}

	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO backups (id, tenant_id, created_by, file_name, size_bytes,
			status, table_count, row_count)
		VALUES (?, ?, ?, ?, ?, 'completed', ?, ?)`,
		backupID, tenantID, actor.UserID, fileName,
		stats.SizeBytes, stats.TableCount, stats.RowCount).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actor.userIDPtr(), ActorName: actor.Name,
		Action: "backup_create", Module: "settings",
		EntityType: "backup", EntityID: &backupID,
		After: map[string]any{"file_name": fileName, "size_bytes": stats.SizeBytes},
	})

	return &Backup{
		ID: backupID.String(), FileName: fileName, SizeBytes: stats.SizeBytes,
		Status: "completed", TableCount: stats.TableCount, RowCount: stats.RowCount,
		CreatedAt: time.Now().UTC(), CreatedBy: &actor.Name,
	}, nil
}

// GetBackupFile validates access and returns the backup file path.
func (s *Service) GetBackupFile(ctx context.Context, actor Actor, backupID uuid.UUID) (string, error) {
	tenantID, err := tenantScope(actor)
	if err != nil {
		return "", err
	}
	var row struct {
		FileName string `gorm:"column:file_name"`
		Status   string `gorm:"column:status"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT file_name, status FROM backups
		WHERE id = ? AND tenant_id = ?`, backupID, tenantID).Scan(&row).Error; err != nil {
		return "", apperror.Internal(err)
	}
	if row.FileName == "" {
		return "", apperror.NotFound("Backup tidak ditemukan.")
	}
	if row.Status != "completed" {
		return "", apperror.Conflict("BACKUP_NOT_READY", "Backup belum selesai dibuat.")
	}
	path := filepathJoin(s.backupDir, row.FileName)
	if !fileExists(path) {
		return "", apperror.NotFound("File backup tidak ditemukan di penyimpanan.")
	}
	return path, nil
}
