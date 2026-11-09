// Package tracking exposes the public, privacy-filtered service status page.
package tracking

import (
	"time"
)

// StoreInfo is the public store identity.
type StoreInfo struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}

// WarrantyInfo is the public warranty summary.
type WarrantyInfo struct {
	Active  bool    `json:"active"`
	Status  string  `json:"status"`
	Days    int     `json:"days"`
	EndDate *string `json:"end_date"`
}

// PublicTracking is the whitelisted response for /public/track. It deliberately
// omits customer identity, address, IMEI, credentials, internal notes, photos,
// cost prices, and technician names.
type PublicTracking struct {
	TicketNo           string       `json:"ticket_no"`
	Store              StoreInfo    `json:"store"`
	Status             string       `json:"status"`
	StatusLabel        string       `json:"status_label"`
	ReceivedAt         time.Time    `json:"received_at"`
	EstimatedDoneAt    *time.Time   `json:"estimated_done_at"`
	Device             string       `json:"device"`
	Complaint          string       `json:"complaint"`
	Diagnosis          *string      `json:"diagnosis"`
	InvoiceStatus      *string      `json:"invoice_status"`
	PaymentStatusLabel string       `json:"payment_status_label"`
	PickupStatus       string       `json:"pickup_status"`
	Warranty           WarrantyInfo `json:"warranty"`
}
