package warranty

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/docnum"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service implements warranties, claims, and rework orders.
type Service struct {
	db           *gorm.DB
	claimNotifier func(context.Context, *gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID) error
}

// NewService builds the warranty module.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// SetClaimNotifier wires the notification hook used after a claim is created.
func (s *Service) SetClaimNotifier(fn func(context.Context, *gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID) error) {
	s.claimNotifier = fn
}

func (a Actor) hasOutlet(outletID uuid.UUID) bool {
	for _, id := range a.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

func (s *Service) tenant(actor Actor) (uuid.UUID, error) {
	if actor.TenantID == uuid.Nil {
		return uuid.Nil, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return actor.TenantID, nil
}

func (s *Service) expireStale(ctx context.Context, q *gorm.DB, tenantID uuid.UUID) error {
	return q.WithContext(ctx).Exec(`
		UPDATE warranties SET status = 'expired', updated_at = now()
		WHERE tenant_id = ? AND status = 'active' AND end_date < CURRENT_DATE`, tenantID).Error
}

type warrantyRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	WarrantyNo   string    `gorm:"column:warranty_no"`
	Type         string    `gorm:"column:type"`
	Status       string    `gorm:"column:status"`
	StartDate    time.Time `gorm:"column:start_date"`
	EndDate      time.Time `gorm:"column:end_date"`
	Days         int       `gorm:"column:days"`
	Terms        *string   `gorm:"column:terms"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	CustomerID   uuid.UUID `gorm:"column:customer_id"`
	CustomerName string    `gorm:"column:customer_name"`
	CustomerPhone string   `gorm:"column:customer_phone"`
	ServiceOrderID uuid.UUID `gorm:"column:service_order_id"`
	TicketNo     string    `gorm:"column:ticket_no"`
	ServiceStatus string   `gorm:"column:service_status"`
	DeviceBrand  string    `gorm:"column:device_brand"`
	DeviceModel  string    `gorm:"column:device_model"`
	OutletID     uuid.UUID `gorm:"column:outlet_id"`
	OutletName   string    `gorm:"column:outlet_name"`
	ClaimCount   int64     `gorm:"column:claim_count"`
}

const warrantySelect = `
	SELECT w.id, w.warranty_no, w.type, w.status, w.start_date, w.end_date, w.days,
	       w.terms, w.created_at, c.id AS customer_id, c.name AS customer_name,
	       c.phone AS customer_phone, so.id AS service_order_id, so.ticket_no,
	       so.status AS service_status, so.device_brand, so.device_model,
	       o.id AS outlet_id, o.name AS outlet_name,
	       (SELECT COUNT(*) FROM warranty_claims wc WHERE wc.warranty_id = w.id) AS claim_count
	FROM warranties w
	JOIN customers c ON c.id = w.customer_id
	JOIN service_orders so ON so.id = w.service_order_id
	JOIN outlets o ON o.id = w.outlet_id
`

func warrantyFromRow(row warrantyRow) Warranty {
	item := Warranty{
		ID: row.ID.String(), WarrantyNo: row.WarrantyNo, Type: row.Type,
		Status: row.Status, StartDate: row.StartDate.Format("2006-01-02"),
		EndDate: row.EndDate.Format("2006-01-02"), Days: row.Days, Terms: row.Terms,
		Customer: CustomerRef{ID: row.CustomerID.String(), Name: row.CustomerName, Phone: row.CustomerPhone},
		Device:   DeviceRef{Brand: row.DeviceBrand, Model: row.DeviceModel},
		ServiceOrder: ServiceRef{ID: row.ServiceOrderID.String(), TicketNo: row.TicketNo, Status: row.ServiceStatus},
		Outlet:   OutletRef{ID: row.OutletID.String(), Name: row.OutletName},
		ClaimCount: row.ClaimCount, CreatedAt: row.CreatedAt,
	}
	return item
}

// ListWarranties returns tenant-scoped warranties.
func (s *Service) ListWarranties(ctx context.Context, actor Actor, filter WarrantyFilter) ([]Warranty, int64, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, 0, err
	}
	if err := s.expireStale(ctx, s.db, tenantID); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	where := ` WHERE w.tenant_id = ?`
	args := []any{tenantID}
	if !actor.AllOutletAccess {
		if len(actor.OutletIDs) == 0 {
			return []Warranty{}, 0, nil
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(actor.OutletIDs)), ",")
		where += ` AND w.outlet_id IN (` + placeholders + `)`
		for _, id := range actor.OutletIDs {
			args = append(args, id)
		}
	}
	if filter.OutletID != nil {
		if !actor.hasOutlet(*filter.OutletID) {
			return nil, 0, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		where += ` AND w.outlet_id = ?`
		args = append(args, *filter.OutletID)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		where += ` AND w.status = ?`
		args = append(args, status)
	}
	if filter.ExpiringInDays > 0 {
		where += ` AND w.status = 'active' AND w.end_date BETWEEN CURRENT_DATE AND CURRENT_DATE + ?`
		args = append(args, filter.ExpiringInDays)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		where += ` AND (w.warranty_no ILIKE ? OR so.ticket_no ILIKE ? OR c.name ILIKE ? OR so.device_brand ILIKE ? OR so.device_model ILIKE ?)`
		args = append(args, like, like, like, like, like)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM warranties w
		JOIN customers c ON c.id = w.customer_id
		JOIN service_orders so ON so.id = w.service_order_id` + where
	if err := s.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows := make([]warrantyRow, 0, perPage)
	query := warrantySelect + where + ` ORDER BY w.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, perPage, (page-1)*perPage)
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}
	items := make([]Warranty, 0, len(rows))
	for _, row := range rows {
		items = append(items, warrantyFromRow(row))
	}
	return items, total, nil
}

// GetWarranty returns one warranty with claims and invoice.
func (s *Service) GetWarranty(ctx context.Context, actor Actor, id uuid.UUID) (*Warranty, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, err
	}
	if err := s.expireStale(ctx, s.db, tenantID); err != nil {
		return nil, apperror.Internal(err)
	}
	var row warrantyRow
	if err := s.db.WithContext(ctx).Raw(warrantySelect+`
		WHERE w.id = ? AND w.tenant_id = ?`, id, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil || !actor.hasOutlet(row.OutletID) {
		return nil, apperror.NotFound("Garansi tidak ditemukan.")
	}
	item := warrantyFromRow(row)

	var invoice InvoiceRef
	if err := s.db.WithContext(ctx).Raw(`
		SELECT i.id, i.invoice_no, i.status, i.total_amount, i.paid_amount, i.remaining_amount
		FROM invoices i WHERE i.tenant_id = ? AND i.service_order_id = ?`,
		tenantID, row.ServiceOrderID).Scan(&invoice).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if strings.TrimSpace(invoice.ID) != "" {
		item.Invoice = &invoice
	}
	claims, _, err := s.ListClaims(ctx, actor, ClaimFilter{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	item.Claims = make([]Claim, 0)
	for _, claim := range claims {
		if claim.WarrantyID == item.ID {
			item.Claims = append(item.Claims, claim)
		}
	}
	return &item, nil
}

type claimRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	ClaimNo        string     `gorm:"column:claim_no"`
	WarrantyID     uuid.UUID  `gorm:"column:warranty_id"`
	WarrantyNo     string     `gorm:"column:warranty_no"`
	Status         string     `gorm:"column:status"`
	Complaint      string     `gorm:"column:complaint"`
	Resolution     *string    `gorm:"column:resolution"`
	CustomerID     uuid.UUID  `gorm:"column:customer_id"`
	CustomerName   string     `gorm:"column:customer_name"`
	CustomerPhone  string     `gorm:"column:customer_phone"`
	ServiceOrderID uuid.UUID  `gorm:"column:service_order_id"`
	TicketNo       string     `gorm:"column:ticket_no"`
	ServiceStatus  string     `gorm:"column:service_status"`
	DeviceBrand    string     `gorm:"column:device_brand"`
	DeviceModel    string     `gorm:"column:device_model"`
	OutletID       uuid.UUID  `gorm:"column:outlet_id"`
	OutletName     string     `gorm:"column:outlet_name"`
	HandledBy      *string    `gorm:"column:handled_by"`
	ReworkID       *uuid.UUID `gorm:"column:rework_service_order_id"`
	ReworkTicketNo *string    `gorm:"column:rework_ticket_no"`
	ReworkStatus   *string    `gorm:"column:rework_status"`
	ClaimedAt      time.Time  `gorm:"column:claimed_at"`
	ResolvedAt     *time.Time `gorm:"column:resolved_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

const claimSelect = `
	SELECT wc.id, wc.claim_no, wc.warranty_id, w.warranty_no, wc.status, wc.complaint,
	       wc.resolution, c.id AS customer_id, c.name AS customer_name, c.phone AS customer_phone,
	       so.id AS service_order_id, so.ticket_no, so.status AS service_status,
	       so.device_brand, so.device_model, o.id AS outlet_id, o.name AS outlet_name,
	       u.name AS handled_by, wc.rework_service_order_id,
	       rso.ticket_no AS rework_ticket_no, rso.status AS rework_status,
	       wc.claimed_at, wc.resolved_at, wc.created_at, wc.updated_at
	FROM warranty_claims wc
	JOIN warranties w ON w.id = wc.warranty_id
	JOIN service_orders so ON so.id = w.service_order_id
	JOIN customers c ON c.id = w.customer_id
	JOIN outlets o ON o.id = wc.outlet_id
	LEFT JOIN users u ON u.id = wc.handled_by
	LEFT JOIN service_orders rso ON rso.id = wc.rework_service_order_id
`

func claimFromRow(row claimRow) Claim {
	item := Claim{
		ID: row.ID.String(), ClaimNo: row.ClaimNo, WarrantyID: row.WarrantyID.String(),
		WarrantyNo: row.WarrantyNo, Status: row.Status, Complaint: row.Complaint,
		Resolution: row.Resolution,
		Customer: CustomerRef{ID: row.CustomerID.String(), Name: row.CustomerName, Phone: row.CustomerPhone},
		Device: DeviceRef{Brand: row.DeviceBrand, Model: row.DeviceModel},
		ServiceOrder: ServiceRef{ID: row.ServiceOrderID.String(), TicketNo: row.TicketNo, Status: row.ServiceStatus},
		Outlet: OutletRef{ID: row.OutletID.String(), Name: row.OutletName},
		HandledBy: row.HandledBy, ClaimedAt: row.ClaimedAt, ResolvedAt: row.ResolvedAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.ReworkID != nil && row.ReworkTicketNo != nil {
		status := ""
		if row.ReworkStatus != nil {
			status = *row.ReworkStatus
		}
		item.ReworkServiceOrder = &ServiceRef{ID: row.ReworkID.String(), TicketNo: *row.ReworkTicketNo, Status: status}
	}
	return item
}

// CreateClaim records a new claim against an active warranty.
func (s *Service) CreateClaim(ctx context.Context, actor Actor, warrantyID uuid.UUID, input CreateClaimRequest) (*Claim, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, err
	}
	complaint := strings.TrimSpace(input.Complaint)
	if complaint == "" {
		return nil, apperror.Validation(map[string]string{"complaint": "Keluhan klaim wajib diisi."})
	}
	if len(complaint) > 2000 {
		return nil, apperror.Validation(map[string]string{"complaint": "Keluhan klaim maksimal 2000 karakter."})
	}

	var claimID uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID         uuid.UUID `gorm:"column:id"`
			OutletID   uuid.UUID `gorm:"column:outlet_id"`
			Status     string    `gorm:"column:status"`
			EndDate    time.Time `gorm:"column:end_date"`
			WarrantyNo string    `gorm:"column:warranty_no"`
		}
		if err := tx.Raw(`
			SELECT id, outlet_id, status, end_date, warranty_no FROM warranties
			WHERE id = ? AND tenant_id = ? FOR UPDATE`, warrantyID, tenantID).Scan(&row).Error; err != nil {
			return apperror.Internal(err)
		}
		if row.ID == uuid.Nil || !actor.hasOutlet(row.OutletID) {
			return apperror.NotFound("Garansi tidak ditemukan.")
		}
		if row.Status != "active" || row.EndDate.Before(dateOnly(time.Now())) {
			_ = tx.Exec(`UPDATE warranties SET status = 'expired', updated_at = now() WHERE id = ?`, warrantyID).Error
			return apperror.Conflict("WARRANTY_INACTIVE", "Garansi sudah tidak aktif atau telah berakhir.")
		}

		number, err := docnum.Next(ctx, tx, tenantID, row.OutletID, "claim", "CLM", "{PREFIX}-{YYMMDD}-{SEQ:4}", time.Now())
		if err != nil {
			return apperror.Internal(err)
		}
		id, err := uuid.NewV7()
		if err != nil {
			return apperror.Internal(err)
		}
		if err := tx.Exec(`
			INSERT INTO warranty_claims (
				id, tenant_id, outlet_id, warranty_id, claim_no, complaint, status, handled_by, claimed_at
			) VALUES (?, ?, ?, ?, ?, ?, 'submitted', ?, now())`,
			id, tenantID, row.OutletID, warrantyID, number, complaint, actorID(actor)).Error; err != nil {
			return apperror.Internal(err)
		}
		claimID = id
		if s.claimNotifier != nil {
			if err := s.claimNotifier(ctx, tx, tenantID, row.OutletID, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetClaim(ctx, actor, claimID)
}

// ListClaims returns tenant-scoped claims.
func (s *Service) ListClaims(ctx context.Context, actor Actor, filter ClaimFilter) ([]Claim, int64, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, 0, err
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	where := ` WHERE wc.tenant_id = ?`
	args := []any{tenantID}
	if !actor.AllOutletAccess {
		if len(actor.OutletIDs) == 0 {
			return []Claim{}, 0, nil
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(actor.OutletIDs)), ",")
		where += ` AND wc.outlet_id IN (` + placeholders + `)`
		for _, id := range actor.OutletIDs {
			args = append(args, id)
		}
	}
	if filter.OutletID != nil {
		if !actor.hasOutlet(*filter.OutletID) {
			return nil, 0, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		where += ` AND wc.outlet_id = ?`
		args = append(args, *filter.OutletID)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		where += ` AND wc.status = ?`
		args = append(args, status)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		where += ` AND (wc.claim_no ILIKE ? OR w.warranty_no ILIKE ? OR so.ticket_no ILIKE ? OR c.name ILIKE ?)`
		args = append(args, like, like, like, like)
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM warranty_claims wc
		JOIN warranties w ON w.id = wc.warranty_id
		JOIN service_orders so ON so.id = w.service_order_id
		JOIN customers c ON c.id = w.customer_id`+where, args...).Scan(&total).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}
	rows := make([]claimRow, 0, perPage)
	args = append(args, perPage, (page-1)*perPage)
	if err := s.db.WithContext(ctx).Raw(claimSelect+where+`
		ORDER BY wc.created_at DESC LIMIT ? OFFSET ?`, args...).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}
	items := make([]Claim, 0, len(rows))
	for _, row := range rows {
		items = append(items, claimFromRow(row))
	}
	return items, total, nil
}

// GetClaim returns one claim.
func (s *Service) GetClaim(ctx context.Context, actor Actor, id uuid.UUID) (*Claim, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, err
	}
	var row claimRow
	if err := s.db.WithContext(ctx).Raw(claimSelect+`
		WHERE wc.id = ? AND wc.tenant_id = ?`, id, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil || !actor.hasOutlet(row.OutletID) {
		return nil, apperror.NotFound("Klaim garansi tidak ditemukan.")
	}
	item := claimFromRow(row)
	return &item, nil
}

var claimTransitions = map[string]map[string]bool{
	ClaimSubmitted:  {ClaimApproved: true, ClaimRejected: true, ClaimInProgress: true},
	ClaimApproved:   {ClaimInProgress: true, ClaimCompleted: true, ClaimRejected: true},
	ClaimInProgress: {ClaimCompleted: true, ClaimRejected: true},
	ClaimCompleted:  {},
	ClaimRejected:   {},
}

// UpdateClaim changes claim status/resolution and records the handler.
func (s *Service) UpdateClaim(ctx context.Context, actor Actor, id uuid.UUID, input UpdateClaimRequest) (*Claim, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, err
	}
	var resolution *string
	if input.Resolution != nil {
		value := strings.TrimSpace(*input.Resolution)
		if len(value) > 2000 {
			return nil, apperror.Validation(map[string]string{"resolution": "Resolusi maksimal 2000 karakter."})
		}
		if value != "" {
			resolution = &value
		}
	}
	target := ""
	if input.Status != nil {
		target = strings.ToLower(strings.TrimSpace(*input.Status))
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID       uuid.UUID `gorm:"column:id"`
			OutletID uuid.UUID `gorm:"column:outlet_id"`
			Status   string    `gorm:"column:status"`
			Resolution *string `gorm:"column:resolution"`
		}
		if err := tx.Raw(`
			SELECT id, outlet_id, status, resolution FROM warranty_claims
			WHERE id = ? AND tenant_id = ? FOR UPDATE`, id, tenantID).Scan(&row).Error; err != nil {
			return apperror.Internal(err)
		}
		if row.ID == uuid.Nil || !actor.hasOutlet(row.OutletID) {
			return apperror.NotFound("Klaim garansi tidak ditemukan.")
		}
		nextStatus := row.Status
		if target != "" && target != row.Status {
			if !claimTransitions[row.Status][target] {
				return apperror.Conflict("INVALID_CLAIM_TRANSITION",
					"Perubahan status klaim dari '"+row.Status+"' ke '"+target+"' tidak diizinkan.")
			}
			nextStatus = target
		}
		if resolution != nil {
			row.Resolution = resolution
		}
		if (nextStatus == ClaimCompleted || nextStatus == ClaimRejected) && (row.Resolution == nil || strings.TrimSpace(*row.Resolution) == "") {
			return apperror.Validation(map[string]string{"resolution": "Resolusi wajib diisi saat klaim selesai atau ditolak."})
		}
		resolved := nextStatus == ClaimCompleted || nextStatus == ClaimRejected
		return tx.Exec(`
			UPDATE warranty_claims
			SET status = ?, resolution = ?, handled_by = ?, resolved_at = CASE WHEN ? THEN now() ELSE resolved_at END,
			    updated_at = now()
			WHERE id = ? AND tenant_id = ?`,
			nextStatus, row.Resolution, actorID(actor), resolved, id, tenantID).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetClaim(ctx, actor, id)
}

// CreateRework creates a linked service order for an approved claim.
func (s *Service) CreateRework(ctx context.Context, actor Actor, claimID uuid.UUID) (*ReworkResult, error) {
	tenantID, err := s.tenant(actor)
	if err != nil {
		return nil, err
	}
	var result ReworkResult
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var claim struct {
			ID             uuid.UUID  `gorm:"column:id"`
			ClaimNo        string     `gorm:"column:claim_no"`
			OutletID       uuid.UUID  `gorm:"column:outlet_id"`
			Status         string     `gorm:"column:status"`
			Complaint      string     `gorm:"column:complaint"`
			WarrantyID     uuid.UUID  `gorm:"column:warranty_id"`
			ServiceOrderID uuid.UUID  `gorm:"column:service_order_id"`
			ReworkID       *uuid.UUID `gorm:"column:rework_service_order_id"`
			CustomerID     uuid.UUID  `gorm:"column:customer_id"`
			DeviceID       *uuid.UUID `gorm:"column:device_id"`
			DeviceBrand    string     `gorm:"column:device_brand"`
			DeviceModel    string     `gorm:"column:device_model"`
			DeviceIMEI     *string    `gorm:"column:device_imei"`
			DeviceColor    *string    `gorm:"column:device_color"`
		}
		if err := tx.Raw(`
			SELECT wc.id, wc.claim_no, wc.outlet_id, wc.status, wc.complaint,
			       wc.warranty_id, w.service_order_id, wc.rework_service_order_id,
			       so.customer_id, so.device_id, so.device_brand, so.device_model,
			       so.device_imei, so.device_color
			FROM warranty_claims wc
			JOIN warranties w ON w.id = wc.warranty_id
			JOIN service_orders so ON so.id = w.service_order_id
			WHERE wc.id = ? AND wc.tenant_id = ? FOR UPDATE`, claimID, tenantID).Scan(&claim).Error; err != nil {
			return apperror.Internal(err)
		}
		if claim.ID == uuid.Nil || !actor.hasOutlet(claim.OutletID) {
			return apperror.NotFound("Klaim garansi tidak ditemukan.")
		}
		if claim.ReworkID != nil {
			return apperror.Conflict("REWORK_EXISTS", "Klaim ini sudah memiliki order pengerjaan ulang.")
		}
		switch claim.Status {
		case ClaimSubmitted, ClaimApproved, ClaimInProgress:
		default:
			return apperror.Conflict("CLAIM_NOT_APPROVED", "Klaim harus disetujui sebelum membuat order pengerjaan ulang.")
		}

		settings, err := s.documentSettings(ctx, tx, tenantID, claim.OutletID)
		if err != nil {
			return apperror.Internal(err)
		}
		now := time.Now().UTC()
		ticketNo, err := docnum.Next(ctx, tx, tenantID, claim.OutletID, "service_order",
			settings.TicketPrefix, settings.TicketFormat, now)
		if err != nil {
			return apperror.Internal(err)
		}
		invoiceNo, err := docnum.Next(ctx, tx, tenantID, claim.OutletID, "invoice",
			settings.InvoicePrefix, settings.InvoiceFormat, now)
		if err != nil {
			return apperror.Internal(err)
		}
		token, err := newPublicToken()
		if err != nil {
			return err
		}
		orderID, err := uuid.NewV7()
		if err != nil {
			return apperror.Internal(err)
		}
		complaint := "Klaim garansi " + claim.ClaimNo + ": " + claim.Complaint
		if len(complaint) > 1000 {
			complaint = complaint[:1000]
		}
		if err := tx.Exec(`
			INSERT INTO service_orders (
				id, tenant_id, outlet_id, ticket_no, public_token, public_token_expires_at,
				customer_id, device_id, status, priority, device_brand, device_model,
				device_imei, device_color, complaint, estimated_cost, order_type,
				rework_of_service_order_id, warranty_claim_id, created_by, updated_by
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'received', 'normal', ?, ?, ?, ?, ?, 0,
			          'rework', ?, ?, ?, ?)`,
			orderID, tenantID, claim.OutletID, ticketNo, token, now.AddDate(0, 0, 30),
			claim.CustomerID, claim.DeviceID, claim.DeviceBrand, claim.DeviceModel,
			claim.DeviceIMEI, claim.DeviceColor, complaint, claim.ServiceOrderID, claim.ID,
			actorID(actor), actorID(actor)).Error; err != nil {
			return apperror.Internal(err)
		}
		if err := tx.Exec(`
			INSERT INTO service_status_history (
				tenant_id, service_order_id, from_status, to_status, notes, changed_by
			) VALUES (?, ?, NULL, 'received', ?, ?)`,
			tenantID, orderID, "Order pengerjaan ulang dari klaim "+claim.ClaimNo, actorID(actor)).Error; err != nil {
			return apperror.Internal(err)
		}
		invoiceID, err := uuid.NewV7()
		if err != nil {
			return apperror.Internal(err)
		}
		if err := tx.Exec(`
			INSERT INTO invoices (id, tenant_id, outlet_id, service_order_id, invoice_no, customer_id, status)
			VALUES (?, ?, ?, ?, ?, ?, 'draft')`,
			invoiceID, tenantID, claim.OutletID, orderID, invoiceNo, claim.CustomerID).Error; err != nil {
			return apperror.Internal(err)
		}
		if err := tx.Exec(`
			UPDATE warranty_claims
			SET status = 'in_progress', rework_service_order_id = ?, handled_by = ?, updated_at = now()
			WHERE id = ? AND tenant_id = ?`,
			orderID, actorID(actor), claim.ID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		result = ReworkResult{
			ServiceOrderID: orderID.String(), TicketNo: ticketNo, Status: "received",
			ClaimID: claim.ID.String(), ClaimStatus: ClaimInProgress,
			InvoiceNo: invoiceNo, CreatedAt: now,
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "warranty.rework", Module: "warranty", EntityType: "service_order",
			EntityID: &orderID,
			After: map[string]any{
				"ticket_no": ticketNo, "claim_no": claim.ClaimNo,
				"rework_of_service_order_id": claim.ServiceOrderID.String(),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type documentSettings struct {
	TicketPrefix  string `gorm:"column:ticket_prefix"`
	TicketFormat  string `gorm:"column:ticket_format"`
	InvoicePrefix string `gorm:"column:invoice_prefix"`
	InvoiceFormat string `gorm:"column:invoice_format"`
	Timezone      string `gorm:"column:timezone"`
}

func (s *Service) documentSettings(ctx context.Context, q *gorm.DB, tenantID, outletID uuid.UUID) (documentSettings, error) {
	settings := documentSettings{}
	err := q.WithContext(ctx).Raw(`
		SELECT ticket_prefix, ticket_format, invoice_prefix, invoice_format, timezone
		FROM business_settings
		WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
		ORDER BY (outlet_id IS NULL)
		LIMIT 1`, tenantID, outletID).Scan(&settings).Error
	if strings.TrimSpace(settings.TicketPrefix) == "" {
		settings.TicketPrefix = "FP"
	}
	if strings.TrimSpace(settings.TicketFormat) == "" {
		settings.TicketFormat = "{PREFIX}-{YYMMDD}-{SEQ:4}"
	}
	if strings.TrimSpace(settings.InvoicePrefix) == "" {
		settings.InvoicePrefix = "INV"
	}
	if strings.TrimSpace(settings.InvoiceFormat) == "" {
		settings.InvoiceFormat = "{PREFIX}-{YYMMDD}-{SEQ:4}"
	}
	return settings, err
}

// CreateForPickupTx creates the warranty exactly once when an order is picked up.
func (s *Service) CreateForPickupTx(ctx context.Context, tx *gorm.DB, tenantID, outletID, orderID, customerID uuid.UUID, pickedUpAt time.Time) error {
	var invoice struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT id FROM invoices WHERE tenant_id = ? AND service_order_id = ?`,
		tenantID, orderID).Scan(&invoice).Error; err != nil {
		return err
	}
	invoiceID := invoice.ID
	if invoiceID == uuid.Nil {
		// A warranty requires an invoice row; pickup without an invoice is a
		// data-integrity problem rather than something this hook can repair.
		return nil
	}

	type totals struct {
		ServiceDays  int `gorm:"column:service_days"`
		PartDays     int `gorm:"column:part_days"`
		ServiceCount int `gorm:"column:service_count"`
		PartCount    int `gorm:"column:part_count"`
	}
	var itemTotals totals
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(MAX(warranty_days) FILTER (WHERE item_type IN ('service', 'custom')), 0) AS service_days,
		       COALESCE(MAX(warranty_days) FILTER (WHERE item_type = 'sparepart'), 0) AS part_days,
		       COUNT(*) FILTER (WHERE item_type IN ('service', 'custom')) AS service_count,
		       COUNT(*) FILTER (WHERE item_type = 'sparepart') AS part_count
		FROM invoice_items WHERE tenant_id = ? AND invoice_id = ?`,
		tenantID, invoiceID).Scan(&itemTotals).Error; err != nil {
		return err
	}
	if itemTotals.ServiceCount == 0 && itemTotals.PartCount == 0 {
		if err := tx.WithContext(ctx).Raw(`
			SELECT COALESCE(MAX(warranty_days) FILTER (WHERE item_type IN ('service', 'custom')), 0) AS service_days,
			       COALESCE(MAX(warranty_days) FILTER (WHERE item_type = 'sparepart'), 0) AS part_days,
			       COUNT(*) FILTER (WHERE item_type IN ('service', 'custom')) AS service_count,
			       COUNT(*) FILTER (WHERE item_type = 'sparepart') AS part_count
			FROM service_order_items WHERE tenant_id = ? AND service_order_id = ?`,
			tenantID, orderID).Scan(&itemTotals).Error; err != nil {
			return err
		}
	}

	var defaults struct {
		ServiceDays int    `gorm:"column:default_service_warranty_days"`
		PartDays    int    `gorm:"column:default_part_warranty_days"`
		Timezone    string `gorm:"column:timezone"`
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT default_service_warranty_days, default_part_warranty_days, timezone
		FROM business_settings
		WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
		ORDER BY (outlet_id IS NULL)
		LIMIT 1`, tenantID, outletID).Scan(&defaults).Error; err != nil {
		return err
	}
	serviceDays := itemTotals.ServiceDays
	if serviceDays <= 0 {
		serviceDays = defaults.ServiceDays
	}
	partDays := itemTotals.PartDays
	if itemTotals.PartCount > 0 && partDays <= 0 {
		partDays = defaults.PartDays
	}
	if serviceDays < 0 {
		serviceDays = 0
	}
	if partDays < 0 {
		partDays = 0
	}
	kind := "service"
	days := serviceDays
	switch {
	case serviceDays > 0 && partDays > 0:
		kind = "both"
		if partDays > days {
			days = partDays
		}
	case serviceDays == 0 && partDays > 0:
		kind = "part"
		days = partDays
	}
	if days == 0 {
		kind = "service"
		days = 30
	}

	loc := time.UTC
	if strings.TrimSpace(defaults.Timezone) != "" {
		if loaded, loadErr := time.LoadLocation(defaults.Timezone); loadErr == nil {
			loc = loaded
		}
	}
	local := pickedUpAt.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, days)

	number, err := docnum.Next(ctx, tx, tenantID, outletID, "warranty", "WR", "{PREFIX}-{YYMMDD}-{SEQ:4}", pickedUpAt)
	if err != nil {
		return err
	}
	warrantyID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO warranties (
			id, tenant_id, outlet_id, service_order_id, invoice_id, customer_id,
			warranty_no, type, start_date, end_date, days, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active')
		ON CONFLICT (service_order_id) DO NOTHING`,
		warrantyID, tenantID, outletID, orderID, invoiceID, customerID,
		number, kind, start, end, days).Error
}

func actorID(actor Actor) *uuid.UUID {
	if actor.UserID == uuid.Nil {
		return nil
	}
	id := actor.UserID
	return &id
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// newPublicToken returns the 32-character public tracking token.
func newPublicToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", apperror.Internal(err)
	}
	return hex.EncodeToString(buffer), nil
}
