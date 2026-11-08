// Package payment implements cashier payments, voiding, and daily summaries.
package payment

import (
	"time"

	"github.com/google/uuid"
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

// PaymentListItem is one payment row.
type PaymentListItem struct {
	ID            string     `json:"id"`
	PaymentNo     string     `json:"payment_no"`
	InvoiceID     string     `json:"invoice_id"`
	InvoiceNo     string     `json:"invoice_no"`
	TicketNo      string     `json:"ticket_no"`
	CustomerName  string     `json:"customer_name"`
	CustomerPhone string     `json:"customer_phone"`
	OutletID      string     `json:"outlet_id"`
	OutletName    string     `json:"outlet_name"`
	Method        string     `json:"method"`
	Amount        int64      `json:"amount"`
	PaidAt        time.Time  `json:"paid_at"`
	ReferenceNo   *string    `json:"reference_no"`
	Notes         *string    `json:"notes"`
	Status        string     `json:"status"`
	ReceivedBy    *string    `json:"received_by"`
	VoidedBy      *string    `json:"voided_by"`
	VoidedAt      *time.Time `json:"voided_at"`
	VoidReason    *string    `json:"void_reason"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PaymentList is a paginated payment result.
type PaymentList struct {
	Items []PaymentListItem
	Total int64
}

// PaymentFilter is the payment list query.
type PaymentFilter struct {
	Query     string
	Statuses  []string
	Method    string
	InvoiceID uuid.UUID
	OutletID  uuid.UUID
	FromDate  string
	ToDate    string
	Page      int
	PerPage   int
}

// InvoiceSummary is the invoice state returned after a payment mutation.
type InvoiceSummary struct {
	ID              string `json:"id"`
	InvoiceNo       string `json:"invoice_no"`
	Status          string `json:"status"`
	TotalAmount     int64  `json:"total_amount"`
	PaidAmount      int64  `json:"paid_amount"`
	RemainingAmount int64  `json:"remaining_amount"`
}

// CreatePaymentResult is the response of a successful payment.
type CreatePaymentResult struct {
	Payment PaymentListItem `json:"payment"`
	Invoice InvoiceSummary  `json:"invoice"`
}

// VoidPaymentResult is the response of a successful payment void.
type VoidPaymentResult struct {
	Payment PaymentListItem `json:"payment"`
	Invoice InvoiceSummary  `json:"invoice"`
}

// CreatePaymentRequest accepts a full or partial payment.
type CreatePaymentRequest struct {
	Method      string  `json:"method" validate:"required,oneof=cash bank_transfer qris card other"`
	Amount      int64   `json:"amount"`
	PaidAt      *string `json:"paid_at"`
	ReferenceNo *string `json:"reference_no" validate:"omitempty,max=80"`
	Notes       *string `json:"notes" validate:"omitempty,max=1000"`
}

// VoidPaymentRequest cancels a posted payment.
type VoidPaymentRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

// MethodSummary aggregates posted payments per method.
type MethodSummary struct {
	Method string `json:"method"`
	Amount int64  `json:"amount"`
	Count  int64  `json:"count"`
}

// DailySummary is the cashier daily report.
type DailySummary struct {
	Date         string          `json:"date"`
	OutletID     string          `json:"outlet_id"`
	OutletName   string          `json:"outlet_name"`
	TotalAmount  int64           `json:"total_amount"`
	PostedCount  int64           `json:"posted_count"`
	InvoiceCount int64           `json:"invoice_count"`
	VoidedAmount int64           `json:"voided_amount"`
	VoidedCount  int64           `json:"voided_count"`
	Methods      []MethodSummary `json:"methods"`
}
