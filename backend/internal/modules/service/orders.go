package service

import (
	"context"
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

type serviceOrderRow struct {
	ID              uuid.UUID  `gorm:"column:id"`
	TenantID        uuid.UUID  `gorm:"column:tenant_id"`
	OutletID        uuid.UUID  `gorm:"column:outlet_id"`
	TicketNo        string     `gorm:"column:ticket_no"`
	Status          string     `gorm:"column:status"`
	Priority        string     `gorm:"column:priority"`
	CustomerID      uuid.UUID  `gorm:"column:customer_id"`
	CustomerName    string     `gorm:"column:customer_name"`
	CustomerPhone   string     `gorm:"column:customer_phone"`
	DeviceID        *uuid.UUID `gorm:"column:device_id"`
	DeviceBrand     string     `gorm:"column:device_brand"`
	DeviceModel     string     `gorm:"column:device_model"`
	DeviceIMEI      *string    `gorm:"column:device_imei"`
	DeviceColor     *string    `gorm:"column:device_color"`
	TechnicianID    *uuid.UUID `gorm:"column:technician_id"`
	TechnicianName  *string    `gorm:"column:technician_name"`
	Complaint       string     `gorm:"column:complaint"`
	EstimatedCost   int64      `gorm:"column:estimated_cost"`
	EstimatedDoneAt *time.Time `gorm:"column:estimated_done_at"`
	ReceivedAt      time.Time  `gorm:"column:received_at"`
	CompletedAt     *time.Time `gorm:"column:completed_at"`
	PickedUpAt      *time.Time `gorm:"column:picked_up_at"`
	CredentialType  string     `gorm:"column:credential_type"`
	PhotoCount      int64      `gorm:"column:photo_count"`
}

type serviceOrderDetailRow struct {
	// NOTE: GORM's Raw().Scan() only maps fields directly on this struct —
	// embedded structs (anonymous or via the "embedded" tag) silently scan as
	// zero values on gorm v1.25.x. Keep this row flat.
	ID                uuid.UUID  `gorm:"column:id"`
	TenantID          uuid.UUID  `gorm:"column:tenant_id"`
	OutletID          uuid.UUID  `gorm:"column:outlet_id"`
	TicketNo          string     `gorm:"column:ticket_no"`
	Status            string     `gorm:"column:status"`
	Priority          string     `gorm:"column:priority"`
	CustomerID        uuid.UUID  `gorm:"column:customer_id"`
	CustomerName      string     `gorm:"column:customer_name"`
	CustomerPhone     string     `gorm:"column:customer_phone"`
	DeviceID          *uuid.UUID `gorm:"column:device_id"`
	DeviceBrand       string     `gorm:"column:device_brand"`
	DeviceModel       string     `gorm:"column:device_model"`
	DeviceIMEI        *string    `gorm:"column:device_imei"`
	DeviceColor       *string    `gorm:"column:device_color"`
	TechnicianID      *uuid.UUID `gorm:"column:technician_id"`
	TechnicianName    *string    `gorm:"column:technician_name"`
	Complaint         string     `gorm:"column:complaint"`
	EstimatedCost     int64      `gorm:"column:estimated_cost"`
	EstimatedDoneAt   *time.Time `gorm:"column:estimated_done_at"`
	ReceivedAt        time.Time  `gorm:"column:received_at"`
	CompletedAt       *time.Time `gorm:"column:completed_at"`
	PickedUpAt        *time.Time `gorm:"column:picked_up_at"`
	CredentialType    string     `gorm:"column:credential_type"`
	PhotoCount        int64      `gorm:"column:photo_count"`
	Completeness      *string    `gorm:"column:completeness"`
	PhysicalCondition *string    `gorm:"column:physical_condition"`
	Notes             *string    `gorm:"column:notes"`
	InternalNotes     *string    `gorm:"column:internal_notes"`
	PublicToken       *string    `gorm:"column:public_token"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

// toOrderRow rebuilds the flat list-row projection from a detail row.
func (r serviceOrderDetailRow) toOrderRow() serviceOrderRow {
	return serviceOrderRow{
		ID: r.ID, TenantID: r.TenantID, OutletID: r.OutletID,
		TicketNo: r.TicketNo, Status: r.Status, Priority: r.Priority,
		CustomerID: r.CustomerID, CustomerName: r.CustomerName, CustomerPhone: r.CustomerPhone,
		DeviceID: r.DeviceID, DeviceBrand: r.DeviceBrand, DeviceModel: r.DeviceModel,
		DeviceIMEI: r.DeviceIMEI, DeviceColor: r.DeviceColor,
		TechnicianID: r.TechnicianID, TechnicianName: r.TechnicianName,
		Complaint: r.Complaint, EstimatedCost: r.EstimatedCost,
		EstimatedDoneAt: r.EstimatedDoneAt, ReceivedAt: r.ReceivedAt,
		CompletedAt: r.CompletedAt, PickedUpAt: r.PickedUpAt,
		CredentialType: r.CredentialType, PhotoCount: r.PhotoCount,
	}
}

// CreateServiceOrder performs the full intake in one transaction.
func (s *Service) CreateServiceOrder(ctx context.Context, actor Actor, input CreateServiceRequest) (*ServiceOrderCreated, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	outletID, err := actor.outletFor(nil)
	if err != nil {
		return nil, err
	}
	if ok, err := s.outletExists(ctx, tenantID, outletID); err != nil {
		return nil, apperror.Internal(err)
	} else if !ok {
		return nil, apperror.NotFound("Outlet tidak ditemukan.")
	}

	complaint := strings.TrimSpace(input.Complaint)
	if complaint == "" {
		return nil, apperror.Validation(map[string]string{"complaint": "Keluhan pelanggan wajib diisi."})
	}
	if len(complaint) > 1000 {
		return nil, apperror.Validation(map[string]string{"complaint": "Keluhan maksimal 1000 karakter."})
	}
	priority := strings.TrimSpace(input.Priority)
	if priority == "" {
		priority = PriorityNormal
	}
	if priority != PriorityNormal && priority != PriorityExpress {
		return nil, apperror.Validation(map[string]string{"priority": "Prioritas harus normal atau express."})
	}
	if input.EstimatedCost < 0 {
		return nil, apperror.Validation(map[string]string{"estimated_cost": "Estimasi biaya tidak boleh negatif."})
	}
	estimatedDoneAt, err := parseTimePointer(input.EstimatedDoneAt)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"estimated_done_at": "Format estimasi selesai tidak valid."})
	}

	brand, brandErr := validateName(input.Device.Brand, 60)
	if brandErr != nil {
		return nil, apperror.Validation(map[string]string{"device.brand": "Merek perangkat wajib diisi, maksimal 60 karakter."})
	}
	model, modelErr := validateName(input.Device.Model, 100)
	if modelErr != nil {
		return nil, apperror.Validation(map[string]string{"device.model": "Model perangkat wajib diisi, maksimal 100 karakter."})
	}
	imei := trimOptional(input.Device.IMEI)
	if imei != nil && !onlyDigits(*imei) {
		return nil, apperror.Validation(map[string]string{"device.imei": "IMEI hanya boleh berisi angka."})
	}
	if imei != nil && len(*imei) > 20 {
		return nil, apperror.Validation(map[string]string{"device.imei": "IMEI maksimal 20 karakter."})
	}

	var technicianID *uuid.UUID
	if input.AssignedTechnicianID != nil && strings.TrimSpace(*input.AssignedTechnicianID) != "" {
		id, parseErr := uuid.Parse(strings.TrimSpace(*input.AssignedTechnicianID))
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"assigned_technician_id": "ID teknisi tidak valid."})
		}
		ok, checkErr := s.technicianExists(ctx, tenantID, outletID, id)
		if checkErr != nil {
			return nil, apperror.Internal(checkErr)
		}
		if !ok {
			return nil, apperror.Validation(map[string]string{"assigned_technician_id": "Teknisi tidak ditemukan atau tidak ditugaskan ke outlet ini."})
		}
		technicianID = &id
	}

	credentialType := CredentialNone
	var credentialEnc []byte
	var credentialSetAt *time.Time
	if input.Credential != nil && strings.TrimSpace(input.Credential.Value) != "" {
		if !validCredentialType(input.Credential.Type) {
			return nil, apperror.Validation(map[string]string{"credential.type": "Tipe kredensial harus pin, password, atau pattern."}) // #nosec G101 -- teks validasi untuk pengguna, bukan kredensial hardcoded.
		}
		if !validCredentialValue(input.Credential.Value) {
			return nil, apperror.Validation(map[string]string{"credential.value": "Kredensial wajib diisi, maksimal 200 karakter."}) // #nosec G101 -- teks validasi untuk pengguna, bukan kredensial hardcoded.
		}
		credentialEnc, err = encryptCredential(s.opts.EncKey, input.Credential.Value)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		credentialType = input.Credential.Type
		now := time.Now().UTC()
		credentialSetAt = &now
	}

	loc, err := s.outletTimezone(ctx, tenantID, outletID)
	if err != nil {
		return nil, err
	}
	settings, err := s.settingsFor(ctx, tenantID, outletID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	now := time.Now().In(loc)

	ticketID, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	publicToken, err := newPublicToken()
	if err != nil {
		return nil, err
	}

	var result ServiceOrderCreated
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		customerID, txErr := s.resolveCustomer(ctx, tx, actor, input.Customer)
		if txErr != nil {
			return txErr
		}
		deviceID, txErr := s.resolveDevice(ctx, tx, tenantID, customerID, input.Device, brand, model, imei)
		if txErr != nil {
			return txErr
		}

		ticketNo, txErr := s.nextDocumentNumber(ctx, tx, tenantID, outletID, "service_order",
			settings.TicketPrefix, settings.TicketFormat, now)
		if txErr != nil {
			return txErr
		}
		invoiceNo, txErr := s.nextDocumentNumber(ctx, tx, tenantID, outletID, "invoice",
			settings.InvoicePrefix, settings.InvoiceFormat, now)
		if txErr != nil {
			return txErr
		}

		tokenExpires := now.UTC().AddDate(0, 0, 30)
		if txErr := tx.Exec(`
			INSERT INTO service_orders (
				id, tenant_id, outlet_id, ticket_no, public_token, public_token_expires_at,
				customer_id, device_id, assigned_technician_id, status, priority,
				device_brand, device_model, device_imei, device_color,
				completeness, physical_condition, complaint, notes, internal_notes,
				estimated_cost, estimated_done_at, credential_type, credential_enc,
				credential_key_version, credential_set_at, created_by, updated_by
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ticketID, tenantID, outletID, ticketNo, publicToken, tokenExpires,
			customerID, deviceID, technicianID, StatusReceived, priority,
			brand, model, imei, trimOptional(input.Device.Color),
			trimOptional(input.Completeness), trimOptional(input.PhysicalCondition),
			complaint, trimOptional(input.Notes), trimOptional(input.InternalNotes),
			input.EstimatedCost, estimatedDoneAt, credentialType, credentialEnc,
			nullableKeyVersion(credentialType), credentialSetAt, actorID(actor), actorID(actor),
		).Error; txErr != nil {
			return txErr
		}

		if txErr := tx.Exec(`
			INSERT INTO service_status_history (tenant_id, service_order_id, from_status, to_status, notes, changed_by)
			VALUES (?, ?, NULL, ?, 'Servis diterima', ?)`,
			tenantID, ticketID, StatusReceived, actorID(actor)).Error; txErr != nil {
			return txErr
		}
		if txErr := tx.Exec(`UPDATE customers SET last_service_at = ?, updated_at = now() WHERE id = ?`,
			now.UTC(), customerID).Error; txErr != nil {
			return txErr
		}

		invoiceID, txErr := uuid.NewV7()
		if txErr != nil {
			return txErr
		}
		if txErr := tx.Exec(`
			INSERT INTO invoices (id, tenant_id, outlet_id, service_order_id, invoice_no, customer_id, status)
			VALUES (?, ?, ?, ?, ?, ?, 'draft')`,
			invoiceID, tenantID, outletID, ticketID, invoiceNo, customerID).Error; txErr != nil {
			return txErr
		}

		result = ServiceOrderCreated{
			ID:          ticketID.String(),
			TicketNo:    ticketNo,
			PublicToken: publicToken,
			TrackingURL: s.trackingURL(publicToken),
			Status:      StatusReceived,
			CustomerID:  customerID.String(),
			DeviceID:    deviceID.String(),
			InvoiceNo:   invoiceNo,
			CreatedAt:   now.UTC(),
		}
		return nil
	})
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("CONFLICT", "Nomor tiket atau invoice bentrok. Silakan coba lagi.")
		}
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.create", Module: "service", EntityType: "service_order", EntityID: &ticketID,
		After: map[string]any{
			"ticket_no": result.TicketNo, "status": result.Status,
			"customer_id": result.CustomerID, "device_id": result.DeviceID,
			"invoice_no": result.InvoiceNo, "estimated_cost": input.EstimatedCost,
		},
	})
	return &result, nil
}

func nullableKeyVersion(credentialType string) *int {
	if credentialType == CredentialNone {
		return nil
	}
	version := credentialKeyVersion
	return &version
}

func (s *Service) resolveCustomer(ctx context.Context, tx *gorm.DB, actor Actor, input ServiceCustomerInput) (uuid.UUID, error) {
	tenantID := actor.TenantID
	if input.ID != nil && strings.TrimSpace(*input.ID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*input.ID))
		if err != nil {
			return uuid.Nil, apperror.Validation(map[string]string{"customer.id": "ID pelanggan tidak valid."})
		}
		var customer Customer
		if err := tx.WithContext(ctx).Raw(`
			SELECT id FROM customers
			WHERE id = ? AND tenant_id = ? AND deleted_at IS NULL`, id, tenantID).Scan(&customer).Error; err != nil {
			return uuid.Nil, err
		}
		if customer.ID == uuid.Nil {
			return uuid.Nil, apperror.NotFound("Pelanggan tidak ditemukan.")
		}
		return customer.ID, nil
	}

	name, nameErr := validateName(input.Name, 120)
	if nameErr != nil {
		return uuid.Nil, apperror.Validation(map[string]string{"customer.name": "Nama pelanggan wajib diisi, maksimal 120 karakter."})
	}
	displayPhone, normalized, phoneErr := NormalizePhone(input.Phone)
	if phoneErr != nil {
		return uuid.Nil, apperror.Validation(map[string]string{"customer.phone": phoneErr.Error()})
	}

	var existing Customer
	if err := tx.WithContext(ctx).Raw(`
		SELECT id FROM customers
		WHERE tenant_id = ? AND phone_normalized = ? AND deleted_at IS NULL`,
		tenantID, normalized).Scan(&existing).Error; err != nil {
		return uuid.Nil, err
	}
	if existing.ID != uuid.Nil {
		return existing.ID, nil
	}

	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Exec(`
		INSERT INTO customers (id, tenant_id, name, phone, phone_normalized, address, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, name, displayPhone, normalized, trimOptional(input.Address), actorID(actor), actorID(actor)).Error; err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict("DUPLICATE_PHONE", "Nomor WhatsApp sudah terdaftar untuk pelanggan lain.")
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Service) resolveDevice(
	ctx context.Context,
	tx *gorm.DB,
	tenantID, customerID uuid.UUID,
	input ServiceDeviceInput,
	brand, model string,
	imei *string,
) (uuid.UUID, error) {
	if imei != nil {
		var device Device
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, customer_id FROM devices
			WHERE tenant_id = ? AND imei = ? AND deleted_at IS NULL`, tenantID, *imei).Scan(&device).Error; err != nil {
			return uuid.Nil, err
		}
		if device.ID != uuid.Nil {
			if device.CustomerID != customerID {
				return uuid.Nil, apperror.Conflict("DEVICE_IMEI_EXISTS", "IMEI sudah terdaftar pada pelanggan lain.")
			}
			return device.ID, nil
		}
	} else {
		var device Device
		if err := tx.WithContext(ctx).Raw(`
			SELECT id FROM devices
			WHERE tenant_id = ? AND customer_id = ? AND brand = ? AND model = ? AND deleted_at IS NULL
			ORDER BY created_at DESC LIMIT 1`, tenantID, customerID, brand, model).Scan(&device).Error; err != nil {
			return uuid.Nil, err
		}
		if device.ID != uuid.Nil {
			return device.ID, nil
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Exec(`
		INSERT INTO devices (id, tenant_id, customer_id, brand, model, imei, color)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, customerID, brand, model, imei, trimOptional(input.Color)).Error; err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict("DEVICE_IMEI_EXISTS", "IMEI sudah terdaftar pada perangkat lain.")
		}
		return uuid.Nil, err
	}
	return id, nil
}

// ListServiceOrders returns a paginated service-order list for one outlet.
func (s *Service) ListServiceOrders(ctx context.Context, actor Actor, filter ServiceFilter) (*ServiceOrderList, error) {
	return s.listServiceOrders(ctx, actor, filter)
}

func (s *Service) listServiceOrders(ctx context.Context, actor Actor, filter ServiceFilter) (*ServiceOrderList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var requested *uuid.UUID
	if filter.OutletID != uuid.Nil {
		id := filter.OutletID
		requested = &id
	}
	outletID, err := actor.outletFor(requested)
	if err != nil {
		return nil, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}

	where := ` WHERE so.tenant_id = ? AND so.outlet_id = ?`
	args := []any{tenantID, outletID}

	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + query + "%"
		where += ` AND (so.ticket_no ILIKE ? OR c.name ILIKE ? OR c.phone ILIKE ? OR so.device_brand ILIKE ? OR so.device_model ILIKE ?)`
		args = append(args, like, like, like, like, like)
	}
	if len(filter.Statuses) > 0 {
		for _, status := range filter.Statuses {
			if _, ok := allowedStatuses[status]; !ok {
				return nil, apperror.Validation(map[string]string{"status": "Status servis tidak dikenal."})
			}
		}
		where += ` AND so.status IN ?`
		args = append(args, filter.Statuses)
	}
	if filter.TechnicianID != nil {
		where += ` AND so.assigned_technician_id = ?`
		args = append(args, *filter.TechnicianID)
	}
	if filter.CustomerID != nil {
		where += ` AND so.customer_id = ?`
		args = append(args, *filter.CustomerID)
	}

	loc, err := s.outletTimezone(ctx, tenantID, outletID)
	if err != nil {
		return nil, err
	}
	var from, to *time.Time
	if strings.TrimSpace(filter.FromDate) != "" {
		value, parseErr := time.ParseInLocation("2006-01-02", strings.TrimSpace(filter.FromDate), loc)
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"from": "Format tanggal awal harus YYYY-MM-DD."})
		}
		from = &value
	}
	if strings.TrimSpace(filter.ToDate) != "" {
		value, parseErr := time.ParseInLocation("2006-01-02", strings.TrimSpace(filter.ToDate), loc)
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"to": "Format tanggal akhir harus YYYY-MM-DD."})
		}
		end := value.AddDate(0, 0, 1)
		to = &end
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, apperror.Validation(map[string]string{"from": "Tanggal awal tidak boleh setelah tanggal akhir."})
	}
	if from != nil {
		where += ` AND so.received_at >= ?`
		args = append(args, *from)
	}
	if to != nil {
		where += ` AND so.received_at < ?`
		args = append(args, *to)
	}
	if actor.Role == "technician" && s.technicianOnlyAssigned(ctx, tenantID, outletID) {
		where += ` AND so.assigned_technician_id = ?`
		args = append(args, actor.UserID)
	}

	base := `
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		LEFT JOIN users u ON u.id = so.assigned_technician_id`

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*)`+base+where, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	rows := make([]serviceOrderRow, 0, filter.PerPage)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT so.id, so.outlet_id, so.ticket_no, so.status, so.priority,
		       c.id AS customer_id, c.name AS customer_name, c.phone AS customer_phone,
		       so.device_id, so.device_brand, so.device_model, so.device_imei, so.device_color,
		       u.id AS technician_id, u.name AS technician_name,
		       so.complaint, so.estimated_cost, so.estimated_done_at, so.received_at,
		       so.completed_at, so.picked_up_at, so.credential_type,
		       (SELECT count(*) FROM service_photos p WHERE p.service_order_id = so.id)::bigint AS photo_count`+
		base+where+`
		ORDER BY so.received_at DESC, so.ticket_no DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]ServiceOrderListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toListItem(row))
	}
	return &ServiceOrderList{Items: items, Total: total}, nil
}

// GetServiceOrder returns full order detail.
func (s *Service) GetServiceOrder(ctx context.Context, actor Actor, id uuid.UUID) (*ServiceOrderDetail, error) {
	row, err := s.findOrder(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	photos, err := s.photosForOrder(ctx, row.TenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	timeline, err := s.timelineForOrder(ctx, row.TenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	invoice, err := s.invoiceForOrder(ctx, row.TenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	// Diagnosis notes are internal to roles with diagnosis.view.
	var diagnostic *DiagnosticDTO
	if actor.HasPermission("diagnosis.view") {
		diagnostic, err = s.diagnosticForOrder(ctx, row.TenantID, id)
		if err != nil {
			return nil, apperror.Internal(err)
		}
	}
	items, err := s.itemsForOrder(ctx, row.TenantID, id, actor.HasPermission("inventory.cost.view"))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	store, err := s.storeFor(ctx, row.TenantID, row.OutletID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	detail := &ServiceOrderDetail{
		ServiceOrderListItem: toListItem(row.toOrderRow()),
		Completeness:         row.Completeness,
		PhysicalCondition:    row.PhysicalCondition,
		Notes:                row.Notes,
		InternalNotes:        row.InternalNotes,
		CredentialType:       row.CredentialType,
		Photos:               photos,
		Timeline:             timeline,
		Invoice:              invoice,
		Diagnostic:           diagnostic,
		Items:                items,
		Store:                store,
		PublicToken:          row.PublicToken,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	if row.PublicToken != nil {
		url := s.trackingURL(*row.PublicToken)
		detail.PublicTrackingURL = &url
	}
	return detail, nil
}

func (s *Service) findOrder(ctx context.Context, actor Actor, id uuid.UUID) (*serviceOrderDetailRow, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var row serviceOrderDetailRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT so.id, so.tenant_id, so.outlet_id, so.ticket_no, so.status, so.priority,
		       c.id AS customer_id, c.name AS customer_name, c.phone AS customer_phone,
		       so.device_id, so.device_brand, so.device_model, so.device_imei, so.device_color,
		       u.id AS technician_id, u.name AS technician_name,
		       so.complaint, so.estimated_cost, so.estimated_done_at, so.received_at,
		       so.completed_at, so.picked_up_at, so.credential_type,
		       (SELECT count(*) FROM service_photos p WHERE p.service_order_id = so.id)::bigint AS photo_count,
		       so.completeness, so.physical_condition, so.notes, so.internal_notes,
		       so.public_token, so.created_at, so.updated_at
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		LEFT JOIN users u ON u.id = so.assigned_technician_id
		WHERE so.id = ? AND so.tenant_id = ?`, id, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Servis tidak ditemukan.")
	}
	if !actor.HasOutlet(row.OutletID) {
		return nil, apperror.NotFound("Servis tidak ditemukan.")
	}
	if actor.Role == "technician" && s.technicianOnlyAssigned(ctx, tenantID, row.OutletID) {
		if row.TechnicianID == nil || *row.TechnicianID != actor.UserID {
			return nil, apperror.NotFound("Servis tidak ditemukan.")
		}
	}
	return &row, nil
}

// UpdateServiceOrder edits intake fields without changing the status.
func (s *Service) UpdateServiceOrder(ctx context.Context, actor Actor, id uuid.UUID, input UpdateServiceRequest) (*ServiceOrderDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	row, err := s.findOrder(ctx, actor, id)
	if err != nil {
		return nil, err
	}

	complaint := row.Complaint
	if input.Complaint != nil {
		complaint = strings.TrimSpace(*input.Complaint)
		if complaint == "" || len(complaint) > 1000 {
			return nil, apperror.Validation(map[string]string{"complaint": "Keluhan wajib diisi, maksimal 1000 karakter."})
		}
	}
	priority := row.Priority
	if input.Priority != nil {
		priority = strings.TrimSpace(*input.Priority)
		if priority != PriorityNormal && priority != PriorityExpress {
			return nil, apperror.Validation(map[string]string{"priority": "Prioritas harus normal atau express."})
		}
	}
	estimatedCost := row.EstimatedCost
	if input.EstimatedCost != nil {
		if *input.EstimatedCost < 0 {
			return nil, apperror.Validation(map[string]string{"estimated_cost": "Estimasi biaya tidak boleh negatif."})
		}
		estimatedCost = *input.EstimatedCost
	}
	estimatedDoneAt := row.EstimatedDoneAt
	if input.EstimatedDoneAt != nil {
		if strings.TrimSpace(*input.EstimatedDoneAt) == "" {
			estimatedDoneAt = nil
		} else {
			value, parseErr := parseTimePointer(input.EstimatedDoneAt)
			if parseErr != nil {
				return nil, apperror.Validation(map[string]string{"estimated_done_at": "Format estimasi selesai tidak valid."})
			}
			estimatedDoneAt = value
		}
	}
	completeness := row.Completeness
	if input.Completeness != nil {
		completeness = trimOptional(input.Completeness)
	}
	physicalCondition := row.PhysicalCondition
	if input.PhysicalCondition != nil {
		physicalCondition = trimOptional(input.PhysicalCondition)
	}
	notes := row.Notes
	if input.Notes != nil {
		notes = trimOptional(input.Notes)
	}
	internalNotes := row.InternalNotes
	if input.InternalNotes != nil {
		internalNotes = trimOptional(input.InternalNotes)
	}

	technicianID := row.TechnicianID
	if input.AssignedTechnicianID != nil {
		raw := strings.TrimSpace(*input.AssignedTechnicianID)
		if raw == "" {
			technicianID = nil
		} else {
			parsed, parseErr := uuid.Parse(raw)
			if parseErr != nil {
				return nil, apperror.Validation(map[string]string{"assigned_technician_id": "ID teknisi tidak valid."})
			}
			ok, checkErr := s.technicianExists(ctx, tenantID, row.OutletID, parsed)
			if checkErr != nil {
				return nil, apperror.Internal(checkErr)
			}
			if !ok {
				return nil, apperror.Validation(map[string]string{"assigned_technician_id": "Teknisi tidak ditemukan atau tidak ditugaskan ke outlet ini."})
			}
			technicianID = &parsed
		}
	}

	if err := s.db.WithContext(ctx).Exec(`
		UPDATE service_orders
		SET complaint = ?, completeness = ?, physical_condition = ?, notes = ?, internal_notes = ?,
		    estimated_cost = ?, estimated_done_at = ?, priority = ?,
		    assigned_technician_id = ?, updated_by = ?, updated_at = now()
		WHERE id = ? AND tenant_id = ?`,
		complaint, completeness, physicalCondition, notes, internalNotes,
		estimatedCost, estimatedDoneAt, priority, technicianID, actorID(actor), id, tenantID).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.update", Module: "service", EntityType: "service_order", EntityID: &id,
		Before: map[string]any{"complaint": row.Complaint, "priority": row.Priority, "estimated_cost": row.EstimatedCost},
		After:  map[string]any{"complaint": complaint, "priority": priority, "estimated_cost": estimatedCost},
	})
	return s.GetServiceOrder(ctx, actor, id)
}

// PublicLink returns (and regenerates when expired) the public tracking link.
func (s *Service) PublicLink(ctx context.Context, actor Actor, orderID uuid.UUID) (*PublicLink, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var row struct {
		ID          uuid.UUID  `gorm:"column:id"`
		OutletID    uuid.UUID  `gorm:"column:outlet_id"`
		TicketNo    string     `gorm:"column:ticket_no"`
		PublicToken *string    `gorm:"column:public_token"`
		ExpiresAt   *time.Time `gorm:"column:public_token_expires_at"`
		PickedUpAt  *time.Time `gorm:"column:picked_up_at"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, outlet_id, ticket_no, public_token, public_token_expires_at, picked_up_at
		FROM service_orders WHERE id = ? AND tenant_id = ?`, orderID, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Servis tidak ditemukan.")
	}
	if !actor.HasOutlet(row.OutletID) {
		return nil, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
	}

	now := time.Now().UTC()
	token := ""
	if row.PublicToken != nil {
		token = strings.TrimSpace(*row.PublicToken)
	}
	expires := now
	if row.ExpiresAt != nil {
		expires = row.ExpiresAt.UTC()
	}
	if token != "" && expires.After(now) {
		return &PublicLink{
			ServiceOrderID: row.ID.String(), TicketNo: row.TicketNo,
			PublicToken: token, TrackingURL: s.trackingURL(token),
			ExpiresAt: expires, Regenerated: false,
		}, nil
	}

	token, err = newPublicToken()
	if err != nil {
		return nil, err
	}
	days := 30
	if row.PickedUpAt != nil {
		days = 90
	}
	expires = now.AddDate(0, 0, days)
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE service_orders
		SET public_token = ?, public_token_expires_at = ?, updated_at = now()
		WHERE id = ? AND tenant_id = ?`, token, expires, orderID, tenantID).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return &PublicLink{
		ServiceOrderID: row.ID.String(), TicketNo: row.TicketNo,
		PublicToken: token, TrackingURL: s.trackingURL(token),
		ExpiresAt: expires, Regenerated: true,
	}, nil
}

// Timeline returns the immutable status history.
func (s *Service) Timeline(ctx context.Context, actor Actor, id uuid.UUID) ([]StatusHistoryEntry, error) {
	row, err := s.findOrder(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	entries, err := s.timelineForOrder(ctx, row.TenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return entries, nil
}

// SetCredential stores an encrypted device credential.
func (s *Service) SetCredential(ctx context.Context, actor Actor, id uuid.UUID, input CredentialRequest) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	row, err := s.findOrder(ctx, actor, id)
	if err != nil {
		return err
	}
	if row.Status == "picked_up" || row.Status == "cancelled" {
		return apperror.Conflict("INVALID_STATUS_TRANSITION", "Kredensial tidak dapat diubah pada status ini.")
	}
	if !validCredentialType(input.Type) {
		return apperror.Validation(map[string]string{"type": "Tipe kredensial harus pin, password, atau pattern."})
	}
	if !validCredentialValue(input.Value) {
		return apperror.Validation(map[string]string{"value": "Kredensial wajib diisi, maksimal 200 karakter."})
	}
	encrypted, err := encryptCredential(s.opts.EncKey, input.Value)
	if err != nil {
		return apperror.Internal(err)
	}
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE service_orders
		SET credential_type = ?, credential_enc = ?, credential_key_version = ?,
		    credential_set_at = now(), updated_by = ?, updated_at = now()
		WHERE id = ? AND tenant_id = ?`,
		input.Type, encrypted, credentialKeyVersion, actorID(actor), id, tenantID).Error; err != nil {
		return apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.credential_set", Module: "service", EntityType: "service_order", EntityID: &id,
		After: map[string]any{"credential_type": input.Type},
	})
	return nil
}

// DeleteCredential removes a stored credential.
func (s *Service) DeleteCredential(ctx context.Context, actor Actor, id uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	row, err := s.findOrder(ctx, actor, id)
	if err != nil {
		return err
	}
	if row.CredentialType == CredentialNone {
		return apperror.NotFound("Kredensial tidak ditemukan.")
	}
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE service_orders
		SET credential_type = 'none', credential_enc = NULL, credential_key_version = NULL,
		    credential_set_at = NULL, updated_by = ?, updated_at = now()
		WHERE id = ? AND tenant_id = ?`, actorID(actor), id, tenantID).Error; err != nil {
		return apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.credential_delete", Module: "service", EntityType: "service_order", EntityID: &id,
	})
	return nil
}

// RevealCredential decrypts a credential and records the access.
func (s *Service) RevealCredential(ctx context.Context, actor Actor, id uuid.UUID, reason string) (*CredentialReveal, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	row, err := s.findOrder(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if row.CredentialType == CredentialNone {
		return nil, apperror.NotFound("Kredensial tidak tersimpan untuk servis ini.")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, apperror.Validation(map[string]string{"reason": "Alasan pengungkapan wajib diisi."})
	}
	var payload []byte
	var credentialType string
	if err := s.db.WithContext(ctx).Raw(`
		SELECT credential_type, credential_enc FROM service_orders
		WHERE id = ? AND tenant_id = ?`, id, tenantID).Row().Scan(&credentialType, &payload); err != nil {
		return nil, apperror.Internal(err)
	}
	if len(payload) == 0 {
		return nil, apperror.NotFound("Kredensial tidak tersimpan untuk servis ini.")
	}
	value, err := decryptCredential(s.opts.EncKey, payload)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
		Action: "service.credential_reveal", Module: "service", EntityType: "service_order", EntityID: &id,
		After: map[string]any{"credential_type": credentialType, "reason": strings.TrimSpace(reason)},
	})
	return &CredentialReveal{Type: credentialType, Value: value}, nil
}

// CredentialReveal is the write-only reveal response.
type CredentialReveal struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Technicians lists active technicians available to a tenant/outlet.
func (s *Service) Technicians(ctx context.Context, actor Actor, outletID *uuid.UUID) ([]TechnicianOption, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	target, err := actor.outletFor(outletID)
	if err != nil {
		return nil, err
	}
	items := make([]TechnicianOption, 0, 8)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT u.id::text AS id, u.name
		FROM users u
		JOIN roles r ON r.id = u.role_id AND r.slug = 'technician'
		WHERE u.tenant_id = ? AND u.deleted_at IS NULL AND u.is_active = true
		  AND (NOT EXISTS (SELECT 1 FROM user_outlets x WHERE x.user_id = u.id)
		       OR EXISTS (SELECT 1 FROM user_outlets uo WHERE uo.user_id = u.id AND uo.outlet_id = ?))
		ORDER BY u.name`, tenantID, target).Scan(&items).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return items, nil
}

func (s *Service) photosForOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]ServicePhoto, error) {
	items := make([]ServicePhoto, 0, 4)
	err := s.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, stage, file_path, mime_type, size_bytes, is_public, created_at
		FROM service_photos
		WHERE tenant_id = ? AND service_order_id = ?
		ORDER BY created_at, id`, tenantID, orderID).Scan(&items).Error
	return items, err
}

func (s *Service) timelineForOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]StatusHistoryEntry, error) {
	type row struct {
		ID          uuid.UUID  `gorm:"column:id"`
		FromStatus  *string    `gorm:"column:from_status"`
		ToStatus    string     `gorm:"column:to_status"`
		Notes       *string    `gorm:"column:notes"`
		ChangedByID *uuid.UUID `gorm:"column:changed_by_id"`
		ChangedBy   *string    `gorm:"column:changed_by"`
		CreatedAt   time.Time  `gorm:"column:created_at"`
	}
	rows := make([]row, 0, 4)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT h.id, h.from_status, h.to_status, h.notes,
		       h.changed_by AS changed_by_id, u.name AS changed_by, h.created_at
		FROM service_status_history h
		LEFT JOIN users u ON u.id = h.changed_by
		WHERE h.tenant_id = ? AND h.service_order_id = ?
		ORDER BY h.created_at, h.id`, tenantID, orderID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]StatusHistoryEntry, 0, len(rows))
	for _, item := range rows {
		entry := StatusHistoryEntry{
			ID: item.ID.String(), FromStatus: item.FromStatus, ToStatus: item.ToStatus,
			Notes: item.Notes, ChangedBy: item.ChangedBy, CreatedAt: item.CreatedAt,
		}
		if item.ChangedByID != nil {
			id := item.ChangedByID.String()
			entry.ChangedByID = &id
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Service) invoiceForOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*InvoiceRef, error) {
	var invoice InvoiceRef
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, invoice_no, status, total_amount, paid_amount, remaining_amount
		FROM invoices WHERE tenant_id = ? AND service_order_id = ?`, tenantID, orderID).Scan(&invoice).Error; err != nil {
		return nil, err
	}
	if invoice.ID == "" {
		return nil, nil
	}
	return &invoice, nil
}

func (s *Service) storeFor(ctx context.Context, tenantID, outletID uuid.UUID) (StoreRef, error) {
	var store StoreRef
	if err := s.db.WithContext(ctx).Raw(`
		SELECT store_name AS name, address, phone
		FROM business_settings
		WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
		ORDER BY (outlet_id IS NULL)
		LIMIT 1`, tenantID, outletID).Scan(&store).Error; err != nil {
		return store, err
	}
	if strings.TrimSpace(store.Name) == "" {
		store.Name = "FixPhone"
	}
	return store, nil
}

func (s *Service) trackingURL(token string) string {
	base := strings.TrimRight(strings.TrimSpace(s.opts.AppBaseURL), "/")
	if base == "" {
		return ""
	}
	return fmt.Sprintf("%s/track/%s", base, token)
}

func toListItem(row serviceOrderRow) ServiceOrderListItem {
	item := ServiceOrderListItem{
		ID:                   row.ID.String(),
		TicketNo:             row.TicketNo,
		Status:               row.Status,
		Priority:             row.Priority,
		Customer:             CustomerRef{ID: row.CustomerID.String(), Name: row.CustomerName, Phone: row.CustomerPhone},
		Device:               DeviceRef{Brand: row.DeviceBrand, Model: row.DeviceModel, IMEI: row.DeviceIMEI, Color: row.DeviceColor},
		Complaint:            row.Complaint,
		EstimatedCost:        row.EstimatedCost,
		EstimatedDoneAt:      row.EstimatedDoneAt,
		ReceivedAt:           row.ReceivedAt,
		CompletedAt:          row.CompletedAt,
		PickedUpAt:           row.PickedUpAt,
		CredentialConfigured: row.CredentialType != CredentialNone,
		PhotoCount:           row.PhotoCount,
	}
	if row.DeviceID != nil {
		item.Device.ID = row.DeviceID.String()
	}
	if row.TechnicianID != nil && row.TechnicianName != nil {
		item.AssignedTechnician = &TechnicianRef{ID: row.TechnicianID.String(), Name: *row.TechnicianName}
	}
	return item
}
