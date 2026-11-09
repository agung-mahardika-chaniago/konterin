// Package warranty implements after-sales warranties, claims, and rework orders.
package warranty

import (
	"time"

	"github.com/google/uuid"
)

// Claim statuses accepted by the database constraint.
const (
	ClaimSubmitted  = "submitted"
	ClaimApproved   = "approved"
	ClaimRejected   = "rejected"
	ClaimInProgress = "in_progress"
	ClaimCompleted  = "completed"
)

// Actor is the authenticated staff member plus tenant/outlet boundaries.
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

// CustomerRef is the customer summary embedded in warranty responses.
type CustomerRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// DeviceRef is the device summary.
type DeviceRef struct {
	Brand string `json:"brand"`
	Model string `json:"model"`
}

// ServiceRef points at the originating service order.
type ServiceRef struct {
	ID       string `json:"id"`
	TicketNo string `json:"ticket_no"`
	Status   string `json:"status"`
}

// OutletRef is the outlet summary.
type OutletRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InvoiceRef is the invoice summary linked to a warranty.
type InvoiceRef struct {
	ID              string `json:"id"`
	InvoiceNo       string `json:"invoice_no"`
	Status          string `json:"status"`
	TotalAmount     int64  `json:"total_amount"`
	PaidAmount      int64  `json:"paid_amount"`
	RemainingAmount int64  `json:"remaining_amount"`
}

// Warranty is the API representation of a warranty.
type Warranty struct {
	ID           string      `json:"id"`
	WarrantyNo   string      `json:"warranty_no"`
	Type         string      `json:"type"`
	Status       string      `json:"status"`
	StartDate    string      `json:"start_date"`
	EndDate      string      `json:"end_date"`
	Days         int         `json:"days"`
	Terms        *string     `json:"terms"`
	Customer     CustomerRef `json:"customer"`
	Device       DeviceRef   `json:"device"`
	ServiceOrder ServiceRef  `json:"service_order"`
	Outlet       OutletRef   `json:"outlet"`
	ClaimCount   int64       `json:"claim_count"`
	Invoice      *InvoiceRef `json:"invoice,omitempty"`
	Claims       []Claim     `json:"claims,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

// Claim is the API representation of a warranty claim.
type Claim struct {
	ID               string      `json:"id"`
	ClaimNo          string      `json:"claim_no"`
	WarrantyID       string      `json:"warranty_id"`
	WarrantyNo       string      `json:"warranty_no"`
	Status           string      `json:"status"`
	Complaint        string      `json:"complaint"`
	Resolution       *string     `json:"resolution"`
	Customer         CustomerRef `json:"customer"`
	Device           DeviceRef   `json:"device"`
	ServiceOrder     ServiceRef  `json:"service_order"`
	Outlet           OutletRef   `json:"outlet"`
	HandledBy        *string     `json:"handled_by"`
	ReworkServiceOrder *ServiceRef `json:"rework_service_order,omitempty"`
	ClaimedAt        time.Time   `json:"claimed_at"`
	ResolvedAt       *time.Time  `json:"resolved_at"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// WarrantyFilter narrows the warranty list.
type WarrantyFilter struct {
	Status        string
	Query         string
	OutletID      *uuid.UUID
	ExpiringInDays int
	Page          int
	PerPage       int
}

// ClaimFilter narrows the claim list.
type ClaimFilter struct {
	Status   string
	Query    string
	OutletID *uuid.UUID
	Page     int
	PerPage  int
}

// CreateClaimRequest is the POST /warranties/:id/claims payload.
type CreateClaimRequest struct {
	Complaint string `json:"complaint"`
}

// UpdateClaimRequest is the PATCH /claims/:id payload.
type UpdateClaimRequest struct {
	Status     *string `json:"status"`
	Resolution *string `json:"resolution"`
}

// ReworkResult is returned after a rework order is created.
type ReworkResult struct {
	ServiceOrderID string `json:"service_order_id"`
	TicketNo       string `json:"ticket_no"`
	Status         string `json:"status"`
	ClaimID        string `json:"claim_id"`
	ClaimStatus    string `json:"claim_status"`
	InvoiceNo      string `json:"invoice_no"`
	CreatedAt      time.Time `json:"created_at"`
}
