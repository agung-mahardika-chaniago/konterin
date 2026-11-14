// Package settings exposes tenant business settings for Stage 10.
package settings

import "time"

// Settings is the tenant-level business settings payload.
type Settings struct {
	StoreName                  string      `json:"store_name"`
	LegalName                  *string     `json:"legal_name"`
	LogoPath                   *string     `json:"logo_path"`
	Address                    *string     `json:"address"`
	Phone                      *string     `json:"phone"`
	Email                      *string     `json:"email"`
	TaxID                      *string     `json:"tax_id"`
	Timezone                   string      `json:"timezone"`
	TicketPrefix               string      `json:"ticket_prefix"`
	TicketFormat               string      `json:"ticket_format"`
	InvoicePrefix              string      `json:"invoice_prefix"`
	InvoiceFormat              string      `json:"invoice_format"`
	InvoiceTerms               *string     `json:"invoice_terms"`
	InvoiceFooter              *string     `json:"invoice_footer"`
	DefaultServiceWarrantyDays int         `json:"default_service_warranty_days"`
	DefaultPartWarrantyDays    int         `json:"default_part_warranty_days"`
	ReceiptPaperWidth          int         `json:"receipt_paper_width"`
	TechnicianSeesOnlyAssigned bool        `json:"technician_sees_only_assigned"`
	PublicTrackingEnabled      bool        `json:"public_tracking_enabled"`
	WAProvider                 string      `json:"wa_provider"`
	WATemplates                WATemplates `json:"wa_templates"`
	UpdatedAt                  time.Time   `json:"updated_at"`
}

// WATemplates holds editable WhatsApp message templates.
type WATemplates struct {
	StatusUpdate  string `json:"status_update"`
	ReadyPickup   string `json:"ready_for_pickup"`
	Invoice       string `json:"invoice"`
	Reminder      string `json:"reminder"`
	Warranty      string `json:"warranty"`
	ClaimReceived string `json:"claim_received"`
}

// UpdateSettingsRequest is the partial update payload for settings.
type UpdateSettingsRequest struct {
	StoreName                  *string      `json:"store_name" validate:"omitempty,max=150"`
	LegalName                  *string      `json:"legal_name" validate:"omitempty,max=150"`
	Address                    *string      `json:"address" validate:"omitempty"`
	Phone                      *string      `json:"phone" validate:"omitempty,max=30"`
	Email                      *string      `json:"email" validate:"omitempty,email,max=150"`
	TaxID                      *string      `json:"tax_id" validate:"omitempty,max=30"`
	Timezone                   *string      `json:"timezone" validate:"omitempty,max=50"`
	TicketPrefix               *string      `json:"ticket_prefix" validate:"omitempty,max=10"`
	TicketFormat               *string      `json:"ticket_format" validate:"omitempty,max=60"`
	InvoicePrefix              *string      `json:"invoice_prefix" validate:"omitempty,max=10"`
	InvoiceFormat              *string      `json:"invoice_format" validate:"omitempty,max=60"`
	InvoiceTerms               *string      `json:"invoice_terms" validate:"omitempty"`
	InvoiceFooter              *string      `json:"invoice_footer" validate:"omitempty"`
	DefaultServiceWarrantyDays *int         `json:"default_service_warranty_days" validate:"omitempty,gte=0,lte=3650"`
	DefaultPartWarrantyDays    *int         `json:"default_part_warranty_days" validate:"omitempty,gte=0,lte=3650"`
	ReceiptPaperWidth          *int         `json:"receipt_paper_width" validate:"omitempty,oneof=58 80"`
	TechnicianSeesOnlyAssigned *bool        `json:"technician_sees_only_assigned"`
	PublicTrackingEnabled      *bool        `json:"public_tracking_enabled"`
	WAProvider                 *string      `json:"wa_provider" validate:"omitempty,oneof=noop fonnte wablas"`
	WATemplates                *WATemplates `json:"wa_templates"`
}

// Backup is one manual backup metadata row.
type Backup struct {
	ID         string    `json:"id"`
	FileName   string    `json:"file_name"`
	SizeBytes  int64     `json:"size_bytes"`
	Status     string    `json:"status"`
	TableCount int       `json:"table_count"`
	RowCount   int64     `json:"row_count"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  *string   `json:"created_by"`
}
