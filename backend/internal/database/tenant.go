package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	slugPattern       = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,58}[a-z0-9])?$`)
	outletCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,19}$`)
)

// CreateTenantInput describes the first tenant, its main outlet, and its owner.
type CreateTenantInput struct {
	Name          string
	Slug          string
	OwnerName     string
	OwnerEmail    string
	OwnerPassword string
	OutletName    string
	OutletCode    string
	Timezone      string
	Phone         string
	Address       string
}

// CreateTenantResult contains the created entity IDs.
type CreateTenantResult struct {
	TenantID uuid.UUID
	OutletID uuid.UUID
	OwnerID  uuid.UUID
}

// CreateTenant provisions a tenant with a main outlet, tenant settings,
// default expense categories, and an owner account.
func CreateTenant(ctx context.Context, db *gorm.DB, input CreateTenantInput) (*CreateTenantResult, error) {
	normalized, err := normalizeTenantInput(input)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(normalized.OwnerPassword), BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password owner: %w", err)
	}

	result := &CreateTenantResult{}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists bool
		if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM tenants WHERE slug = ?)`, normalized.Slug).Scan(&exists).Error; err != nil {
			return fmt.Errorf("memeriksa slug tenant: %w", err)
		}
		if exists {
			return fmt.Errorf("tenant dengan slug %q sudah ada", normalized.Slug)
		}

		if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM users WHERE email = ? AND deleted_at IS NULL)`, normalized.OwnerEmail).Scan(&exists).Error; err != nil {
			return fmt.Errorf("memeriksa email owner: %w", err)
		}
		if exists {
			return fmt.Errorf("email %q sudah dipakai pengguna lain", normalized.OwnerEmail)
		}

		var ownerRoleRaw string
		if err := tx.Raw(`SELECT id::text FROM roles WHERE slug = 'owner'`).Scan(&ownerRoleRaw).Error; err != nil {
			return fmt.Errorf("mencari role owner: %w", err)
		}
		if ownerRoleRaw == "" {
			return errors.New("role owner belum ada; jalankan perintah seed terlebih dahulu")
		}
		ownerRoleID, err := uuid.Parse(ownerRoleRaw)
		if err != nil {
			return fmt.Errorf("membaca role owner: %w", err)
		}

		tenantID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("membuat ID tenant: %w", err)
		}
		outletID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("membuat ID outlet: %w", err)
		}
		settingsID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("membuat ID pengaturan: %w", err)
		}
		ownerID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("membuat ID owner: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO tenants (id, name, slug, status, plan, owner_name, owner_phone)
			VALUES (?, ?, ?, 'active', 'free', ?, NULLIF(?, ''))`,
			tenantID, normalized.Name, normalized.Slug, normalized.OwnerName, normalized.Phone).Error; err != nil {
			return fmt.Errorf("membuat tenant: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO outlets (id, tenant_id, code, name, address, phone, timezone, is_active)
			VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, true)`,
			outletID, tenantID, normalized.OutletCode, normalized.OutletName,
			normalized.Address, normalized.Phone, normalized.Timezone).Error; err != nil {
			return fmt.Errorf("membuat outlet: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO business_settings (
				id, tenant_id, outlet_id, store_name, address, phone, timezone,
				ticket_prefix, ticket_format, invoice_prefix, invoice_format
			) VALUES (?, ?, NULL, ?, NULLIF(?, ''), NULLIF(?, ''), ?, 'FP', '{PREFIX}-{YYMMDD}-{SEQ:4}', 'INV', '{PREFIX}-{YYMMDD}-{SEQ:4}')`,
			settingsID, tenantID, normalized.Name, normalized.Address, normalized.Phone, normalized.Timezone).Error; err != nil {
			return fmt.Errorf("membuat pengaturan bisnis: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO users (id, tenant_id, role_id, name, email, phone, password_hash, is_active, must_change_password)
			VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, true, false)`,
			ownerID, tenantID, ownerRoleID, normalized.OwnerName, normalized.OwnerEmail,
			normalized.Phone, string(hash)).Error; err != nil {
			return fmt.Errorf("membuat owner: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO expense_categories (tenant_id, name, is_system)
			SELECT ?, v.name, true
			FROM (VALUES
				('Operasional'), ('Gaji'), ('Sewa'),
				('Listrik & Internet'), ('Pembelian Alat'), ('Lain-lain')
			) AS v(name)
			ON CONFLICT (tenant_id, name) DO NOTHING`, tenantID).Error; err != nil {
			return fmt.Errorf("membuat kategori pengeluaran: %w", err)
		}

		result.TenantID = tenantID
		result.OutletID = outletID
		result.OwnerID = ownerID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeTenantInput(input CreateTenantInput) (CreateTenantInput, error) {
	out := input
	out.Name = strings.TrimSpace(input.Name)
	out.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	out.OwnerName = strings.TrimSpace(input.OwnerName)
	out.OwnerEmail = strings.ToLower(strings.TrimSpace(input.OwnerEmail))
	out.OutletName = strings.TrimSpace(input.OutletName)
	out.OutletCode = strings.ToUpper(strings.TrimSpace(input.OutletCode))
	out.Timezone = strings.TrimSpace(input.Timezone)
	out.Phone = strings.TrimSpace(input.Phone)
	out.Address = strings.TrimSpace(input.Address)

	if out.Name == "" {
		return out, errors.New("nama tenant wajib diisi")
	}
	if len(out.Name) > 150 {
		return out, errors.New("nama tenant maksimal 150 karakter")
	}
	if !slugPattern.MatchString(out.Slug) {
		return out, errors.New("slug harus 3-60 karakter, huruf kecil/angka/tanda hubung, dan tidak diawali atau diakhiri tanda hubung")
	}

	email, err := NormalizeEmail(out.OwnerEmail)
	if err != nil {
		return out, fmt.Errorf("email owner: %w", err)
	}
	out.OwnerEmail = email
	if err := ValidatePassword(out.OwnerPassword); err != nil {
		return out, fmt.Errorf("password owner: %w", err)
	}

	if out.OwnerName == "" {
		out.OwnerName = "Owner"
	}
	if len(out.OwnerName) > 120 {
		return out, errors.New("nama owner maksimal 120 karakter")
	}
	if out.OutletName == "" {
		out.OutletName = "Outlet Utama"
	}
	if len(out.OutletName) > 120 {
		return out, errors.New("nama outlet maksimal 120 karakter")
	}
	if out.OutletCode == "" {
		out.OutletCode = "MAIN"
	}
	if !outletCodePattern.MatchString(out.OutletCode) {
		return out, errors.New("kode outlet harus 2-20 karakter huruf besar/angka/tanda hubung")
	}
	if out.Timezone == "" {
		out.Timezone = "Asia/Jakarta"
	}
	if len(out.Timezone) > 50 {
		return out, errors.New("timezone maksimal 50 karakter")
	}
	if _, err := time.LoadLocation(out.Timezone); err != nil {
		return out, fmt.Errorf("timezone tidak valid: %s", out.Timezone)
	}
	if len(out.Phone) > 30 {
		return out, errors.New("nomor telepon maksimal 30 karakter")
	}
	return out, nil
}
