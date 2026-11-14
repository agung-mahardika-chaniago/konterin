// Package jobs implements background workers for the WhatsApp outbox, service
// reminders, and credential purging.
package jobs

import (
	"context"
	"strings"
	"time"

	"fixphone/backend/internal/modules/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Runner executes idempotent background jobs.
type Runner struct {
	db      *gorm.DB
	notif   *notification.Service
	baseURL string
}

// NewRunner builds the worker runner.
func NewRunner(db *gorm.DB, notif *notification.Service, baseURL string) *Runner {
	return &Runner{db: db, notif: notif, baseURL: baseURL}
}

// ReminderSummary reports how many reminders were emitted in one pass.
type ReminderSummary struct {
	OverdueServices    int
	UncollectedDevices int
	ExpiringWarranties int
}

type reminderOrder struct {
	ID            uuid.UUID `gorm:"column:id"`
	TenantID      uuid.UUID `gorm:"column:tenant_id"`
	OutletID      uuid.UUID `gorm:"column:outlet_id"`
	TicketNo      string    `gorm:"column:ticket_no"`
	CustomerID    uuid.UUID `gorm:"column:customer_id"`
	CustomerName  string    `gorm:"column:customer_name"`
	CustomerPhone string    `gorm:"column:customer_phone"`
	DeviceBrand   string    `gorm:"column:device_brand"`
	DeviceModel   string    `gorm:"column:device_model"`
	PublicToken   *string   `gorm:"column:public_token"`
	StoreName     string    `gorm:"column:store_name"`
	EndDate       time.Time `gorm:"column:end_date"`
}

// RunReminders emits overdue, uncollected, and expiring-warranty reminders.
// Every notification/outbox row carries a dedupe key so repeated runs do not
// spam staff or customers.
func (r *Runner) RunReminders(ctx context.Context) (ReminderSummary, error) {
	summary := ReminderSummary{}
	today := time.Now().UTC().Format("2006-01-02")

	overdue, err := r.overdueServices(ctx)
	if err != nil {
		return summary, err
	}
	for _, row := range overdue {
		if err := r.notifyOrder(ctx, row, "service_overdue",
			"Servis melewati estimasi: "+row.TicketNo,
			"Estimasi selesai untuk "+row.DeviceBrand+" "+row.DeviceModel+" telah terlewati.",
			"Servis "+row.TicketNo+" melewati estimasi selesai. Mohon segera ditindaklanjuti.",
			"overdue:"+row.ID.String()+":"+today); err != nil {
			return summary, err
		}
		summary.OverdueServices++
	}

	uncollected, err := r.uncollectedDevices(ctx)
	if err != nil {
		return summary, err
	}
	for _, row := range uncollected {
		if err := r.notifyOrder(ctx, row, "service_uncollected",
			"Perangkat belum diambil: "+row.TicketNo,
			row.DeviceBrand+" "+row.DeviceModel+" sudah siap diambil lebih dari 3 hari.",
			"Perangkat untuk tiket "+row.TicketNo+" belum diambil. Silakan hubungi pelanggan.",
			"uncollected:"+row.ID.String()+":"+today); err != nil {
			return summary, err
		}
		summary.UncollectedDevices++
	}

	expiring, err := r.expiringWarranties(ctx)
	if err != nil {
		return summary, err
	}
	for _, row := range expiring {
		if err := r.notifyOrder(ctx, row, "warranty_expiring",
			"Garansi segera berakhir: "+row.TicketNo,
			"Garansi untuk "+row.DeviceBrand+" "+row.DeviceModel+" berakhir pada "+row.EndDate.Format("02 Jan 2006")+".",
			"Garansi servis "+row.TicketNo+" berakhir pada "+row.EndDate.Format("2006-01-02")+".",
			"warranty-expiring:"+row.ID.String()); err != nil {
			return summary, err
		}
		summary.ExpiringWarranties++
	}
	return summary, nil
}

func (r *Runner) notifyOrder(ctx context.Context, row reminderOrder, typ, title, body, reminderText, dedupePrefix string) error {
	link := "/services/" + row.ID.String()
	entityType := "service_order"
	var entityID *uuid.UUID
	var serviceOrderID *uuid.UUID
	if typ == "warranty_expiring" {
		link = "/warranties/" + row.ID.String()
		entityType = "warranty"
		entityID = &row.ID
	} else {
		entityID = &row.ID
		serviceOrderID = &row.ID
	}
	if err := r.notif.CreateBroadcast(ctx, nil, notification.BroadcastInput{
		TenantID: row.TenantID, Type: typ, Title: title, Body: &body,
		Link: &link, EntityType: &entityType, EntityID: entityID,
		DedupeKey: dedupePrefix,
	}); err != nil {
		return err
	}
	if strings.TrimSpace(row.CustomerPhone) == "" {
		return nil
	}
	trackingURL := ""
	if row.PublicToken != nil && strings.TrimSpace(*row.PublicToken) != "" {
		trackingURL = strings.TrimRight(r.baseURL, "/") + "/track/" + strings.TrimSpace(*row.PublicToken)
	}
	_, err := r.notif.Enqueue(ctx, nil, notification.EnqueueInput{
		TenantID: row.TenantID, OutletID: row.OutletID,
		ServiceOrderID: serviceOrderID, CustomerID: &row.CustomerID,
		Phone: row.CustomerPhone, TemplateKey: notification.TemplateReminder,
		Payload: map[string]any{
			"customer_name": row.CustomerName,
			"ticket_no":     row.TicketNo,
			"store_name":    row.StoreName,
			"device":        strings.TrimSpace(row.DeviceBrand + " " + row.DeviceModel),
			"reminder_text": reminderText,
			"tracking_url":  trackingURL,
		},
		DedupeKey: "wa-" + dedupePrefix,
	})
	return err
}

func (r *Runner) overdueServices(ctx context.Context) ([]reminderOrder, error) {
	rows := make([]reminderOrder, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT so.id, so.tenant_id, so.outlet_id, so.ticket_no, so.customer_id,
		       c.name AS customer_name, c.phone AS customer_phone,
		       so.device_brand, so.device_model, so.public_token,
		       COALESCE((
		           SELECT b.store_name FROM business_settings b
		           WHERE b.tenant_id = so.tenant_id AND (b.outlet_id = so.outlet_id OR b.outlet_id IS NULL)
		           ORDER BY (b.outlet_id IS NULL) LIMIT 1
		       ), 'FixPhone') AS store_name
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		WHERE so.status IN ('received', 'waiting_diagnosis', 'waiting_approval',
		                    'waiting_part', 'in_progress', 'testing')
		  AND so.estimated_done_at IS NOT NULL
		  AND so.estimated_done_at < now()`).Scan(&rows).Error
	return rows, err
}

func (r *Runner) uncollectedDevices(ctx context.Context) ([]reminderOrder, error) {
	rows := make([]reminderOrder, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT so.id, so.tenant_id, so.outlet_id, so.ticket_no, so.customer_id,
		       c.name AS customer_name, c.phone AS customer_phone,
		       so.device_brand, so.device_model, so.public_token,
		       COALESCE((
		           SELECT b.store_name FROM business_settings b
		           WHERE b.tenant_id = so.tenant_id AND (b.outlet_id = so.outlet_id OR b.outlet_id IS NULL)
		           ORDER BY (b.outlet_id IS NULL) LIMIT 1
		       ), 'FixPhone') AS store_name
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		WHERE so.status = 'ready_for_pickup'
		  AND so.completed_at IS NOT NULL
		  AND so.completed_at < now() - interval '3 days'`).Scan(&rows).Error
	return rows, err
}

func (r *Runner) expiringWarranties(ctx context.Context) ([]reminderOrder, error) {
	rows := make([]reminderOrder, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT w.id, w.tenant_id, w.outlet_id, so.ticket_no, so.customer_id,
		       c.name AS customer_name, c.phone AS customer_phone,
		       so.device_brand, so.device_model, so.public_token, w.end_date,
		       COALESCE((
		           SELECT b.store_name FROM business_settings b
		           WHERE b.tenant_id = w.tenant_id AND (b.outlet_id = w.outlet_id OR b.outlet_id IS NULL)
		           ORDER BY (b.outlet_id IS NULL) LIMIT 1
		       ), 'FixPhone') AS store_name
		FROM warranties w
		JOIN service_orders so ON so.id = w.service_order_id
		JOIN customers c ON c.id = w.customer_id
		WHERE w.status = 'active'
		  AND w.end_date BETWEEN CURRENT_DATE AND CURRENT_DATE + 7`).Scan(&rows).Error
	return rows, err
}

// ProcessOutbox drains the queue in batches. The outbox itself claims rows with
// SKIP LOCKED, so multiple workers are safe.
func (r *Runner) ProcessOutbox(ctx context.Context) (sent, failed int, err error) {
	for i := 0; i < 20; i++ {
		batchSent, batchFailed, processErr := r.notif.ProcessOutbox(ctx, 50)
		sent += batchSent
		failed += batchFailed
		if processErr != nil {
			return sent, failed, processErr
		}
		if batchSent+batchFailed == 0 {
			break
		}
	}
	return sent, failed, nil
}

// PurgeCredentials clears device credentials 30 days after pickup.
func (r *Runner) PurgeCredentials(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE service_orders
		SET credential_type = 'none', credential_enc = NULL,
		    credential_key_version = NULL, credential_set_at = NULL, updated_at = now()
		WHERE credential_purge_at IS NOT NULL AND credential_purge_at <= now()
		  AND credential_type <> 'none'`)
	return result.RowsAffected, result.Error
}
