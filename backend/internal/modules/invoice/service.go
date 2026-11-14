package invoice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var allowedInvoiceStatuses = map[string]struct{}{
	StatusDraft: {}, StatusUnpaid: {}, StatusPartial: {}, StatusPaid: {}, StatusVoid: {},
}

// Service implements invoice reads and finalisation.
type Service struct {
	db *gorm.DB
}

// NewService builds the invoice module.
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

func parseDateBoundary(raw string, endOfDay bool, loc *time.Location) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		utc := value.UTC()
		return &utc, nil
	}
	value, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		value = value.AddDate(0, 0, 1)
	}
	utc := value.UTC()
	return &utc, nil
}

func calculateDiscount(subtotal int64, discountType string, value int64) (int64, error) {
	if value < 0 {
		return 0, apperror.Validation(map[string]string{"discount_value": "Nilai diskon tidak boleh negatif."})
	}
	switch discountType {
	case "percent":
		if value > 100 {
			return 0, apperror.Validation(map[string]string{"discount_value": "Diskon persen maksimal 100."})
		}
		return (subtotal*value + 50) / 100, nil
	case "amount":
		if value > subtotal {
			return 0, apperror.Validation(map[string]string{"discount_value": "Diskon tidak boleh melebihi subtotal."})
		}
		return value, nil
	default:
		return 0, apperror.Validation(map[string]string{"discount_type": "Tipe diskon harus amount atau percent."})
	}
}

type invoiceRow struct {
	ID              uuid.UUID  `gorm:"column:id"`
	TenantID        uuid.UUID  `gorm:"column:tenant_id"`
	OutletID        uuid.UUID  `gorm:"column:outlet_id"`
	ServiceOrderID  uuid.UUID  `gorm:"column:service_order_id"`
	InvoiceNo       string     `gorm:"column:invoice_no"`
	CustomerID      uuid.UUID  `gorm:"column:customer_id"`
	CustomerName    string     `gorm:"column:customer_name"`
	CustomerPhone   string     `gorm:"column:customer_phone"`
	OutletName      string     `gorm:"column:outlet_name"`
	Status          string     `gorm:"column:status"`
	SubtotalService int64      `gorm:"column:subtotal_service"`
	SubtotalPart    int64      `gorm:"column:subtotal_part"`
	DiscountType    string     `gorm:"column:discount_type"`
	DiscountValue   int64      `gorm:"column:discount_value"`
	DiscountAmount  int64      `gorm:"column:discount_amount"`
	TaxAmount       int64      `gorm:"column:tax_amount"`
	TotalAmount     int64      `gorm:"column:total_amount"`
	PaidAmount      int64      `gorm:"column:paid_amount"`
	RemainingAmount int64      `gorm:"column:remaining_amount"`
	IssuedAt        *time.Time `gorm:"column:issued_at"`
	DueAt           *time.Time `gorm:"column:due_at"`
	Notes           *string    `gorm:"column:notes"`
	VoidReason      *string    `gorm:"column:void_reason"`
	VoidedBy        *string    `gorm:"column:voided_by_name"`
	VoidedAt        *time.Time `gorm:"column:voided_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`

	TicketNo       string     `gorm:"column:ticket_no"`
	ServiceStatus  string     `gorm:"column:service_status"`
	Priority       string     `gorm:"column:priority"`
	DeviceBrand    string     `gorm:"column:device_brand"`
	DeviceModel    string     `gorm:"column:device_model"`
	DeviceIMEI     *string    `gorm:"column:device_imei"`
	DeviceColor    *string    `gorm:"column:device_color"`
	Complaint      string     `gorm:"column:complaint"`
	ReceivedAt     time.Time  `gorm:"column:received_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
	PickedUpAt     *time.Time `gorm:"column:picked_up_at"`
	TechnicianName *string    `gorm:"column:technician_name"`
	PublicToken    *string    `gorm:"column:public_token"`
}

func toListItem(row invoiceRow) InvoiceListItem {
	return InvoiceListItem{
		ID: row.ID.String(), InvoiceNo: row.InvoiceNo, Status: row.Status,
		ServiceOrderID: row.ServiceOrderID.String(), TicketNo: row.TicketNo,
		ServiceStatus: row.ServiceStatus,
		CustomerID:    row.CustomerID.String(), CustomerName: row.CustomerName,
		CustomerPhone: row.CustomerPhone,
		OutletID:      row.OutletID.String(), OutletName: row.OutletName,
		SubtotalService: row.SubtotalService, SubtotalPart: row.SubtotalPart,
		DiscountType: row.DiscountType, DiscountValue: row.DiscountValue,
		DiscountAmount: row.DiscountAmount, TaxAmount: row.TaxAmount,
		TotalAmount: row.TotalAmount, PaidAmount: row.PaidAmount,
		RemainingAmount: row.RemainingAmount,
		IssuedAt:        row.IssuedAt, DueAt: row.DueAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

const invoiceBaseQuery = `
	FROM invoices i
	JOIN service_orders so ON so.id = i.service_order_id
	JOIN customers c ON c.id = i.customer_id
	JOIN outlets o ON o.id = i.outlet_id
	LEFT JOIN users u ON u.id = so.assigned_technician_id
	LEFT JOIN users vu ON vu.id = i.voided_by`

const invoiceSelectColumns = `
	i.id, i.tenant_id, i.outlet_id, i.service_order_id, i.invoice_no,
	i.customer_id, c.name AS customer_name, c.phone AS customer_phone,
	o.name AS outlet_name, i.status, i.subtotal_service, i.subtotal_part,
	i.discount_type, i.discount_value, i.discount_amount, i.tax_amount,
	i.total_amount, i.paid_amount, i.remaining_amount,
	i.issued_at, i.due_at, i.notes, i.void_reason,
	vu.name AS voided_by_name, i.voided_at, i.created_at, i.updated_at,
	so.ticket_no, so.status AS service_status, so.priority,
	so.device_brand, so.device_model, so.device_imei, so.device_color,
	so.complaint, so.received_at, so.completed_at, so.picked_up_at,
	u.name AS technician_name, so.public_token`

// ListInvoices returns invoices for the active outlet.
func (s *Service) ListInvoices(ctx context.Context, actor Actor, filter InvoiceFilter) (*InvoiceList, error) {
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
	for _, status := range filter.Statuses {
		if _, ok := allowedInvoiceStatuses[status]; !ok {
			return nil, apperror.Validation(map[string]string{"status": "Status invoice tidak dikenal."})
		}
	}

	loc, err := s.outletTimezone(ctx, tenantID, outletID)
	if err != nil {
		return nil, err
	}
	from, err := parseDateBoundary(filter.FromDate, false, loc)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"from": "Format tanggal awal harus YYYY-MM-DD."})
	}
	to, err := parseDateBoundary(filter.ToDate, true, loc)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"to": "Format tanggal akhir harus YYYY-MM-DD."})
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, apperror.Validation(map[string]string{"from": "Tanggal awal tidak boleh setelah tanggal akhir."})
	}

	where := ` WHERE i.tenant_id = ? AND i.outlet_id = ?`
	args := []any{tenantID, outletID}
	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + query + "%"
		where += ` AND (i.invoice_no ILIKE ? OR so.ticket_no ILIKE ? OR c.name ILIKE ? OR c.phone ILIKE ?)`
		args = append(args, like, like, like, like)
	}
	if len(filter.Statuses) > 0 {
		where += ` AND i.status IN ?`
		args = append(args, filter.Statuses)
	}
	if from != nil {
		where += ` AND i.created_at >= ?`
		args = append(args, *from)
	}
	if to != nil {
		where += ` AND i.created_at < ?`
		args = append(args, *to)
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*)`+invoiceBaseQuery+where, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	rows := make([]invoiceRow, 0, filter.PerPage)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT `+invoiceSelectColumns+invoiceBaseQuery+where+`
		ORDER BY i.created_at DESC, i.invoice_no DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]InvoiceListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toListItem(row))
	}
	return &InvoiceList{Items: items, Total: total}, nil
}

// GetInvoice returns full invoice detail, including snapshots and payments.
func (s *Service) GetInvoice(ctx context.Context, actor Actor, id uuid.UUID) (*InvoiceDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var row invoiceRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT `+invoiceSelectColumns+invoiceBaseQuery+`
		WHERE i.id = ? AND i.tenant_id = ?`, id, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Invoice tidak ditemukan.")
	}
	if !actor.HasOutlet(row.OutletID) {
		return nil, apperror.NotFound("Invoice tidak ditemukan.")
	}

	items, err := s.invoiceItems(ctx, tenantID, id, actor.HasPermission("inventory.cost.view"))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	payments, err := s.invoicePayments(ctx, tenantID, id)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	store, err := s.storeFor(ctx, tenantID, row.OutletID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &InvoiceDetail{
		InvoiceListItem: toListItem(row),
		Notes:           row.Notes,
		VoidReason:      row.VoidReason,
		VoidedBy:        row.VoidedBy,
		VoidedAt:        row.VoidedAt,
		Order: ServiceRef{
			ID: row.ServiceOrderID.String(), TicketNo: row.TicketNo,
			Status: row.ServiceStatus, Priority: row.Priority,
			DeviceBrand: row.DeviceBrand, DeviceModel: row.DeviceModel,
			DeviceIMEI: row.DeviceIMEI, DeviceColor: row.DeviceColor,
			Complaint: row.Complaint, ReceivedAt: row.ReceivedAt,
			CompletedAt: row.CompletedAt, PickedUpAt: row.PickedUpAt,
			TechnicianName: row.TechnicianName,
		},
		Items:    items,
		Payments: payments,
		Store:    store,
	}, nil
}

// GetInvoiceByServiceOrder resolves the invoice attached to an order.
func (s *Service) GetInvoiceByServiceOrder(ctx context.Context, actor Actor, orderID uuid.UUID) (*InvoiceDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id FROM invoices WHERE service_order_id = ? AND tenant_id = ?`,
		orderID, tenantID).Scan(&row).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return nil, apperror.NotFound("Invoice tidak ditemukan untuk servis ini.")
	}
	return s.GetInvoice(ctx, actor, row.ID)
}

func (s *Service) invoiceItems(ctx context.Context, tenantID, invoiceID uuid.UUID, canViewCost bool) ([]InvoiceItem, error) {
	type row struct {
		ID             uuid.UUID  `gorm:"column:id"`
		ItemType       string     `gorm:"column:item_type"`
		SparepartID    *uuid.UUID `gorm:"column:sparepart_id"`
		Description    string     `gorm:"column:description"`
		Qty            float64    `gorm:"column:qty"`
		UnitPrice      int64      `gorm:"column:unit_price"`
		CostPrice      int64      `gorm:"column:cost_price"`
		DiscountAmount int64      `gorm:"column:discount_amount"`
		Subtotal       int64      `gorm:"column:subtotal"`
		WarrantyDays   int        `gorm:"column:warranty_days"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, item_type, sparepart_id, description, qty, unit_price,
		       cost_price, discount_amount, subtotal, warranty_days
		FROM invoice_items
		WHERE invoice_id = ? AND tenant_id = ?
		ORDER BY created_at, id`, invoiceID, tenantID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]InvoiceItem, 0, len(rows))
	for _, r := range rows {
		var sparepartID *string
		if r.SparepartID != nil {
			value := r.SparepartID.String()
			sparepartID = &value
		}
		cost := int64(0)
		if canViewCost {
			cost = r.CostPrice
		}
		items = append(items, InvoiceItem{
			ID: r.ID.String(), ItemType: r.ItemType, SparepartID: sparepartID,
			Description: r.Description, Qty: r.Qty, UnitPrice: r.UnitPrice,
			CostPrice: cost, DiscountAmount: r.DiscountAmount,
			Subtotal: r.Subtotal, WarrantyDays: r.WarrantyDays,
		})
	}
	return items, nil
}

func (s *Service) invoicePayments(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]Payment, error) {
	type row struct {
		ID           uuid.UUID  `gorm:"column:id"`
		PaymentNo    string     `gorm:"column:payment_no"`
		InvoiceID    uuid.UUID  `gorm:"column:invoice_id"`
		InvoiceNo    string     `gorm:"column:invoice_no"`
		TicketNo     string     `gorm:"column:ticket_no"`
		CustomerName string     `gorm:"column:customer_name"`
		Method       string     `gorm:"column:method"`
		Amount       int64      `gorm:"column:amount"`
		PaidAt       time.Time  `gorm:"column:paid_at"`
		ReferenceNo  *string    `gorm:"column:reference_no"`
		Notes        *string    `gorm:"column:notes"`
		Status       string     `gorm:"column:status"`
		ReceivedBy   *string    `gorm:"column:received_by"`
		VoidedBy     *string    `gorm:"column:voided_by"`
		VoidedAt     *time.Time `gorm:"column:voided_at"`
		VoidReason   *string    `gorm:"column:void_reason"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT p.id, p.payment_no, p.invoice_id, i.invoice_no, so.ticket_no,
		       c.name AS customer_name, p.method, p.amount, p.paid_at,
		       p.reference_no, p.notes, p.status,
		       ru.name AS received_by, vu.name AS voided_by,
		       p.voided_at, p.void_reason, p.created_at
		FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		JOIN service_orders so ON so.id = i.service_order_id
		JOIN customers c ON c.id = i.customer_id
		LEFT JOIN users ru ON ru.id = p.received_by
		LEFT JOIN users vu ON vu.id = p.voided_by
		WHERE p.invoice_id = ? AND p.tenant_id = ?
		ORDER BY p.paid_at, p.created_at`, invoiceID, tenantID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	payments := make([]Payment, 0, len(rows))
	for _, r := range rows {
		payments = append(payments, Payment{
			ID: r.ID.String(), PaymentNo: r.PaymentNo, InvoiceID: r.InvoiceID.String(),
			InvoiceNo: r.InvoiceNo, TicketNo: r.TicketNo, CustomerName: r.CustomerName,
			Method: r.Method, Amount: r.Amount, PaidAt: r.PaidAt,
			ReferenceNo: r.ReferenceNo, Notes: r.Notes, Status: r.Status,
			ReceivedBy: r.ReceivedBy, VoidedBy: r.VoidedBy,
			VoidedAt: r.VoidedAt, VoidReason: r.VoidReason, CreatedAt: r.CreatedAt,
		})
	}
	return payments, nil
}

func (s *Service) storeFor(ctx context.Context, tenantID, outletID uuid.UUID) (StoreRef, error) {
	var store StoreRef
	if err := s.db.WithContext(ctx).Raw(`
		SELECT store_name AS name, legal_name, logo_path, address, phone, email, tax_id,
		       invoice_terms, invoice_footer, timezone, receipt_paper_width, currency
		FROM business_settings
		WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
		ORDER BY (outlet_id IS NULL)
		LIMIT 1`, tenantID, outletID).Scan(&store).Error; err != nil {
		return store, err
	}
	if strings.TrimSpace(store.Name) == "" {
		store.Name = "FixPhone"
	}
	if strings.TrimSpace(store.Timezone) == "" {
		store.Timezone = "Asia/Jakarta"
	}
	if store.ReceiptPaperWidth == 0 {
		store.ReceiptPaperWidth = 58
	}
	if strings.TrimSpace(store.Currency) == "" {
		store.Currency = "IDR"
	}
	return store, nil
}

type lineTotals struct {
	Service int64 `gorm:"column:service_total"`
	Part    int64 `gorm:"column:part_total"`
}

func lineTotalsFor(tx *gorm.DB, tenantID, orderID uuid.UUID) (lineTotals, error) {
	var totals lineTotals
	err := tx.Raw(`
		SELECT
			COALESCE(SUM(subtotal) FILTER (WHERE item_type IN ('service', 'custom')), 0) AS service_total,
			COALESCE(SUM(subtotal) FILTER (WHERE item_type = 'sparepart'), 0) AS part_total
		FROM service_order_items
		WHERE service_order_id = ? AND tenant_id = ?`, orderID, tenantID).Scan(&totals).Error
	return totals, err
}

// UpdateInvoice edits the draft discount and notes. Totals are always
// recalculated from the service-order lines on the server.
func (s *Service) UpdateInvoice(ctx context.Context, actor Actor, id uuid.UUID, input UpdateInvoiceRequest) (*InvoiceDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}

	var current invoiceRow
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`
			SELECT i.id, i.tenant_id, i.outlet_id, i.service_order_id, i.invoice_no,
			       i.status, i.subtotal_service, i.subtotal_part, i.discount_type,
			       i.discount_value, i.discount_amount, i.tax_amount, i.total_amount,
			       i.paid_amount, i.issued_at, i.due_at, i.notes
			FROM invoices i
			WHERE i.id = ? AND i.tenant_id = ?
			FOR UPDATE`, id, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.ID == uuid.Nil {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if !actor.HasOutlet(current.OutletID) {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if current.Status != StatusDraft || current.PaidAmount > 0 || current.IssuedAt != nil {
			return apperror.Conflict("INVOICE_LOCKED", "Invoice sudah difinalisasi atau memiliki pembayaran dan tidak dapat diubah.")
		}

		totals, err := lineTotalsFor(tx, tenantID, current.ServiceOrderID)
		if err != nil {
			return apperror.Internal(err)
		}
		discountType := current.DiscountType
		if input.DiscountType != nil {
			discountType = strings.ToLower(strings.TrimSpace(*input.DiscountType))
		}
		discountValue := current.DiscountValue
		if input.DiscountValue != nil {
			discountValue = *input.DiscountValue
		}
		discountAmount, err := calculateDiscount(totals.Service+totals.Part, discountType, discountValue)
		if err != nil {
			return err
		}
		total := totals.Service + totals.Part - discountAmount

		notes := current.Notes
		if input.Notes != nil {
			notes = trimOptional(input.Notes)
		}
		dueAt := current.DueAt
		if input.DueAt != nil {
			dueAt = nil
			if strings.TrimSpace(*input.DueAt) != "" {
				parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(*input.DueAt))
				if parseErr != nil {
					return apperror.Validation(map[string]string{"due_at": "Format jatuh tempo tidak valid."})
				}
				utc := parsed.UTC()
				dueAt = &utc
			}
		}

		now := time.Now().UTC()
		if err := tx.Exec(`
			UPDATE invoices SET subtotal_service = ?, subtotal_part = ?,
				discount_type = ?, discount_value = ?, discount_amount = ?,
				tax_amount = 0, total_amount = ?, notes = ?, due_at = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`,
			totals.Service, totals.Part, discountType, discountValue, discountAmount,
			total, notes, dueAt, now, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "invoice_update", Module: "invoice", EntityType: "invoice", EntityID: &id,
			Before: map[string]any{
				"discount_type": current.DiscountType, "discount_value": current.DiscountValue,
				"discount_amount": current.DiscountAmount, "total_amount": current.TotalAmount,
				"notes": current.Notes,
			},
			After: map[string]any{
				"discount_type": discountType, "discount_value": discountValue,
				"discount_amount": discountAmount, "total_amount": total,
				"notes": notes,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, actor, id)
}

// IssueInvoice finalises the draft: it snapshots the service items, applies the
// discount, and makes the invoice payable.
func (s *Service) IssueInvoice(ctx context.Context, actor Actor, id uuid.UUID) (*InvoiceDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}

	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		var current invoiceRow
		if err := tx.Raw(`
			SELECT i.id, i.tenant_id, i.outlet_id, i.service_order_id, i.invoice_no,
			       i.status, i.discount_type, i.discount_value, i.paid_amount,
			       i.total_amount, i.issued_at
			FROM invoices i
			WHERE i.id = ? AND i.tenant_id = ?
			FOR UPDATE`, id, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.ID == uuid.Nil {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if !actor.HasOutlet(current.OutletID) {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if current.Status != StatusDraft || current.PaidAmount > 0 || current.IssuedAt != nil {
			return apperror.Conflict("INVOICE_NOT_DRAFT", "Invoice hanya dapat difinalisasi sekali dari status draft.")
		}

		type lineRow struct {
			ID             uuid.UUID  `gorm:"column:id"`
			ItemType       string     `gorm:"column:item_type"`
			SparepartID    *uuid.UUID `gorm:"column:sparepart_id"`
			Description    string     `gorm:"column:description"`
			Qty            float64    `gorm:"column:qty"`
			UnitPrice      int64      `gorm:"column:unit_price"`
			CostPrice      int64      `gorm:"column:cost_price"`
			DiscountAmount int64      `gorm:"column:discount_amount"`
			Subtotal       int64      `gorm:"column:subtotal"`
			WarrantyDays   int        `gorm:"column:warranty_days"`
		}
		var lines []lineRow
		if err := tx.Raw(`
			SELECT id, item_type, sparepart_id, description, qty, unit_price,
			       cost_price, discount_amount, subtotal, warranty_days
			FROM service_order_items
			WHERE service_order_id = ? AND tenant_id = ?
			ORDER BY created_at, id`, current.ServiceOrderID, tenantID).Scan(&lines).Error; err != nil {
			return apperror.Internal(err)
		}

		var serviceTotal, partTotal int64
		for _, line := range lines {
			if line.ItemType == "sparepart" {
				partTotal += line.Subtotal
			} else {
				serviceTotal += line.Subtotal
			}
		}
		discountAmount, err := calculateDiscount(serviceTotal+partTotal, current.DiscountType, current.DiscountValue)
		if err != nil {
			return err
		}
		total := serviceTotal + partTotal - discountAmount
		status := StatusUnpaid
		if total == 0 {
			status = StatusPaid
		}

		if err := tx.Exec(`DELETE FROM invoice_items WHERE invoice_id = ? AND tenant_id = ?`, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		now := time.Now().UTC()
		for _, line := range lines {
			itemID, idErr := uuid.NewV7()
			if idErr != nil {
				return apperror.Internal(idErr)
			}
			if err := tx.Exec(`
				INSERT INTO invoice_items (
					id, tenant_id, invoice_id, service_order_item_id, item_type,
					sparepart_id, description, qty, unit_price, cost_price,
					discount_amount, subtotal, warranty_days, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				itemID, tenantID, id, line.ID, line.ItemType, line.SparepartID,
				line.Description, line.Qty, line.UnitPrice, line.CostPrice,
				line.DiscountAmount, line.Subtotal, line.WarrantyDays, now).Error; err != nil {
				return apperror.Internal(err)
			}
		}

		if err := tx.Exec(`
			UPDATE invoices SET subtotal_service = ?, subtotal_part = ?, discount_amount = ?,
				tax_amount = 0, total_amount = ?, paid_amount = 0, status = ?,
				issued_at = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`,
			serviceTotal, partTotal, discountAmount, total, status, now, now, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "invoice_issue", Module: "invoice", EntityType: "invoice", EntityID: &id,
			After: map[string]any{
				"invoice_no": current.InvoiceNo, "status": status,
				"subtotal_service": serviceTotal, "subtotal_part": partTotal,
				"discount_amount": discountAmount, "total_amount": total,
				"item_count": len(lines),
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, actor, id)
}

// VoidInvoice cancels an invoice that has no posted payments.
func (s *Service) VoidInvoice(ctx context.Context, actor Actor, id uuid.UUID, reason string) (*InvoiceDetail, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperror.Validation(map[string]string{"reason": "Alasan pembatalan invoice wajib diisi."})
	}

	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		var current invoiceRow
		if err := tx.Raw(`
			SELECT i.id, i.tenant_id, i.outlet_id, i.invoice_no, i.status,
			       i.paid_amount, i.total_amount
			FROM invoices i
			WHERE i.id = ? AND i.tenant_id = ?
			FOR UPDATE`, id, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.ID == uuid.Nil {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if !actor.HasOutlet(current.OutletID) {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if current.Status == StatusVoid {
			return apperror.Conflict("INVOICE_ALREADY_VOID", "Invoice sudah dibatalkan.")
		}
		var postedPayments int64
		if err := tx.Raw(`
			SELECT count(*) FROM payments
			WHERE invoice_id = ? AND tenant_id = ? AND status = 'posted'`,
			id, tenantID).Scan(&postedPayments).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.PaidAmount > 0 || postedPayments > 0 {
			return apperror.Conflict("INVOICE_HAS_PAYMENT",
				"Invoice yang sudah memiliki pembayaran tidak dapat dibatalkan. Void pembayaran terlebih dahulu.")
		}

		now := time.Now().UTC()
		if err := tx.Exec(`
			UPDATE invoices SET status = 'void', void_reason = ?, voided_by = ?, voided_at = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`, reason, actorID(actor), now, now, id, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}
		return audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "invoice_void", Module: "invoice", EntityType: "invoice", EntityID: &id,
			Before: map[string]any{"status": current.Status},
			After:  map[string]any{"status": StatusVoid, "reason": reason},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, actor, id)
}

// EnsureInvoiceAccess verifies that an invoice belongs to an actor outlet. It is
// used by the payment module through the HTTP layer rather than cross-package
// imports.
func (s *Service) EnsureInvoiceAccess(ctx context.Context, actor Actor, id uuid.UUID) (uuid.UUID, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return uuid.Nil, err
	}
	var row struct {
		OutletID uuid.UUID `gorm:"column:outlet_id"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT outlet_id FROM invoices WHERE id = ? AND tenant_id = ?`,
		id, tenantID).Scan(&row).Error; err != nil {
		return uuid.Nil, apperror.Internal(err)
	}
	if row.OutletID == uuid.Nil || !actor.HasOutlet(row.OutletID) {
		return uuid.Nil, apperror.NotFound("Invoice tidak ditemukan.")
	}
	return row.OutletID, nil
}

// Label is used in error messages where an invoice number is helpful.
func invoiceLabel(number string) string {
	if strings.TrimSpace(number) == "" {
		return "Invoice"
	}
	return fmt.Sprintf("Invoice %s", number)
}
