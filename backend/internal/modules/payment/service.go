package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/docnum"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	statusPosted = "posted"
	statusVoid   = "void"
)

var allowedPaymentMethods = map[string]struct{}{
	"cash": {}, "bank_transfer": {}, "qris": {}, "card": {}, "other": {},
}

var allowedPaymentStatuses = map[string]struct{}{
	statusPosted: {}, statusVoid: {},
}

// Service implements payment creation, listing, voiding, and summaries.
type Service struct {
	db *gorm.DB
}

// NewService builds the payment module.
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

type paymentRow struct {
	ID            uuid.UUID  `gorm:"column:id"`
	PaymentNo     string     `gorm:"column:payment_no"`
	InvoiceID     uuid.UUID  `gorm:"column:invoice_id"`
	InvoiceNo     string     `gorm:"column:invoice_no"`
	TicketNo      string     `gorm:"column:ticket_no"`
	CustomerName  string     `gorm:"column:customer_name"`
	CustomerPhone string     `gorm:"column:customer_phone"`
	OutletID      uuid.UUID  `gorm:"column:outlet_id"`
	OutletName    string     `gorm:"column:outlet_name"`
	Method        string     `gorm:"column:method"`
	Amount        int64      `gorm:"column:amount"`
	PaidAt        time.Time  `gorm:"column:paid_at"`
	ReferenceNo   *string    `gorm:"column:reference_no"`
	Notes         *string    `gorm:"column:notes"`
	Status        string     `gorm:"column:status"`
	ReceivedBy    *string    `gorm:"column:received_by"`
	VoidedBy      *string    `gorm:"column:voided_by"`
	VoidedAt      *time.Time `gorm:"column:voided_at"`
	VoidReason    *string    `gorm:"column:void_reason"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (row paymentRow) toItem() PaymentListItem {
	return PaymentListItem{
		ID: row.ID.String(), PaymentNo: row.PaymentNo,
		InvoiceID: row.InvoiceID.String(), InvoiceNo: row.InvoiceNo,
		TicketNo: row.TicketNo, CustomerName: row.CustomerName,
		CustomerPhone: row.CustomerPhone,
		OutletID:      row.OutletID.String(), OutletName: row.OutletName,
		Method: row.Method, Amount: row.Amount, PaidAt: row.PaidAt,
		ReferenceNo: row.ReferenceNo, Notes: row.Notes, Status: row.Status,
		ReceivedBy: row.ReceivedBy, VoidedBy: row.VoidedBy,
		VoidedAt: row.VoidedAt, VoidReason: row.VoidReason,
		CreatedAt: row.CreatedAt,
	}
}

const paymentBaseQuery = `
	FROM payments p
	JOIN invoices i ON i.id = p.invoice_id
	JOIN service_orders so ON so.id = i.service_order_id
	JOIN customers c ON c.id = i.customer_id
	JOIN outlets o ON o.id = p.outlet_id
	LEFT JOIN users ru ON ru.id = p.received_by
	LEFT JOIN users vu ON vu.id = p.voided_by`

const paymentSelectColumns = `
	p.id, p.payment_no, p.invoice_id, i.invoice_no, so.ticket_no,
	c.name AS customer_name, c.phone AS customer_phone,
	p.outlet_id, o.name AS outlet_name, p.method, p.amount, p.paid_at,
	p.reference_no, p.notes, p.status,
	ru.name AS received_by, vu.name AS voided_by,
	p.voided_at, p.void_reason, p.created_at`

func (s *Service) paymentRows(ctx context.Context, db *gorm.DB, where string, args []any,
	page, perPage int) ([]paymentRow, error) {
	queryArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	var rows []paymentRow
	err := db.WithContext(ctx).Raw(`
		SELECT `+paymentSelectColumns+paymentBaseQuery+where+`
		ORDER BY p.paid_at DESC, p.created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...).Scan(&rows).Error
	return rows, err
}

// ListPayments returns payments scoped to an outlet.
func (s *Service) ListPayments(ctx context.Context, actor Actor, filter PaymentFilter) (*PaymentList, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	var requested *uuid.UUID
	if filter.InvoiceID != uuid.Nil {
		var row struct {
			OutletID uuid.UUID `gorm:"column:outlet_id"`
		}
		if err := s.db.WithContext(ctx).Raw(`
			SELECT outlet_id FROM invoices WHERE id = ? AND tenant_id = ?`,
			filter.InvoiceID, tenantID).Scan(&row).Error; err != nil {
			return nil, apperror.Internal(err)
		}
		if row.OutletID == uuid.Nil || !actor.HasOutlet(row.OutletID) {
			return nil, apperror.NotFound("Invoice tidak ditemukan.")
		}
		requested = &row.OutletID
	} else if filter.OutletID != uuid.Nil {
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
	if filter.Method != "" {
		filter.Method = strings.ToLower(strings.TrimSpace(filter.Method))
		if _, ok := allowedPaymentMethods[filter.Method]; !ok {
			return nil, apperror.Validation(map[string]string{"method": "Metode pembayaran tidak dikenal."})
		}
	}
	for _, status := range filter.Statuses {
		if _, ok := allowedPaymentStatuses[status]; !ok {
			return nil, apperror.Validation(map[string]string{"status": "Status pembayaran tidak dikenal."})
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

	where := ` WHERE p.tenant_id = ? AND p.outlet_id = ?`
	args := []any{tenantID, outletID}
	if filter.InvoiceID != uuid.Nil {
		where += ` AND p.invoice_id = ?`
		args = append(args, filter.InvoiceID)
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + query + "%"
		where += ` AND (p.payment_no ILIKE ? OR i.invoice_no ILIKE ? OR so.ticket_no ILIKE ? OR c.name ILIKE ? OR c.phone ILIKE ? OR p.reference_no ILIKE ?)`
		args = append(args, like, like, like, like, like, like)
	}
	if filter.Method != "" {
		where += ` AND p.method = ?`
		args = append(args, filter.Method)
	}
	if len(filter.Statuses) > 0 {
		where += ` AND p.status IN ?`
		args = append(args, filter.Statuses)
	}
	if from != nil {
		where += ` AND p.paid_at >= ?`
		args = append(args, *from)
	}
	if to != nil {
		where += ` AND p.paid_at < ?`
		args = append(args, *to)
	}

	var total int64
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*)`+paymentBaseQuery+where, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	rows, err := s.paymentRows(ctx, s.db, where, args, filter.Page, filter.PerPage)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]PaymentListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toItem())
	}
	return &PaymentList{Items: items, Total: total}, nil
}

// ListInvoicePayments returns all payments for one invoice.
func (s *Service) ListInvoicePayments(ctx context.Context, actor Actor, invoiceID uuid.UUID) ([]PaymentListItem, error) {
	// Verify the invoice belongs to the caller before filtering payments so a
	// foreign invoice id looks exactly like a missing one (404, not an empty
	// list that would confirm its existence).
	if err := s.ensureInvoiceAccess(ctx, actor, invoiceID); err != nil {
		return nil, err
	}
	result, err := s.ListPayments(ctx, actor, PaymentFilter{
		InvoiceID: invoiceID, Page: 1, PerPage: 100,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// ensureInvoiceAccess rejects invoice ids outside the actor's tenant or outlets.
func (s *Service) ensureInvoiceAccess(ctx context.Context, actor Actor, invoiceID uuid.UUID) error {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return err
	}
	var row struct {
		OutletID uuid.UUID `gorm:"column:outlet_id"`
	}
	result := s.db.WithContext(ctx).Raw(`
		SELECT outlet_id FROM invoices WHERE id = ? AND tenant_id = ?`,
		invoiceID, tenantID).Scan(&row)
	if result.Error != nil {
		return apperror.Internal(result.Error)
	}
	if result.RowsAffected == 0 || row.OutletID == uuid.Nil || !actor.HasOutlet(row.OutletID) {
		return apperror.NotFound("Invoice tidak ditemukan.")
	}
	return nil
}

type invoiceRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	InvoiceNo      string     `gorm:"column:invoice_no"`
	OutletID       uuid.UUID  `gorm:"column:outlet_id"`
	ServiceOrderID uuid.UUID  `gorm:"column:service_order_id"`
	Status         string     `gorm:"column:status"`
	TotalAmount    int64      `gorm:"column:total_amount"`
	PaidAmount     int64      `gorm:"column:paid_amount"`
	IssuedAt       *time.Time `gorm:"column:issued_at"`
}

func (row invoiceRow) summary() InvoiceSummary {
	remaining := row.TotalAmount - row.PaidAmount
	if remaining < 0 {
		remaining = 0
	}
	return InvoiceSummary{
		ID: row.ID.String(), InvoiceNo: row.InvoiceNo, Status: row.Status,
		TotalAmount: row.TotalAmount, PaidAmount: row.PaidAmount, RemainingAmount: remaining,
	}
}

func (s *Service) paymentByKey(ctx context.Context, tenantID uuid.UUID, key string) (*paymentRow, error) {
	var row paymentRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT `+paymentSelectColumns+paymentBaseQuery+`
		WHERE p.tenant_id = ? AND p.idempotency_key = ?`, tenantID, key).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) invoiceSummary(ctx context.Context, db *gorm.DB, tenantID, invoiceID uuid.UUID) (InvoiceSummary, error) {
	var row invoiceRow
	if err := db.WithContext(ctx).Raw(`
		SELECT id, invoice_no, status, total_amount, paid_amount
		FROM invoices WHERE id = ? AND tenant_id = ?`, invoiceID, tenantID).Scan(&row).Error; err != nil {
		return InvoiceSummary{}, err
	}
	if row.ID == uuid.Nil {
		return InvoiceSummary{}, apperror.NotFound("Invoice tidak ditemukan.")
	}
	return row.summary(), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreatePayment posts a full or partial payment and recalculates the invoice.
func (s *Service) CreatePayment(ctx context.Context, actor Actor, invoiceID uuid.UUID,
	idempotencyKey string, input CreatePaymentRequest) (*CreatePaymentResult, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	method := strings.ToLower(strings.TrimSpace(input.Method))
	if _, ok := allowedPaymentMethods[method]; !ok {
		return nil, apperror.Validation(map[string]string{"method": "Metode pembayaran tidak dikenal."})
	}
	if input.Amount <= 0 {
		return nil, apperror.Validation(map[string]string{"amount": "Nominal pembayaran harus lebih dari 0."})
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if len(idempotencyKey) > 100 {
		return nil, apperror.Validation(map[string]string{"Idempotency-Key": "Idempotency-Key maksimal 100 karakter."})
	}
	paidAt := time.Now().UTC()
	if input.PaidAt != nil && strings.TrimSpace(*input.PaidAt) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(*input.PaidAt))
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"paid_at": "Format tanggal pembayaran tidak valid."})
		}
		paidAt = parsed.UTC()
	}
	referenceNo := trimOptional(input.ReferenceNo)
	notes := trimOptional(input.Notes)

	// A retried request returns the original payment without creating a second row.
	if idempotencyKey != "" {
		existing, findErr := s.paymentByKey(ctx, tenantID, idempotencyKey)
		if findErr != nil {
			return nil, apperror.Internal(findErr)
		}
		if existing != nil {
			if !actor.HasOutlet(existing.OutletID) {
				return nil, apperror.Conflict("IDEMPOTENCY_KEY_CONFLICT", "Idempotency-Key sudah digunakan untuk transaksi lain.")
			}
			summary, sumErr := s.invoiceSummary(ctx, s.db, tenantID, existing.InvoiceID)
			if sumErr != nil {
				return nil, sumErr
			}
			return &CreatePaymentResult{Payment: existing.toItem(), Invoice: summary}, nil
		}
	}

	var result *CreatePaymentResult
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if idempotencyKey != "" {
			var existing paymentRow
			if err := tx.Raw(`
				SELECT `+paymentSelectColumns+paymentBaseQuery+`
				WHERE p.tenant_id = ? AND p.idempotency_key = ?`, tenantID, idempotencyKey).
				Scan(&existing).Error; err != nil {
				return apperror.Internal(err)
			}
			if existing.ID != uuid.Nil {
				if !actor.HasOutlet(existing.OutletID) {
					return apperror.Conflict("IDEMPOTENCY_KEY_CONFLICT", "Idempotency-Key sudah digunakan untuk transaksi lain.")
				}
				summary, sumErr := s.invoiceSummary(ctx, tx, tenantID, existing.InvoiceID)
				if sumErr != nil {
					return sumErr
				}
				result = &CreatePaymentResult{Payment: existing.toItem(), Invoice: summary}
				return nil
			}
		}

		var invoice invoiceRow
		if err := tx.Raw(`
			SELECT id, invoice_no, outlet_id, service_order_id, status,
			       total_amount, paid_amount, issued_at
			FROM invoices
			WHERE id = ? AND tenant_id = ?
			FOR UPDATE`, invoiceID, tenantID).Scan(&invoice).Error; err != nil {
			return apperror.Internal(err)
		}
		if invoice.ID == uuid.Nil || !actor.HasOutlet(invoice.OutletID) {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}
		if invoice.Status == "void" {
			return apperror.Conflict("INVOICE_VOID", "Invoice sudah dibatalkan dan tidak dapat menerima pembayaran.")
		}
		if invoice.IssuedAt == nil || invoice.Status == "draft" {
			return apperror.Conflict("INVOICE_NOT_ISSUED", "Invoice belum difinalisasi sehingga belum dapat dibayar.")
		}
		remaining := invoice.TotalAmount - invoice.PaidAmount
		if remaining <= 0 {
			return apperror.Conflict("OVERPAYMENT", "Invoice sudah lunas.").WithDetails(map[string]any{"remaining_amount": int64(0)})
		}
		if input.Amount > remaining {
			return apperror.Conflict("OVERPAYMENT", "Nominal pembayaran melebihi sisa tagihan.").
				WithDetails(map[string]any{"remaining_amount": remaining})
		}

		number, numberErr := docnum.Next(ctx, tx, tenantID, invoice.OutletID, "payment",
			"PAY", "{PREFIX}-{YYMMDD}-{SEQ:4}", time.Now().UTC())
		if numberErr != nil {
			return apperror.Internal(numberErr)
		}
		paymentID, idErr := uuid.NewV7()
		if idErr != nil {
			return apperror.Internal(idErr)
		}
		var key any
		if idempotencyKey != "" {
			key = idempotencyKey
		}
		now := time.Now().UTC()
		if err := tx.Exec(`
			INSERT INTO payments (
				id, tenant_id, outlet_id, invoice_id, payment_no, method, amount,
				paid_at, reference_no, notes, status, received_by, idempotency_key, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'posted', ?, ?, ?)`,
			paymentID, tenantID, invoice.OutletID, invoice.ID, number, method,
			input.Amount, paidAt, referenceNo, notes, actorID(actor), key, now).Error; err != nil {
			return err
		}

		newPaid := invoice.PaidAmount + input.Amount
		status := "partial"
		if newPaid >= invoice.TotalAmount {
			status = "paid"
		}
		if err := tx.Exec(`
			UPDATE invoices SET paid_amount = ?, status = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`, newPaid, status, now, invoice.ID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		if err := audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "payment_create", Module: "payment",
			EntityType: "payment", EntityID: &paymentID,
			After: map[string]any{
				"payment_no": number, "invoice_no": invoice.InvoiceNo,
				"invoice_id": invoice.ID, "method": method, "amount": input.Amount,
				"invoice_status": status, "paid_amount": newPaid,
				"remaining_amount": invoice.TotalAmount - newPaid,
			},
		}); err != nil {
			return apperror.Internal(err)
		}

		created, err := s.paymentRows(ctx, tx,
			` WHERE p.tenant_id = ? AND p.id = ?`, []any{tenantID, paymentID}, 1, 1)
		if err != nil {
			return apperror.Internal(err)
		}
		if len(created) != 1 {
			return apperror.Internal(fmt.Errorf("payment %s not readable after insert", paymentID))
		}
		result = &CreatePaymentResult{
			Payment: created[0].toItem(),
			Invoice: invoiceRow{
				ID: invoice.ID, InvoiceNo: invoice.InvoiceNo, Status: status,
				TotalAmount: invoice.TotalAmount, PaidAmount: newPaid,
			}.summary(),
		}
		return nil
	})
	if err != nil {
		if idempotencyKey != "" && isUniqueViolation(err) {
			existing, findErr := s.paymentByKey(ctx, tenantID, idempotencyKey)
			if findErr != nil {
				return nil, apperror.Internal(findErr)
			}
			if existing != nil && actor.HasOutlet(existing.OutletID) {
				summary, sumErr := s.invoiceSummary(ctx, s.db, tenantID, existing.InvoiceID)
				if sumErr != nil {
					return nil, sumErr
				}
				return &CreatePaymentResult{Payment: existing.toItem(), Invoice: summary}, nil
			}
		}
		if _, ok := apperror.As(err); ok {
			return nil, err
		}
		return nil, apperror.Internal(err)
	}
	return result, nil
}

// VoidPayment cancels a posted payment and recalculates the invoice status.
func (s *Service) VoidPayment(ctx context.Context, actor Actor, paymentID uuid.UUID, reason string) (*VoidPaymentResult, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperror.Validation(map[string]string{"reason": "Alasan pembatalan pembayaran wajib diisi."})
	}

	var result *VoidPaymentResult
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		var current paymentRow
		if err := tx.Raw(`
			SELECT p.id, p.tenant_id, p.outlet_id, p.invoice_id, p.payment_no,
			       p.method, p.amount, p.paid_at, p.status, p.reference_no,
			       p.notes, p.created_at
			FROM payments p
			WHERE p.id = ? AND p.tenant_id = ?
			FOR UPDATE`, paymentID, tenantID).Scan(&current).Error; err != nil {
			return apperror.Internal(err)
		}
		if current.ID == uuid.Nil || !actor.HasOutlet(current.OutletID) {
			return apperror.NotFound("Pembayaran tidak ditemukan.")
		}
		if current.Status == statusVoid {
			return apperror.Conflict("PAYMENT_ALREADY_VOID", "Pembayaran sudah dibatalkan.")
		}

		var invoice invoiceRow
		if err := tx.Raw(`
			SELECT id, invoice_no, outlet_id, service_order_id, status,
			       total_amount, paid_amount, issued_at
			FROM invoices
			WHERE id = ? AND tenant_id = ?
			FOR UPDATE`, current.InvoiceID, tenantID).Scan(&invoice).Error; err != nil {
			return apperror.Internal(err)
		}
		if invoice.ID == uuid.Nil {
			return apperror.NotFound("Invoice tidak ditemukan.")
		}

		now := time.Now().UTC()
		if err := tx.Exec(`
			UPDATE payments SET status = 'void', voided_by = ?, voided_at = ?, void_reason = ?
			WHERE id = ? AND tenant_id = ?`,
			actorID(actor), now, reason, current.ID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		newPaid := invoice.PaidAmount - current.Amount
		if newPaid < 0 {
			newPaid = 0
		}
		status := invoice.Status
		if status != "void" {
			switch {
			case newPaid <= 0:
				status = "unpaid"
			case newPaid >= invoice.TotalAmount:
				status = "paid"
			default:
				status = "partial"
			}
		}
		if err := tx.Exec(`
			UPDATE invoices SET paid_amount = ?, status = ?, updated_at = ?
			WHERE id = ? AND tenant_id = ?`, newPaid, status, now, invoice.ID, tenantID).Error; err != nil {
			return apperror.Internal(err)
		}

		if err := audit.Write(ctx, tx, audit.Entry{
			TenantID: &tenantID, UserID: actorID(actor), ActorName: actor.Name,
			Action: "payment_void", Module: "payment",
			EntityType: "payment", EntityID: &current.ID,
			Before: map[string]any{
				"status": statusPosted, "amount": current.Amount,
				"invoice_status": invoice.Status, "paid_amount": invoice.PaidAmount,
			},
			After: map[string]any{
				"status": statusVoid, "reason": reason,
				"invoice_status": status, "paid_amount": newPaid,
			},
		}); err != nil {
			return apperror.Internal(err)
		}

		updated, err := s.paymentRows(ctx, tx,
			` WHERE p.tenant_id = ? AND p.id = ?`, []any{tenantID, current.ID}, 1, 1)
		if err != nil {
			return apperror.Internal(err)
		}
		if len(updated) != 1 {
			return apperror.Internal(fmt.Errorf("payment %s not readable after void", current.ID))
		}
		result = &VoidPaymentResult{
			Payment: updated[0].toItem(),
			Invoice: invoiceRow{
				ID: invoice.ID, InvoiceNo: invoice.InvoiceNo, Status: status,
				TotalAmount: invoice.TotalAmount, PaidAmount: newPaid,
			}.summary(),
		}
		return nil
	})
	if err != nil {
		if _, ok := apperror.As(err); ok {
			return nil, err
		}
		return nil, apperror.Internal(err)
	}
	return result, nil
}

// DailySummary aggregates posted and voided payments for one business day.
func (s *Service) DailySummary(ctx context.Context, actor Actor, date string, requestedOutlet *uuid.UUID) (*DailySummary, error) {
	tenantID, err := actorTenant(actor)
	if err != nil {
		return nil, err
	}
	outletID, err := actor.outletFor(requestedOutlet)
	if err != nil {
		return nil, err
	}
	loc, err := s.outletTimezone(ctx, tenantID, outletID)
	if err != nil {
		return nil, err
	}
	date = strings.TrimSpace(date)
	base := time.Now().In(loc)
	if date != "" {
		parsed, parseErr := time.ParseInLocation("2006-01-02", date, loc)
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"date": "Format tanggal harus YYYY-MM-DD."})
		}
		base = parsed
	}
	start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, loc).UTC()
	end := start.AddDate(0, 0, 1)

	var outletName string
	if err := s.db.WithContext(ctx).Raw(`
		SELECT name FROM outlets WHERE id = ? AND tenant_id = ?`,
		outletID, tenantID).Scan(&outletName).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	var totals struct {
		TotalAmount  int64 `gorm:"column:total_amount"`
		PostedCount  int64 `gorm:"column:posted_count"`
		InvoiceCount int64 `gorm:"column:invoice_count"`
		VoidedAmount int64 `gorm:"column:voided_amount"`
		VoidedCount  int64 `gorm:"column:voided_count"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE status = 'posted'), 0) AS total_amount,
			count(*) FILTER (WHERE status = 'posted') AS posted_count,
			count(DISTINCT invoice_id) FILTER (WHERE status = 'posted') AS invoice_count,
			COALESCE(SUM(amount) FILTER (WHERE status = 'void'), 0) AS voided_amount,
			count(*) FILTER (WHERE status = 'void') AS voided_count
		FROM payments
		WHERE tenant_id = ? AND outlet_id = ? AND paid_at >= ? AND paid_at < ?`,
		tenantID, outletID, start, end).Scan(&totals).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	var methods []MethodSummary
	if err := s.db.WithContext(ctx).Raw(`
		SELECT method, COALESCE(SUM(amount), 0) AS amount, count(*) AS count
		FROM payments
		WHERE tenant_id = ? AND outlet_id = ? AND paid_at >= ? AND paid_at < ? AND status = 'posted'
		GROUP BY method
		ORDER BY method`, tenantID, outletID, start, end).Scan(&methods).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if methods == nil {
		methods = []MethodSummary{}
	}

	return &DailySummary{
		Date: base.Format("2006-01-02"), OutletID: outletID.String(),
		OutletName: outletName, TotalAmount: totals.TotalAmount,
		PostedCount: totals.PostedCount, InvoiceCount: totals.InvoiceCount,
		VoidedAmount: totals.VoidedAmount, VoidedCount: totals.VoidedCount,
		Methods: methods,
	}, nil
}
