// Package notification implements staff notifications and the WhatsApp outbox.
package notification

import (
	"time"

	"github.com/google/uuid"
)

// WhatsApp outbox statuses accepted by the database constraint.
const (
	WAStatusQueued     = "queued"
	WAStatusProcessing = "processing"
	WAStatusSent       = "sent"
	WAStatusFailed     = "failed"
	WAStatusCancelled  = "cancelled"
)

// Template keys understood by the notification module.
const (
	TemplateStatusUpdate  = "status_update"
	TemplateReadyPickup   = "ready_pickup"
	TemplateInvoice       = "invoice"
	TemplateReminder      = "reminder"
	TemplateWarranty      = "warranty"
	TemplateClaimReceived = "claim_received"
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

// Notification is the API projection of an in-app notification.
type Notification struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Body        *string    `json:"body"`
	Link        *string    `json:"link"`
	EntityType  *string    `json:"entity_type"`
	EntityID    *string    `json:"entity_id"`
	IsRead      bool       `json:"is_read"`
	ReadAt      *time.Time `json:"read_at"`
	IsBroadcast bool       `json:"is_broadcast"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NotificationFilter narrows the notification list.
type NotificationFilter struct {
	Unread  bool
	Page    int
	PerPage int
}

// WhatsAppMessage is the API projection of an outbox row.
type WhatsAppMessage struct {
	ID             string     `json:"id"`
	ServiceOrderID *string    `json:"service_order_id"`
	TicketNo       *string    `json:"ticket_no"`
	CustomerName   *string    `json:"customer_name"`
	CustomerID     *string    `json:"customer_id"`
	Phone          string     `json:"phone"`
	TemplateKey    string     `json:"template_key"`
	Body           *string    `json:"body"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	LastError      *string    `json:"last_error"`
	Provider       string     `json:"provider"`
	ProviderMsgID  *string    `json:"provider_message_id"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	SentAt         *time.Time `json:"sent_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// OutboxFilter narrows the WhatsApp outbox list.
type OutboxFilter struct {
	Status  string
	Query   string
	Page    int
	PerPage int
}

// EnqueueInput describes a WhatsApp message to queue.
type EnqueueInput struct {
	TenantID       uuid.UUID
	OutletID       uuid.UUID
	ServiceOrderID *uuid.UUID
	CustomerID     *uuid.UUID
	Phone          string
	TemplateKey    string
	Payload        map[string]any
	Body           string
	DedupeKey      string
	ScheduledAt    time.Time
}

// NotifyRequest is the POST /service-orders/:id/notify payload.
type NotifyRequest struct {
	TemplateKey string `json:"template_key"`
	Message     string `json:"message"`
}

// NotifyResult is returned after a message enters the outbox.
type NotifyResult struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

// ServiceStatusEvent is emitted by the service module inside its transaction.
type ServiceStatusEvent struct {
	TenantID   uuid.UUID
	OutletID   uuid.UUID
	OrderID    uuid.UUID
	TicketNo   string
	FromStatus string
	ToStatus   string
	ActorID    *uuid.UUID
	PickedUpAt time.Time
}

// ServiceDeps configures external WhatsApp providers.
type ServiceDeps struct {
	AppBaseURL   string
	FonnteToken  string
	WablasToken  string
	WablasSecret string
	WablasURL    string
}

// statusLabels keeps public-facing Indonesian labels in one place.
var statusLabels = map[string]string{
	"received":          "Diterima",
	"waiting_diagnosis": "Menunggu Diagnosis",
	"waiting_approval":  "Menunggu Persetujuan",
	"waiting_part":      "Menunggu Sparepart",
	"in_progress":       "Sedang Diperbaiki",
	"testing":           "Sedang Diuji",
	"completed":         "Selesai Dikerjakan",
	"ready_for_pickup":  "Siap Diambil",
	"picked_up":         "Sudah Diambil",
	"cancelled":         "Dibatalkan",
}

// StatusLabel returns the Indonesian label for a service status.
func StatusLabel(status string) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return status
}
