// Package service implements customer intake and service-order management.
package service

import (
	"time"

	"github.com/google/uuid"
)

// Status and priority values accepted by the database constraints.
const (
	StatusReceived = "received"

	PriorityNormal  = "normal"
	PriorityExpress = "express"

	CredentialNone     = "none"
	CredentialPIN      = "pin"
	CredentialPassword = "password"
	CredentialPattern  = "pattern"
)

var allowedStatuses = map[string]struct{}{
	"received": {}, "waiting_diagnosis": {}, "waiting_approval": {},
	"waiting_part": {}, "in_progress": {}, "testing": {},
	"completed": {}, "ready_for_pickup": {}, "picked_up": {}, "cancelled": {},
}

// Actor is the authenticated staff member plus their outlet boundary.
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
	if a.AllOutletAccess {
		for _, id := range a.OutletIDs {
			if id == outletID {
				return true
			}
		}
		return false
	}
	for _, id := range a.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

// Customer is the customer projection used by the API.
type Customer struct {
	ID            uuid.UUID  `gorm:"column:id"`
	TenantID      uuid.UUID  `gorm:"column:tenant_id"`
	Name          string     `gorm:"column:name"`
	Phone         string     `gorm:"column:phone"`
	Address       *string    `gorm:"column:address"`
	Notes         *string    `gorm:"column:notes"`
	LastServiceAt *time.Time `gorm:"column:last_service_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

// Device is a customer device.
type Device struct {
	ID         uuid.UUID `gorm:"column:id"`
	TenantID   uuid.UUID `gorm:"column:tenant_id"`
	CustomerID uuid.UUID `gorm:"column:customer_id"`
	Brand      string    `gorm:"column:brand"`
	Model      string    `gorm:"column:model"`
	IMEI       *string   `gorm:"column:imei"`
	Color      *string   `gorm:"column:color"`
	Notes      *string   `gorm:"column:notes"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

// CustomerDTO is the API representation of a customer.
type CustomerDTO struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Phone         string         `json:"phone"`
	Address       *string        `json:"address"`
	Notes         *string        `json:"notes"`
	LastServiceAt *time.Time     `json:"last_service_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Devices       []DeviceDTO    `json:"devices,omitempty"`
	Stats         *CustomerStats `json:"stats,omitempty"`
}

// CustomerStats summarises a customer's history.
type CustomerStats struct {
	TotalServices int64      `json:"total_services"`
	TotalPaid     int64      `json:"total_paid"`
	LastServiceAt *time.Time `json:"last_service_at"`
}

// DeviceDTO is the API representation of a device.
type DeviceDTO struct {
	ID        string    `json:"id"`
	Brand     string    `json:"brand"`
	Model     string    `json:"model"`
	IMEI      *string   `json:"imei"`
	Color     *string   `json:"color"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CustomerList is a paginated customer result.
type CustomerList struct {
	Items []CustomerDTO
	Total int64
}

// CreateCustomerRequest creates a customer directly.
type CreateCustomerRequest struct {
	Name    string  `json:"name" validate:"required,max=120"`
	Phone   string  `json:"phone" validate:"required,max=30"`
	Address *string `json:"address" validate:"omitempty,max=500"`
	Notes   *string `json:"notes" validate:"omitempty,max=1000"`
}

// UpdateCustomerRequest edits a customer.
type UpdateCustomerRequest struct {
	Name    *string `json:"name" validate:"omitempty,max=120"`
	Phone   *string `json:"phone" validate:"omitempty,max=30"`
	Address *string `json:"address" validate:"omitempty,max=500"`
	Notes   *string `json:"notes" validate:"omitempty,max=1000"`
}

// CreateDeviceRequest adds a device to a customer.
type CreateDeviceRequest struct {
	Brand string  `json:"brand" validate:"required,max=60"`
	Model string  `json:"model" validate:"required,max=100"`
	IMEI  *string `json:"imei" validate:"omitempty,max=20"`
	Color *string `json:"color" validate:"omitempty,max=40"`
	Notes *string `json:"notes" validate:"omitempty,max=1000"`
}

// ServiceCustomerInput describes an existing or new customer in one call.
type ServiceCustomerInput struct {
	ID      *string `json:"id"`
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Address *string `json:"address"`
}

// ServiceDeviceInput describes the device received at the counter.
type ServiceDeviceInput struct {
	Brand string  `json:"brand"`
	Model string  `json:"model"`
	IMEI  *string `json:"imei"`
	Color *string `json:"color"`
}

// CredentialInput stores a write-only device credential.
type CredentialInput struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// CreateServiceRequest is the intake payload.
type CreateServiceRequest struct {
	Customer             ServiceCustomerInput `json:"customer"`
	Device               ServiceDeviceInput   `json:"device"`
	Complaint            string               `json:"complaint" validate:"required,max=1000"`
	Completeness         *string              `json:"completeness" validate:"omitempty,max=2000"`
	PhysicalCondition    *string              `json:"physical_condition" validate:"omitempty,max=2000"`
	Credential           *CredentialInput     `json:"credential"`
	EstimatedCost        int64                `json:"estimated_cost" validate:"gte=0"`
	EstimatedDoneAt      *string              `json:"estimated_done_at"`
	AssignedTechnicianID *string              `json:"assigned_technician_id"`
	Priority             string               `json:"priority" validate:"omitempty,oneof=normal express"`
	Notes                *string              `json:"notes" validate:"omitempty,max=2000"`
	InternalNotes        *string              `json:"internal_notes" validate:"omitempty,max=2000"`
}

// UpdateServiceRequest edits intake fields.
type UpdateServiceRequest struct {
	Complaint            *string `json:"complaint" validate:"omitempty,max=1000"`
	Completeness         *string `json:"completeness" validate:"omitempty,max=2000"`
	PhysicalCondition    *string `json:"physical_condition" validate:"omitempty,max=2000"`
	EstimatedCost        *int64  `json:"estimated_cost" validate:"omitempty,gte=0"`
	EstimatedDoneAt      *string `json:"estimated_done_at"`
	AssignedTechnicianID *string `json:"assigned_technician_id"`
	Priority             *string `json:"priority" validate:"omitempty,oneof=normal express"`
	Notes                *string `json:"notes" validate:"omitempty,max=2000"`
	InternalNotes        *string `json:"internal_notes" validate:"omitempty,max=2000"`
}

// CredentialRequest sets or reveals a credential.
type CredentialRequest struct {
	Type   string `json:"type" validate:"required,oneof=pin password pattern"`
	Value  string `json:"value" validate:"required,max=200"`
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

// RevealCredentialRequest requires an audit reason.
type RevealCredentialRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

// CustomerRef is embedded in a service order.
type CustomerRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// DeviceRef is embedded in a service order.
type DeviceRef struct {
	ID    string  `json:"id"`
	Brand string  `json:"brand"`
	Model string  `json:"model"`
	IMEI  *string `json:"imei"`
	Color *string `json:"color"`
}

// TechnicianRef identifies an assigned technician.
type TechnicianRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ServiceOrderListItem is the row shown in the service list.
type ServiceOrderListItem struct {
	ID                   string         `json:"id"`
	TicketNo             string         `json:"ticket_no"`
	Status               string         `json:"status"`
	Priority             string         `json:"priority"`
	Customer             CustomerRef    `json:"customer"`
	Device               DeviceRef      `json:"device"`
	AssignedTechnician   *TechnicianRef `json:"assigned_technician"`
	Complaint            string         `json:"complaint"`
	EstimatedCost        int64          `json:"estimated_cost"`
	EstimatedDoneAt      *time.Time     `json:"estimated_done_at"`
	ReceivedAt           time.Time      `json:"received_at"`
	CompletedAt          *time.Time     `json:"completed_at"`
	PickedUpAt           *time.Time     `json:"picked_up_at"`
	CredentialConfigured bool           `json:"credential_configured"`
	PhotoCount           int64          `json:"photo_count"`
}

// ServiceOrderList is a paginated service result.
type ServiceOrderList struct {
	Items []ServiceOrderListItem
	Total int64
}

// ServicePhoto is one uploaded intake/service photo.
type ServicePhoto struct {
	ID        string    `json:"id"`
	Stage     string    `json:"stage"`
	FilePath  string    `json:"file_path"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
}

// StatusHistoryEntry is one immutable status transition.
type StatusHistoryEntry struct {
	ID          string    `json:"id"`
	FromStatus  *string   `json:"from_status"`
	ToStatus    string    `json:"to_status"`
	Notes       *string   `json:"notes"`
	ChangedByID *string   `json:"changed_by_id"`
	ChangedBy   *string   `json:"changed_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// InvoiceRef is the draft invoice attached to an order.
type InvoiceRef struct {
	ID              string `json:"id"`
	InvoiceNo       string `json:"invoice_no"`
	Status          string `json:"status"`
	TotalAmount     int64  `json:"total_amount"`
	PaidAmount      int64  `json:"paid_amount"`
	RemainingAmount int64  `json:"remaining_amount"`
}

// StoreRef is the store header used on printed labels.
type StoreRef struct {
	Name    string  `json:"name"`
	Address *string `json:"address"`
	Phone   *string `json:"phone"`
}

// ServiceOrderDetail is the full service-order payload.
type ServiceOrderDetail struct {
	ServiceOrderListItem
	Completeness      *string               `json:"completeness"`
	PhysicalCondition *string               `json:"physical_condition"`
	Notes             *string               `json:"notes"`
	InternalNotes     *string               `json:"internal_notes"`
	CredentialType    string                `json:"credential_type"`
	Photos            []ServicePhoto        `json:"photos"`
	Timeline          []StatusHistoryEntry  `json:"timeline"`
	Invoice           *InvoiceRef           `json:"invoice"`
	Diagnostic        *DiagnosticDTO        `json:"diagnostic,omitempty"`
	Items             []ServiceOrderItemDTO `json:"items,omitempty"`
	Store             StoreRef              `json:"store"`
	PublicToken       *string               `json:"public_token"`
	PublicTrackingURL *string               `json:"tracking_url"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

// ServiceOrderCreated is returned by POST /service-orders.
type ServiceOrderCreated struct {
	ID          string    `json:"id"`
	TicketNo    string    `json:"ticket_no"`
	PublicToken string    `json:"public_token"`
	TrackingURL string    `json:"tracking_url"`
	Status      string    `json:"status"`
	CustomerID  string    `json:"customer_id"`
	DeviceID    string    `json:"device_id"`
	InvoiceNo   string    `json:"invoice_no"`
	CreatedAt   time.Time `json:"created_at"`
}

// TechnicianOption is used by the intake form and list filter.
type TechnicianOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ServiceFilter is the list query.
type ServiceFilter struct {
	Query        string
	Statuses     []string
	TechnicianID *uuid.UUID
	OutletID     uuid.UUID
	FromDate     string
	ToDate       string
	CustomerID   *uuid.UUID
	Page         int
	PerPage      int
}

// CustomerFilter is the customer list query.
type CustomerFilter struct {
	Query   string
	Page    int
	PerPage int
}

// DiagnosticDTO is the API representation of a diagnosis record.
type DiagnosticDTO struct {
	ID              string     `json:"id"`
	Diagnosis       *string    `json:"diagnosis"`
	Findings        *string    `json:"findings"`
	ActionTaken     *string    `json:"action_taken"`
	TestResult      *string    `json:"test_result"`
	TechnicianNotes *string    `json:"technician_notes"`
	DiagnosedByID   *string    `json:"diagnosed_by_id,omitempty"`
	DiagnosedBy     *string    `json:"diagnosed_by,omitempty"`
	DiagnosedAt     *time.Time `json:"diagnosed_at"`
	EstimatedDoneAt *time.Time `json:"estimated_done_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UpsertDiagnosticRequest is the payload to save diagnosis findings.
type UpsertDiagnosticRequest struct {
	Diagnosis       *string `json:"diagnosis" validate:"omitempty,max=2000"`
	Findings        *string `json:"findings" validate:"omitempty,max=2000"`
	ActionTaken     *string `json:"action_taken" validate:"omitempty,max=2000"`
	TestResult      *string `json:"test_result" validate:"omitempty,max=2000"`
	TechnicianNotes *string `json:"technician_notes" validate:"omitempty,max=2000"`
	EstimatedDoneAt *string `json:"estimated_done_at"`
}

// ChangeStatusRequest is the payload to transition service status.
type ChangeStatusRequest struct {
	Status         string  `json:"status" validate:"required"`
	Notes          *string `json:"notes" validate:"omitempty,max=1000"`
	CancelReason   *string `json:"cancel_reason" validate:"omitempty,max=1000"`
	OverrideReason *string `json:"override_reason" validate:"omitempty,max=1000"`
}

// AssignTechnicianRequest assigns a technician and optional priority.
type AssignTechnicianRequest struct {
	TechnicianID *string `json:"technician_id"`
	Priority     *string `json:"priority" validate:"omitempty,oneof=normal express"`
}

// ServiceOrderItemDTO represents an item line (service or sparepart).
type ServiceOrderItemDTO struct {
	ID             string    `json:"id"`
	ItemType       string    `json:"item_type"`
	SparepartID    *string   `json:"sparepart_id,omitempty"`
	Description    string    `json:"description"`
	Qty            float64   `json:"qty"`
	UnitPrice      int64     `json:"unit_price"`
	CostPrice      int64     `json:"cost_price"`
	DiscountAmount int64     `json:"discount_amount"`
	Subtotal       int64     `json:"subtotal"`
	WarrantyDays   int       `json:"warranty_days"`
	AddedBy        *string   `json:"added_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddServiceItemRequest adds a service charge or sparepart line.
type AddServiceItemRequest struct {
	ItemType       string  `json:"item_type" validate:"required,oneof=service sparepart custom"`
	SparepartID    *string `json:"sparepart_id"`
	Description    string  `json:"description" validate:"omitempty,max=200"`
	Qty            float64 `json:"qty" validate:"required,gt=0"`
	UnitPrice      *int64  `json:"unit_price" validate:"omitempty,gte=0"`
	DiscountAmount int64   `json:"discount_amount" validate:"gte=0"`
	WarrantyDays   int     `json:"warranty_days" validate:"gte=0"`
}

// UpdateServiceItemRequest edits an existing line item.
type UpdateServiceItemRequest struct {
	Description    *string  `json:"description" validate:"omitempty,max=200"`
	Qty            *float64 `json:"qty" validate:"omitempty,gt=0"`
	UnitPrice      *int64   `json:"unit_price" validate:"omitempty,gte=0"`
	DiscountAmount *int64   `json:"discount_amount" validate:"omitempty,gte=0"`
	WarrantyDays   *int     `json:"warranty_days" validate:"omitempty,gte=0"`
}

// SparepartOption is returned when searching spareparts to add to a service.
type SparepartOption struct {
	ID        string `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Stock     int    `json:"stock"`
	CostPrice int64  `json:"cost_price"`
	SellPrice int64  `json:"sell_price"`
	Location  string `json:"location"`
}
