package service

import (
	"context"
	"errors"
	"strings"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListCustomers returns a paginated, searchable customer list.
func (s *Service) ListCustomers(ctx context.Context, actor Actor, filter CustomerFilter) (*CustomerList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}

	where := ` WHERE c.tenant_id = ? AND c.deleted_at IS NULL`
	args := []any{tenantID}
	if query := strings.TrimSpace(filter.Query); query != "" {
		where += ` AND (c.name ILIKE ? OR c.phone ILIKE ? OR c.phone_normalized LIKE ?)`
		like := "%" + query + "%"
		_, normalized, phoneErr := NormalizePhone(query)
		if phoneErr != nil {
			normalized = strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' {
					return r
				}
				return -1
			}, query)
		}
		args = append(args, like, like, "%"+normalized+"%")
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM customers c`+where, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]Customer, 0, filter.PerPage)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT c.id, c.tenant_id, c.name, c.phone, c.address, c.notes,
		       c.last_service_at, c.created_at, c.updated_at
		FROM customers c`+where+`
		ORDER BY c.last_service_at DESC NULLS LAST, c.name
		LIMIT ? OFFSET ?`, args...).Scan(&items).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	out := make([]CustomerDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCustomerDTO(item))
	}
	return &CustomerList{Items: out, Total: total}, nil
}

// GetCustomer returns a customer with devices and statistics.
func (s *Service) GetCustomer(ctx context.Context, actor Actor, id uuid.UUID) (*CustomerDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	customer, err := s.findCustomer(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	devices, err := s.devicesForCustomer(ctx, tenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	stats, err := s.customerStats(ctx, tenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	dto := toCustomerDTO(*customer)
	dto.Devices = make([]DeviceDTO, 0, len(devices))
	for _, device := range devices {
		dto.Devices = append(dto.Devices, toDeviceDTO(device))
	}
	dto.Stats = stats
	return &dto, nil
}

// CreateCustomer adds a customer and rejects duplicate active phone numbers.
func (s *Service) CreateCustomer(ctx context.Context, actor Actor, input CreateCustomerRequest) (*CustomerDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	name, nameErr := validateName(input.Name, 120)
	if nameErr != nil {
		return nil, apperror.Validation(map[string]string{"name": "Nama wajib diisi, maksimal 120 karakter."})
	}
	displayPhone, normalized, phoneErr := NormalizePhone(input.Phone)
	if phoneErr != nil {
		return nil, apperror.Validation(map[string]string{"phone": phoneErr.Error()})
	}
	if exists, err := s.phoneExists(ctx, tenantID, normalized, uuid.Nil); err != nil {
		return nil, apperror.Internal(err)
	} else if exists {
		return nil, apperror.Conflict("DUPLICATE_PHONE", "Nomor WhatsApp sudah terdaftar untuk pelanggan lain.")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO customers (id, tenant_id, name, phone, phone_normalized, address, notes, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, name, displayPhone, normalized, trimOptional(input.Address), trimOptional(input.Notes),
		actorID(actor), actorID(actor)).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("DUPLICATE_PHONE", "Nomor WhatsApp sudah terdaftar untuk pelanggan lain.")
		}
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "customer.create", Module: "customer", EntityType: "customer", EntityID: &id,
		After: map[string]any{"name": name, "phone": normalized},
	})
	return s.GetCustomer(ctx, actor, id)
}

// UpdateCustomer edits a customer.
func (s *Service) UpdateCustomer(ctx context.Context, actor Actor, id uuid.UUID, input UpdateCustomerRequest) (*CustomerDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	current, err := s.findCustomer(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	name := current.Name
	if input.Name != nil {
		value, nameErr := validateName(*input.Name, 120)
		if nameErr != nil {
			return nil, apperror.Validation(map[string]string{"name": "Nama wajib diisi, maksimal 120 karakter."})
		}
		name = value
	}
	phone := current.Phone
	normalized := ""
	if input.Phone != nil {
		var phoneErr error
		phone, normalized, phoneErr = NormalizePhone(*input.Phone)
		if phoneErr != nil {
			return nil, apperror.Validation(map[string]string{"phone": phoneErr.Error()})
		}
		if exists, err := s.phoneExists(ctx, tenantID, normalized, id); err != nil {
			return nil, apperror.Internal(err)
		} else if exists {
			return nil, apperror.Conflict("DUPLICATE_PHONE", "Nomor WhatsApp sudah terdaftar untuk pelanggan lain.")
		}
	}
	address := current.Address
	if input.Address != nil {
		address = trimOptional(input.Address)
	}
	notes := current.Notes
	if input.Notes != nil {
		notes = trimOptional(input.Notes)
	}

	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		query := `UPDATE customers SET name = ?, phone = ?, address = ?, notes = ?, updated_by = ?, updated_at = now()`
		args := []any{name, phone, address, notes, actorID(actor)}
		if normalized != "" {
			query += `, phone_normalized = ?`
			args = append(args, normalized)
		}
		query += ` WHERE id = ? AND tenant_id = ?`
		args = append(args, id, tenantID)
		return tx.Exec(query, args...).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("DUPLICATE_PHONE", "Nomor WhatsApp sudah terdaftar untuk pelanggan lain.")
		}
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "customer.update", Module: "customer", EntityType: "customer", EntityID: &id,
		Before: map[string]any{"name": current.Name, "phone": current.Phone},
		After:  map[string]any{"name": name, "phone": phone},
	})
	return s.GetCustomer(ctx, actor, id)
}

// DeleteCustomer soft-deletes a customer.
func (s *Service) DeleteCustomer(ctx context.Context, actor Actor, id uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	current, err := s.findCustomer(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE customers SET deleted_at = now(), updated_by = ?, updated_at = now()
		WHERE id = ? AND tenant_id = ?`, actorID(actor), id, tenantID).Error; err != nil {
		return apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "customer.delete", Module: "customer", EntityType: "customer", EntityID: &id,
		Before: map[string]any{"name": current.Name, "phone": current.Phone},
	})
	return nil
}

// ListCustomerDevices returns the customer's active devices.
func (s *Service) ListCustomerDevices(ctx context.Context, actor Actor, customerID uuid.UUID) ([]DeviceDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.findCustomer(ctx, tenantID, customerID); err != nil {
		return nil, err
	}
	devices, err := s.devicesForCustomer(ctx, tenantID, customerID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]DeviceDTO, 0, len(devices))
	for _, device := range devices {
		out = append(out, toDeviceDTO(device))
	}
	return out, nil
}

// CreateCustomerDevice adds a device to a customer.
func (s *Service) CreateCustomerDevice(ctx context.Context, actor Actor, customerID uuid.UUID, input CreateDeviceRequest) (*DeviceDTO, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.findCustomer(ctx, tenantID, customerID); err != nil {
		return nil, err
	}
	brand, brandErr := validateName(input.Brand, 60)
	if brandErr != nil {
		return nil, apperror.Validation(map[string]string{"brand": "Merek wajib diisi, maksimal 60 karakter."})
	}
	model, modelErr := validateName(input.Model, 100)
	if modelErr != nil {
		return nil, apperror.Validation(map[string]string{"model": "Model wajib diisi, maksimal 100 karakter."})
	}
	imei := trimOptional(input.IMEI)
	if imei != nil && !onlyDigits(*imei) {
		return nil, apperror.Validation(map[string]string{"imei": "IMEI hanya boleh berisi angka."})
	}
	if imei != nil {
		var exists bool
		if err := s.db.WithContext(ctx).Raw(`
			SELECT EXISTS (SELECT 1 FROM devices
			WHERE tenant_id = ? AND imei = ? AND deleted_at IS NULL)`,
			tenantID, *imei).Scan(&exists).Error; err != nil {
			return nil, apperror.Internal(err)
		}
		if exists {
			return nil, apperror.Conflict("DEVICE_IMEI_EXISTS", "IMEI sudah terdaftar pada perangkat lain.")
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO devices (id, tenant_id, customer_id, brand, model, imei, color, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, customerID, brand, model, imei, trimOptional(input.Color), trimOptional(input.Notes)).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("DEVICE_IMEI_EXISTS", "IMEI sudah terdaftar pada perangkat lain.")
		}
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "customer.device_create", Module: "customer", EntityType: "device", EntityID: &id,
		After: map[string]any{"customer_id": customerID.String(), "brand": brand, "model": model},
	})
	var device Device
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, tenant_id, customer_id, brand, model, imei, color, notes, created_at, updated_at
		FROM devices WHERE id = ? AND tenant_id = ?`, id, tenantID).Scan(&device).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	dto := toDeviceDTO(device)
	return &dto, nil
}

// ListCustomerServices returns the customer's service orders for the active outlet.
func (s *Service) ListCustomerServices(ctx context.Context, actor Actor, customerID uuid.UUID, filter ServiceFilter) (*ServiceOrderList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.findCustomer(ctx, tenantID, customerID); err != nil {
		return nil, err
	}
	filter.CustomerID = &customerID
	return s.listServiceOrders(ctx, actor, filter)
}

// GetCustomerStats returns aggregated customer history.
func (s *Service) GetCustomerStats(ctx context.Context, actor Actor, customerID uuid.UUID) (*CustomerStats, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.findCustomer(ctx, tenantID, customerID); err != nil {
		return nil, err
	}
	stats, err := s.customerStats(ctx, tenantID, customerID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return stats, nil
}

func (s *Service) findCustomer(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error) {
	var customer Customer
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, tenant_id, name, phone, address, notes, last_service_at, created_at, updated_at
		FROM customers WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL`,
		id, tenantID).Scan(&customer).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if customer.ID == uuid.Nil {
		return nil, apperror.NotFound("Pelanggan tidak ditemukan.")
	}
	return &customer, nil
}

func (s *Service) devicesForCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]Device, error) {
	items := make([]Device, 0, 4)
	err := s.db.WithContext(ctx).Raw(`
		SELECT id, tenant_id, customer_id, brand, model, imei, color, notes, created_at, updated_at
		FROM devices
		WHERE tenant_id = ? AND customer_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC`, tenantID, customerID).Scan(&items).Error
	return items, err
}

func (s *Service) customerStats(ctx context.Context, tenantID, customerID uuid.UUID) (*CustomerStats, error) {
	var stats CustomerStats
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*)::bigint AS total_services, max(received_at) AS last_service_at
		FROM service_orders
		WHERE tenant_id = ? AND customer_id = ? AND status <> 'cancelled'`,
		tenantID, customerID).Scan(&stats).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(sum(p.amount), 0)::bigint
		FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		WHERE p.tenant_id = ? AND i.customer_id = ? AND p.status = 'posted'`,
		tenantID, customerID).Scan(&stats.TotalPaid).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *Service) phoneExists(ctx context.Context, tenantID uuid.UUID, normalized string, exclude uuid.UUID) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM customers
		WHERE tenant_id = ? AND phone_normalized = ? AND deleted_at IS NULL`
	args := []any{tenantID, normalized}
	if exclude != uuid.Nil {
		query += ` AND id <> ?`
		args = append(args, exclude)
	}
	query += `)`
	var exists bool
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&exists).Error
	return exists, err
}

func toCustomerDTO(customer Customer) CustomerDTO {
	return CustomerDTO{
		ID:            customer.ID.String(),
		Name:          customer.Name,
		Phone:         customer.Phone,
		Address:       customer.Address,
		Notes:         customer.Notes,
		LastServiceAt: customer.LastServiceAt,
		CreatedAt:     customer.CreatedAt,
		UpdatedAt:     customer.UpdatedAt,
	}
}

func toDeviceDTO(device Device) DeviceDTO {
	return DeviceDTO{
		ID:        device.ID.String(),
		Brand:     device.Brand,
		Model:     device.Model,
		IMEI:      device.IMEI,
		Color:     device.Color,
		Notes:     device.Notes,
		CreatedAt: device.CreatedAt,
		UpdatedAt: device.UpdatedAt,
	}
}

func onlyDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ErrPhoneEmpty is kept for callers that need to distinguish an empty phone.
var ErrPhoneEmpty = errors.New("nomor telepon kosong")
