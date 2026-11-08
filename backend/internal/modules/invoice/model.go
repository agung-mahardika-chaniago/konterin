// Package invoice implements invoice finalisation, discounts, and printing data.
package invoice

import (
	"time"

	"github.com/google/uuid"
)

// Invoice statuses accepted by the database constraint.
const (
	StatusDraft   = "draft"
	StatusUnpaid  = "unpaid"
	StatusPartial = "partial"
	StatusPaid    = "paid"
	StatusVoid    = "void"
)

// Actor is the authenticated staff member plus their tenant/outlet boundary.
type Actor struct {
	UserID          uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Role            string
	Permissions     map[string]struct{}
	AllOutletAccess bool
	OutletIDs       []uuid.UUID
	ActiveOutletID  *uuid.UUID
}

// HasPermission reports whether the actor holds a permission code.
func (a Actor) HasPermission(code string) bool {
	if a.Permissions == nil {
		return false
	}
	_, ok := a.Permissions[code]
	return ok
}

// HasOutlet reports whether the actor may use the outlet.
func (a Actor) HasOutlet(outletID uuid.UUID) bool {
	for _, id := range a.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

// InvoiceListItem is one row in the invoice list.
type InvoiceListItem struct {
	ID              string     `json:"id"`
	InvoiceNo       string     `json:"invoice_no"`
	Status          string     `json:"status"`
	ServiceOrderID  string     `json:"service_order_id"`
	TicketNo        string     `json:"ticket_no"`
	ServiceStatus   string     `json:"service_status"`
	CustomerID      string     `json:"customer_id"`
	CustomerName    string     `json:"customer_name"`
	CustomerPhone   string     `json:"customer_phone"`
	OutletID        string     `json:"outlet_id"`
	OutletName      string     `json:"outlet_name"`
	SubtotalService int64      `json:"subtotal_service"`
	SubtotalPart    int64      `json:"subtotal_part"`
	DiscountType    string     `json:"discount_type"`
	DiscountValue   int64      `json:"discount_value"`
	DiscountAmount  int64      `json:"discount_amount"`
	TaxAmount       int64      `json:"tax_amount"`
	TotalAmount     int64      `json:"total_amount"`
	PaidAmount      int64      `json:"paid_amount"`
	RemainingAmount int64      `json:"remaining_amount"`
	IssuedAt        *time.Time `json:"issued_at"`
	DueAt           *time.Time `json:"due_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// InvoiceList is a paginated invoice result.
type InvoiceList struct {
	Items []InvoiceListItem
	Total int64
}

// ServiceRef describes the service order behind an invoice.
type ServiceRef struct {
	ID             string     `json:"id"`
	TicketNo       string     `json:"ticket_no"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	DeviceBrand    string     `json:"device_brand"`
	DeviceModel    string     `json:"device_model"`
	DeviceIMEI     *string    `json:"device_imei"`
	DeviceColor    *string    `json:"device_color"`
	Complaint      string     `json:"complaint"`
	ReceivedAt     time.Time  `json:"received_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	PickedUpAt     *time.Time `json:"picked_up_at"`
	TechnicianName *string    `json:"technician_name"`
	PublicTracking *string    `json:"public_tracking_url"`
}

// StoreRef is the business header used on printed invoices and receipts.
type StoreRef struct {
	Name              string  `json:"name"`
	LegalName         *string `json:"legal_name"`
	LogoPath          *string `json:"logo_path"`
	Address           *string `json:"address"`
	Phone             *string `json:"phone"`
	Email             *string `json:"email"`
	TaxID             *string `json:"tax_id"`
	InvoiceTerms      *string `json:"invoice_terms"`
	InvoiceFooter     *string `json:"invoice_footer"`
	Timezone          string  `json:"timezone"`
	ReceiptPaperWidth int     `json:"receipt_paper_width"`
	Currency          string  `json:"currency"`
}

// InvoiceItem is one immutable snapshot line.
type InvoiceItem struct {
	ID             string  `json:"id"`
	ItemType       string  `json:"item_type"`
	SparepartID    *string `json:"sparepart_id"`
	Description    string  `json:"description"`
	Qty            float64 `json:"qty"`
	UnitPrice      int64   `json:"unit_price"`
	CostPrice      int64   `json:"cost_price"`
	DiscountAmount int64   `json:"discount_amount"`
	Subtotal       int64   `json:"subtotal"`
	WarrantyDays   int     `json:"warranty_days"`
}

// Payment is a posted or voided payment attached to an invoice.
type Payment struct {
	ID           string     `json:"id"`
	PaymentNo    string     `json:"payment_no"`
	InvoiceID    string     `json:"invoice_id"`
	InvoiceNo    string     `json:"invoice_no"`
	TicketNo     string     `json:"ticket_no"`
	CustomerName string     `json:"customer_name"`
	Method       string     `json:"method"`
	Amount       int64      `json:"amount"`
	PaidAt       time.Time  `json:"paid_at"`
	ReferenceNo  *string    `json:"reference_no"`
	Notes        *string    `json:"notes"`
	Status       string     `json:"status"`
	ReceivedBy   *string    `json:"received_by"`
	VoidedBy     *string    `json:"voided_by"`
	VoidedAt     *time.Time `json:"voided_at"`
	VoidReason   *string    `json:"void_reason"`
	CreatedAt    time.Time  `json:"created_at"`
}

// InvoiceDetail is the full invoice payload used by the detail and print views.
type InvoiceDetail struct {
	InvoiceListItem
	Notes      *string       `json:"notes"`
	VoidReason *string       `json:"void_reason"`
	VoidedBy   *string       `json:"voided_by"`
	VoidedAt   *time.Time    `json:"voided_at"`
	Order      ServiceRef    `json:"service_order"`
	Items      []InvoiceItem `json:"items"`
	Payments   []Payment     `json:"payments"`
	Store      StoreRef      `json:"store"`
}

// InvoiceFilter is the list query.
type InvoiceFilter struct {
	Query    string
	Statuses []string
	OutletID uuid.UUID
	FromDate string
	ToDate   string
	Page     int
	PerPage  int
}

// UpdateInvoiceRequest edits discount and notes while the invoice is a draft.
type UpdateInvoiceRequest struct {
	DiscountType  *string `json:"discount_type"`
	DiscountValue *int64  `json:"discount_value"`
	Notes         *string `json:"notes" validate:"omitempty,max=2000"`
	DueAt         *string `json:"due_at"`
}

// VoidInvoiceRequest cancels an invoice that has not received money.
type VoidInvoiceRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}
