package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service implements notifications and the WhatsApp outbox.
type Service struct {
	db     *gorm.DB
	deps   ServiceDeps
	client *http.Client
}

// NewService builds the notification module.
func NewService(db *gorm.DB, deps ServiceDeps) *Service {
	return &Service{db: db, deps: deps, client: &http.Client{Timeout: 15 * time.Second}}
}

type businessSettings struct {
	StoreName      string          `gorm:"column:store_name"`
	WAProvider     string          `gorm:"column:wa_provider"`
	WATemplates    json.RawMessage `gorm:"column:wa_templates"`
	PublicTracking bool            `gorm:"column:public_tracking_enabled"`
}

func settingsFor(ctx context.Context, q *gorm.DB, tenantID, outletID uuid.UUID) (businessSettings, error) {
	settings := businessSettings{}
	err := q.WithContext(ctx).Raw(`
		SELECT store_name, COALESCE(wa_provider, 'noop') AS wa_provider,
		       COALESCE(wa_templates, '{}'::jsonb) AS wa_templates,
		       public_tracking_enabled
		FROM business_settings
		WHERE tenant_id = ? AND (outlet_id = ? OR outlet_id IS NULL)
		ORDER BY (outlet_id IS NULL)
		LIMIT 1`, tenantID, outletID).Scan(&settings).Error
	if err != nil {
		return settings, err
	}
	if strings.TrimSpace(settings.StoreName) == "" {
		settings.StoreName = "FixPhone"
	}
	if strings.TrimSpace(settings.WAProvider) == "" {
		settings.WAProvider = "noop"
	}
	return settings, nil
}

// defaultTemplates are used when the tenant has not customised wa_templates.
var defaultTemplates = map[string]string{
	TemplateStatusUpdate:  "Halo {customer_name}, status servis {ticket_no} di {store_name} saat ini: {status_label}. Lacak progres: {tracking_url}",
	TemplateReadyPickup:   "Halo {customer_name}, perangkat {device} untuk tiket {ticket_no} sudah siap diambil di {store_name}. Silakan datang dengan membawa nota servis.",
	TemplateInvoice:       "Halo {customer_name}, invoice {invoice_no} untuk tiket {ticket_no} sebesar {total_amount}. Terima kasih.",
	TemplateReminder:      "Halo {customer_name}, pengingat servis {ticket_no} ({device}) di {store_name}: {reminder_text}",
	TemplateWarranty:      "Halo {customer_name}, garansi servis {ticket_no} berlaku sampai {warranty_end_date}. Simpan pesan ini sebagai bukti garansi.",
	TemplateClaimReceived: "Halo {customer_name}, klaim garansi {claim_no} untuk tiket {ticket_no} telah kami terima. Kami akan menghubungi Anda untuk tindak lanjut.",
}

func templateBody(settings businessSettings, key string) string {
	if len(settings.WATemplates) > 0 {
		var templates map[string]any
		if err := json.Unmarshal(settings.WATemplates, &templates); err == nil {
			if raw, ok := templates[key]; ok {
				switch value := raw.(type) {
				case string:
					if strings.TrimSpace(value) != "" {
						return value
					}
				case map[string]any:
					if body, ok := value["body"].(string); ok && strings.TrimSpace(body) != "" {
						return body
					}
				}
			}
		}
	}
	if body, ok := defaultTemplates[key]; ok {
		return body
	}
	return defaultTemplates[TemplateStatusUpdate]
}

func renderTemplate(body string, vars map[string]any) string {
	for key, value := range vars {
		body = strings.ReplaceAll(body, "{"+key+"}", fmt.Sprint(value))
	}
	return body
}

func normalizePhone(raw string) string {
	var digits strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	value := digits.String()
	switch {
	case strings.HasPrefix(value, "62"):
		value = "62" + strings.TrimPrefix(value, "62")
	case strings.HasPrefix(value, "0"):
		value = "62" + strings.TrimPrefix(value, "0")
	case strings.HasPrefix(value, "8"):
		value = "62" + value
	}
	return value
}

type orderContext struct {
	ID            uuid.UUID `gorm:"column:id"`
	TenantID      uuid.UUID `gorm:"column:tenant_id"`
	OutletID      uuid.UUID `gorm:"column:outlet_id"`
	TicketNo      string    `gorm:"column:ticket_no"`
	Status        string    `gorm:"column:status"`
	CustomerID    uuid.UUID `gorm:"column:customer_id"`
	CustomerName  string    `gorm:"column:customer_name"`
	CustomerPhone string    `gorm:"column:customer_phone"`
	DeviceBrand   string    `gorm:"column:device_brand"`
	DeviceModel   string    `gorm:"column:device_model"`
	PublicToken   *string   `gorm:"column:public_token"`
	InvoiceNo     *string   `gorm:"column:invoice_no"`
	TotalAmount   int64     `gorm:"column:total_amount"`
}

func loadOrderContext(ctx context.Context, q *gorm.DB, tenantID, orderID uuid.UUID) (orderContext, error) {
	row := orderContext{}
	err := q.WithContext(ctx).Raw(`
		SELECT so.id, so.tenant_id, so.outlet_id, so.ticket_no, so.status,
		       so.customer_id, c.name AS customer_name, c.phone AS customer_phone,
		       so.device_brand, so.device_model, so.public_token,
		       i.invoice_no, COALESCE(i.total_amount, 0) AS total_amount
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		LEFT JOIN invoices i ON i.service_order_id = so.id
		WHERE so.id = ? AND so.tenant_id = ?`, orderID, tenantID).Scan(&row).Error
	if err != nil {
		return row, err
	}
	if row.ID == uuid.Nil {
		return row, apperror.NotFound("Servis tidak ditemukan.")
	}
	return row, nil
}

func (s *Service) varsFor(row orderContext, settings businessSettings, baseURL string) map[string]any {
	device := strings.TrimSpace(row.DeviceBrand + " " + row.DeviceModel)
	trackingURL := ""
	if row.PublicToken != nil && strings.TrimSpace(*row.PublicToken) != "" {
		trackingURL = strings.TrimRight(baseURL, "/") + "/track/" + *row.PublicToken
	}
	invoiceNo := ""
	if row.InvoiceNo != nil {
		invoiceNo = *row.InvoiceNo
	}
	return map[string]any{
		"customer_name": row.CustomerName,
		"ticket_no":     row.TicketNo,
		"store_name":    settings.StoreName,
		"status":        row.Status,
		"status_label":  StatusLabel(row.Status),
		"device":        device,
		"tracking_url":  trackingURL,
		"invoice_no":    invoiceNo,
		"total_amount":  row.TotalAmount,
	}
}

// Enqueue inserts a WhatsApp message into the outbox. It is safe to call inside
// the caller's transaction; duplicate dedupe keys resolve to the existing row.
func (s *Service) Enqueue(ctx context.Context, tx *gorm.DB, input EnqueueInput) (NotifyResult, error) {
	q := tx
	if q == nil {
		q = s.db
	}
	phone := normalizePhone(input.Phone)
	if phone == "" {
		return NotifyResult{}, errors.New("nomor WhatsApp pelanggan tidak tersedia")
	}
	if strings.TrimSpace(input.TemplateKey) == "" {
		input.TemplateKey = TemplateStatusUpdate
	}
	settings, err := settingsFor(ctx, q, input.TenantID, input.OutletID)
	if err != nil {
		return NotifyResult{}, err
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		body = renderTemplate(templateBody(settings, input.TemplateKey), input.Payload)
	}
	if strings.TrimSpace(body) == "" {
		return NotifyResult{}, errors.New("isi pesan WhatsApp kosong")
	}
	if len(body) > 4000 {
		body = body[:4000]
	}
	if input.ScheduledAt.IsZero() {
		input.ScheduledAt = time.Now().UTC()
	}

	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return NotifyResult{}, err
	}
	var dedupe *string
	if strings.TrimSpace(input.DedupeKey) != "" {
		value := strings.TrimSpace(input.DedupeKey)
		dedupe = &value
	}

	id, err := uuid.NewV7()
	if err != nil {
		return NotifyResult{}, err
	}
	var row struct {
		ID     uuid.UUID `gorm:"column:id"`
		Status string    `gorm:"column:status"`
	}
	result := q.WithContext(ctx).Raw(`
		INSERT INTO whatsapp_messages (
			id, tenant_id, outlet_id, service_order_id, customer_id, phone,
			template_key, payload, body, status, provider, scheduled_at, dedupe_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, 'queued', ?, ?, ?)
		ON CONFLICT DO NOTHING
		RETURNING id, status`,
		id, input.TenantID, input.OutletID, input.ServiceOrderID, input.CustomerID,
		phone, input.TemplateKey, string(payload), body, settings.WAProvider,
		input.ScheduledAt, dedupe).Scan(&row)
	if result.Error != nil {
		return NotifyResult{}, result.Error
	}
	if result.RowsAffected == 0 && dedupe != nil {
		if err := q.WithContext(ctx).Raw(`
			SELECT id, status FROM whatsapp_messages
			WHERE tenant_id = ? AND dedupe_key = ?`, input.TenantID, *dedupe).Scan(&row).Error; err != nil {
			return NotifyResult{}, err
		}
	}
	if row.ID == uuid.Nil {
		return NotifyResult{}, errors.New("gagal menyimpan pesan WhatsApp")
	}
	return NotifyResult{MessageID: row.ID.String(), Status: row.Status}, nil
}

// BroadcastInput describes an in-app notification.
type BroadcastInput struct {
	TenantID   uuid.UUID
	UserID     *uuid.UUID
	Type       string
	Title      string
	Body       *string
	Link       *string
	EntityType *string
	EntityID   *uuid.UUID
	DedupeKey  string
}

// CreateBroadcast inserts an in-app notification idempotently.
func (s *Service) CreateBroadcast(ctx context.Context, tx *gorm.DB, input BroadcastInput) error {
	q := tx
	if q == nil {
		q = s.db
	}
	var dedupe *string
	if strings.TrimSpace(input.DedupeKey) != "" {
		value := strings.TrimSpace(input.DedupeKey)
		dedupe = &value
	}
	return q.WithContext(ctx).Exec(`
		INSERT INTO notifications (
			tenant_id, user_id, type, title, body, link, entity_type, entity_id, dedupe_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING`,
		input.TenantID, input.UserID, input.Type, input.Title, input.Body,
		input.Link, input.EntityType, input.EntityID, dedupe).Error
}

// OnServiceStatus is wired into the service module transaction. It queues the
// customer-facing WhatsApp message and an in-app notification for hand-over
// milestones without ever blocking unrelated statuses.
func (s *Service) OnServiceStatus(ctx context.Context, tx *gorm.DB, event ServiceStatusEvent) error {
	if tx == nil {
		tx = s.db
	}
	switch event.ToStatus {
	case "ready_for_pickup", "picked_up":
	default:
		return nil
	}
	row, err := loadOrderContext(ctx, tx, event.TenantID, event.OrderID)
	if err != nil {
		return err
	}
	settings, err := settingsFor(ctx, tx, row.TenantID, row.OutletID)
	if err != nil {
		return err
	}
	link := "/services/" + row.ID.String()
	entityType := "service_order"

	if event.ToStatus == "ready_for_pickup" {
		title := "Servis siap diambil: " + row.TicketNo
		body := row.DeviceBrand + " " + row.DeviceModel + " untuk " + row.CustomerName + " siap diserahkan."
		if err := s.CreateBroadcast(ctx, tx, BroadcastInput{
			TenantID: row.TenantID, Type: "service_ready", Title: title,
			Body: &body, Link: &link, EntityType: &entityType, EntityID: &row.ID,
			DedupeKey: "service-ready:" + row.ID.String(),
		}); err != nil {
			return err
		}
		if strings.TrimSpace(row.CustomerPhone) != "" {
			if _, err := s.Enqueue(ctx, tx, EnqueueInput{
				TenantID: row.TenantID, OutletID: row.OutletID,
				ServiceOrderID: &row.ID, CustomerID: &row.CustomerID,
				Phone: row.CustomerPhone, TemplateKey: TemplateReadyPickup,
				Payload:   s.varsFor(row, settings, s.deps.AppBaseURL),
				DedupeKey: "wa-ready:" + row.ID.String(),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	title := "Servis telah diambil: " + row.TicketNo
	body := row.DeviceBrand + " " + row.DeviceModel + " telah diserahkan kepada " + row.CustomerName + "."
	return s.CreateBroadcast(ctx, tx, BroadcastInput{
		TenantID: row.TenantID, Type: "service_picked", Title: title,
		Body: &body, Link: &link, EntityType: &entityType, EntityID: &row.ID,
		DedupeKey: "service-picked:" + row.ID.String(),
	})
}

// OnClaimCreated is wired into the warranty transaction. It records an in-app
// notification and queues the claim acknowledgement for WhatsApp.
func (s *Service) OnClaimCreated(ctx context.Context, tx *gorm.DB, tenantID, outletID, claimID uuid.UUID) error {
	if tx == nil {
		tx = s.db
	}
	var row struct {
		ID             uuid.UUID `gorm:"column:id"`
		ClaimNo        string    `gorm:"column:claim_no"`
		Complaint      string    `gorm:"column:complaint"`
		TicketNo       string    `gorm:"column:ticket_no"`
		CustomerID     uuid.UUID `gorm:"column:customer_id"`
		CustomerName   string    `gorm:"column:customer_name"`
		CustomerPhone  string    `gorm:"column:customer_phone"`
		DeviceBrand    string    `gorm:"column:device_brand"`
		DeviceModel    string    `gorm:"column:device_model"`
		WarrantyEnd    time.Time `gorm:"column:end_date"`
		ServiceOrderID uuid.UUID `gorm:"column:service_order_id"`
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT wc.id, wc.claim_no, wc.complaint, so.ticket_no, so.id AS service_order_id,
		       c.id AS customer_id, c.name AS customer_name, c.phone AS customer_phone,
		       so.device_brand, so.device_model, w.end_date
		FROM warranty_claims wc
		JOIN warranties w ON w.id = wc.warranty_id
		JOIN service_orders so ON so.id = w.service_order_id
		JOIN customers c ON c.id = so.customer_id
		WHERE wc.id = ? AND wc.tenant_id = ?`, claimID, tenantID).Scan(&row).Error; err != nil {
		return err
	}
	if row.ID == uuid.Nil {
		return nil
	}
	settings, err := settingsFor(ctx, tx, tenantID, outletID)
	if err != nil {
		return err
	}
	link := "/claims/" + row.ID.String()
	entityType := "warranty_claim"
	title := "Klaim garansi baru: " + row.ClaimNo
	body := row.TicketNo + " - " + row.DeviceBrand + " " + row.DeviceModel + ": " + row.Complaint
	if len(body) > 240 {
		body = body[:240]
	}
	if err := s.CreateBroadcast(ctx, tx, BroadcastInput{
		TenantID: tenantID, Type: "warranty_claim", Title: title, Body: &body,
		Link: &link, EntityType: &entityType, EntityID: &row.ID,
		DedupeKey: "claim-created:" + row.ID.String(),
	}); err != nil {
		return err
	}
	if strings.TrimSpace(row.CustomerPhone) == "" {
		return nil
	}
	vars := map[string]any{
		"customer_name":     row.CustomerName,
		"ticket_no":         row.TicketNo,
		"store_name":        settings.StoreName,
		"device":            strings.TrimSpace(row.DeviceBrand + " " + row.DeviceModel),
		"claim_no":          row.ClaimNo,
		"warranty_end_date": row.WarrantyEnd.Format("2006-01-02"),
	}
	_, err = s.Enqueue(ctx, tx, EnqueueInput{
		TenantID: tenantID, OutletID: outletID,
		ServiceOrderID: &row.ServiceOrderID, CustomerID: &row.CustomerID,
		Phone: row.CustomerPhone, TemplateKey: TemplateClaimReceived,
		Payload: vars, DedupeKey: "wa-claim:" + row.ID.String(),
	})
	if err != nil {
		return err
	}
	return nil
}

// NotifyServiceOrder lets staff place a specific message in the outbox.
func (s *Service) NotifyServiceOrder(ctx context.Context, actor Actor, orderID uuid.UUID, input NotifyRequest) (NotifyResult, error) {
	if actor.TenantID == uuid.Nil {
		return NotifyResult{}, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	row, err := loadOrderContext(ctx, s.db, actor.TenantID, orderID)
	if err != nil {
		return NotifyResult{}, err
	}
	if !actorHasOutlet(actor, row.OutletID) {
		return NotifyResult{}, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
	}
	if strings.TrimSpace(row.CustomerPhone) == "" {
		return NotifyResult{}, apperror.Validation(map[string]string{
			"phone": "Pelanggan belum memiliki nomor WhatsApp.",
		})
	}
	settings, err := settingsFor(ctx, s.db, row.TenantID, row.OutletID)
	if err != nil {
		return NotifyResult{}, err
	}
	key := strings.TrimSpace(input.TemplateKey)
	if key == "" {
		key = TemplateStatusUpdate
	}
	body := strings.TrimSpace(input.Message)
	if body == "" {
		body = renderTemplate(templateBody(settings, key), s.varsFor(row, settings, s.deps.AppBaseURL))
	}
	result, err := s.Enqueue(ctx, nil, EnqueueInput{
		TenantID: row.TenantID, OutletID: row.OutletID,
		ServiceOrderID: &row.ID, CustomerID: &row.CustomerID,
		Phone: row.CustomerPhone, TemplateKey: key, Body: body,
	})
	if err != nil {
		return NotifyResult{}, apperror.Internal(err)
	}
	return result, nil
}

func actorHasOutlet(actor Actor, outletID uuid.UUID) bool {
	if actor.AllOutletAccess {
		for _, id := range actor.OutletIDs {
			if id == outletID {
				return true
			}
		}
		return false
	}
	for _, id := range actor.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

type notificationRow struct {
	ID          uuid.UUID  `gorm:"column:id"`
	Type        string     `gorm:"column:type"`
	Title       string     `gorm:"column:title"`
	Body        *string    `gorm:"column:body"`
	Link        *string    `gorm:"column:link"`
	EntityType  *string    `gorm:"column:entity_type"`
	EntityID    *uuid.UUID `gorm:"column:entity_id"`
	IsRead      bool       `gorm:"column:is_read_effective"`
	ReadAt      *time.Time `gorm:"column:read_at"`
	IsBroadcast bool       `gorm:"column:is_broadcast"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

// ListNotifications returns the current user's inbox plus tenant broadcasts.
func (s *Service) ListNotifications(ctx context.Context, actor Actor, filter NotificationFilter) ([]Notification, int64, int64, error) {
	if actor.TenantID == uuid.Nil || actor.UserID == uuid.Nil {
		return nil, 0, 0, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	where := ` WHERE (n.tenant_id = ? AND (n.user_id = ? OR n.user_id IS NULL))`
	args := []any{actor.UserID, actor.UserID, actor.TenantID, actor.UserID}
	having := ""
	if filter.Unread {
		having = ` WHERE is_read_effective = false`
	}
	query := `
		SELECT * FROM (
			SELECT n.id, n.type, n.title, n.body, n.link, n.entity_type, n.entity_id,
			       n.read_at, n.created_at, (n.user_id IS NULL) AS is_broadcast,
			       CASE WHEN n.user_id = ? THEN n.is_read
			            ELSE EXISTS (
			                SELECT 1 FROM notification_reads r
			                WHERE r.notification_id = n.id AND r.user_id = ?
			            ) END AS is_read_effective
			FROM notifications n` + where + `
		) t` + having + `
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?`
	rows := make([]notificationRow, 0, perPage)
	args = append(args, perPage, (page-1)*perPage)
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, 0, 0, apperror.Internal(err)
	}

	countQuery := `SELECT COUNT(*) FROM (
		SELECT n.id,
		       CASE WHEN n.user_id = ? THEN n.is_read
		            ELSE EXISTS (SELECT 1 FROM notification_reads r WHERE r.notification_id = n.id AND r.user_id = ?)
		       END AS is_read_effective
		FROM notifications n` + where + `
	) t` + having
	var total int64
	if err := s.db.WithContext(ctx).Raw(countQuery, args[:4]...).Scan(&total).Error; err != nil {
		return nil, 0, 0, apperror.Internal(err)
	}

	unread, err := s.UnreadCount(ctx, actor)
	if err != nil {
		return nil, 0, 0, err
	}

	items := make([]Notification, 0, len(rows))
	for _, r := range rows {
		item := Notification{
			ID: r.ID.String(), Type: r.Type, Title: r.Title, Body: r.Body,
			Link: r.Link, EntityType: r.EntityType, IsRead: r.IsRead,
			ReadAt: r.ReadAt, IsBroadcast: r.IsBroadcast, CreatedAt: r.CreatedAt,
		}
		if r.EntityID != nil {
			id := r.EntityID.String()
			item.EntityID = &id
		}
		items = append(items, item)
	}
	return items, total, unread, nil
}

// UnreadCount returns the number of unread notifications for the current user.
func (s *Service) UnreadCount(ctx context.Context, actor Actor) (int64, error) {
	if actor.TenantID == uuid.Nil || actor.UserID == uuid.Nil {
		return 0, nil
	}
	var count int64
	err := s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM notifications n
		WHERE n.tenant_id = ? AND (n.user_id = ? OR n.user_id IS NULL)
		  AND CASE WHEN n.user_id = ? THEN n.is_read = false
		           ELSE NOT EXISTS (
		               SELECT 1 FROM notification_reads r
		               WHERE r.notification_id = n.id AND r.user_id = ?
		           ) END`,
		actor.TenantID, actor.UserID, actor.UserID, actor.UserID).Scan(&count).Error
	if err != nil {
		return 0, apperror.Internal(err)
	}
	return count, nil
}

// MarkRead marks a single notification read for the current user.
func (s *Service) MarkRead(ctx context.Context, actor Actor, id uuid.UUID) error {
	if actor.TenantID == uuid.Nil || actor.UserID == uuid.Nil {
		return apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	var row struct {
		ID     uuid.UUID  `gorm:"column:id"`
		UserID *uuid.UUID `gorm:"column:user_id"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, user_id FROM notifications
		WHERE id = ? AND tenant_id = ? AND (user_id = ? OR user_id IS NULL)`,
		id, actor.TenantID, actor.UserID).Scan(&row).Error; err != nil {
		return apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return apperror.NotFound("Notifikasi tidak ditemukan.")
	}
	return database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if row.UserID != nil {
			return tx.Exec(`
				UPDATE notifications SET is_read = true, read_at = now()
				WHERE id = ? AND tenant_id = ? AND user_id = ?`,
				id, actor.TenantID, actor.UserID).Error
		}
		return tx.Exec(`
			INSERT INTO notification_reads (notification_id, user_id)
			VALUES (?, ?) ON CONFLICT (notification_id, user_id) DO NOTHING`,
			id, actor.UserID).Error
	})
}

// MarkAllRead marks every visible notification read for the current user.
func (s *Service) MarkAllRead(ctx context.Context, actor Actor) error {
	if actor.TenantID == uuid.Nil || actor.UserID == uuid.Nil {
		return apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE notifications SET is_read = true, read_at = now()
			WHERE tenant_id = ? AND user_id = ? AND is_read = false`,
			actor.TenantID, actor.UserID).Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO notification_reads (notification_id, user_id)
			SELECT id, ? FROM notifications
			WHERE tenant_id = ? AND user_id IS NULL
			ON CONFLICT (notification_id, user_id) DO NOTHING`,
			actor.UserID, actor.TenantID).Error
	})
}

type outboxRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	ServiceOrderID *uuid.UUID `gorm:"column:service_order_id"`
	TicketNo       *string    `gorm:"column:ticket_no"`
	CustomerName   *string    `gorm:"column:customer_name"`
	CustomerID     *uuid.UUID `gorm:"column:customer_id"`
	Phone          string     `gorm:"column:phone"`
	TemplateKey    string     `gorm:"column:template_key"`
	Body           *string    `gorm:"column:body"`
	Status         string     `gorm:"column:status"`
	Attempts       int        `gorm:"column:attempts"`
	LastError      *string    `gorm:"column:last_error"`
	Provider       string     `gorm:"column:provider"`
	ProviderMsgID  *string    `gorm:"column:provider_message_id"`
	ScheduledAt    time.Time  `gorm:"column:scheduled_at"`
	SentAt         *time.Time `gorm:"column:sent_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

// ListOutbox returns the tenant WhatsApp outbox with optional filters.
func (s *Service) ListOutbox(ctx context.Context, actor Actor, filter OutboxFilter) ([]WhatsAppMessage, int64, error) {
	if actor.TenantID == uuid.Nil {
		return nil, 0, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	where := ` WHERE w.tenant_id = ?`
	args := []any{actor.TenantID}
	if strings.TrimSpace(filter.Status) != "" {
		where += ` AND w.status = ?`
		args = append(args, strings.TrimSpace(filter.Status))
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		where += ` AND (w.phone ILIKE ? OR so.ticket_no ILIKE ? OR c.name ILIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	var total int64
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM whatsapp_messages w
		LEFT JOIN service_orders so ON so.id = w.service_order_id
		LEFT JOIN customers c ON c.id = w.customer_id`+where, args...).Scan(&total).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}
	args = append(args, perPage, (page-1)*perPage)
	rows := make([]outboxRow, 0, perPage)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT w.id, w.service_order_id, so.ticket_no, c.name AS customer_name,
		       w.customer_id, w.phone, w.template_key, w.body, w.status, w.attempts,
		       w.last_error, w.provider, w.provider_message_id, w.scheduled_at,
		       w.sent_at, w.created_at
		FROM whatsapp_messages w
		LEFT JOIN service_orders so ON so.id = w.service_order_id
		LEFT JOIN customers c ON c.id = w.customer_id`+where+`
		ORDER BY w.created_at DESC
		LIMIT ? OFFSET ?`, args...).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Internal(err)
	}
	items := make([]WhatsAppMessage, 0, len(rows))
	for _, r := range rows {
		item := WhatsAppMessage{
			ID: r.ID.String(), Phone: r.Phone, TemplateKey: r.TemplateKey,
			Body: r.Body, Status: r.Status, Attempts: r.Attempts,
			LastError: r.LastError, Provider: r.Provider,
			ProviderMsgID: r.ProviderMsgID, ScheduledAt: r.ScheduledAt,
			SentAt: r.SentAt, CreatedAt: r.CreatedAt,
		}
		if r.ServiceOrderID != nil {
			id := r.ServiceOrderID.String()
			item.ServiceOrderID = &id
		}
		if r.TicketNo != nil {
			item.TicketNo = r.TicketNo
		}
		if r.CustomerID != nil {
			id := r.CustomerID.String()
			item.CustomerID = &id
		}
		item.CustomerName = r.CustomerName
		items = append(items, item)
	}
	return items, total, nil
}

// RetryOutbox re-queues a failed message. Sent messages cannot be retried.
func (s *Service) RetryOutbox(ctx context.Context, actor Actor, id uuid.UUID) (WhatsAppMessage, error) {
	if actor.TenantID == uuid.Nil {
		return WhatsAppMessage{}, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	var row outboxRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT w.id, w.service_order_id, so.ticket_no, c.name AS customer_name,
		       w.customer_id, w.phone, w.template_key, w.body, w.status, w.attempts,
		       w.last_error, w.provider, w.provider_message_id, w.scheduled_at,
		       w.sent_at, w.created_at
		FROM whatsapp_messages w
		LEFT JOIN service_orders so ON so.id = w.service_order_id
		LEFT JOIN customers c ON c.id = w.customer_id
		WHERE w.id = ? AND w.tenant_id = ?`, id, actor.TenantID).Scan(&row).Error; err != nil {
		return WhatsAppMessage{}, apperror.Internal(err)
	}
	if row.ID == uuid.Nil {
		return WhatsAppMessage{}, apperror.NotFound("Pesan WhatsApp tidak ditemukan.")
	}
	if row.Status == WAStatusSent {
		return WhatsAppMessage{}, apperror.Conflict("MESSAGE_SENT", "Pesan sudah terkirim dan tidak dapat dikirim ulang.")
	}
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE whatsapp_messages
		SET status = 'queued', attempts = 0, last_error = NULL,
		    scheduled_at = now(), provider_message_id = NULL
		WHERE id = ? AND tenant_id = ?`, id, actor.TenantID).Error; err != nil {
		return WhatsAppMessage{}, apperror.Internal(err)
	}
	updated, _, err := s.ListOutbox(ctx, actor, OutboxFilter{Page: 1, PerPage: 100})
	if err != nil {
		return WhatsAppMessage{}, err
	}
	for _, item := range updated {
		if item.ID == id.String() {
			return item, nil
		}
	}
	return WhatsAppMessage{ID: id.String(), Status: WAStatusQueued}, nil
}

// ProcessOutbox sends a batch of queued messages. Safe to run repeatedly: rows
// are claimed with SKIP LOCKED and sent rows are never selected again.
func (s *Service) ProcessOutbox(ctx context.Context, limit int) (int, int, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	type claimedRow struct {
		ID          uuid.UUID `gorm:"column:id"`
		Phone       string    `gorm:"column:phone"`
		Body        *string   `gorm:"column:body"`
		TemplateKey string    `gorm:"column:template_key"`
		Provider    string    `gorm:"column:provider"`
		Attempts    int       `gorm:"column:attempts"`
	}
	rows := make([]claimedRow, 0, limit)
	if err := s.db.WithContext(ctx).Raw(`
		WITH claimed AS (
			SELECT id FROM whatsapp_messages
			WHERE status = 'queued' AND scheduled_at <= now()
			ORDER BY scheduled_at, created_at
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		UPDATE whatsapp_messages w
		SET status = 'processing', attempts = w.attempts + 1
		FROM claimed c
		WHERE w.id = c.id
		RETURNING w.id, w.phone, w.body, w.template_key, w.provider, w.attempts`,
		limit).Scan(&rows).Error; err != nil {
		return 0, 0, err
	}
	sent, failed := 0, 0
	for _, row := range rows {
		body := ""
		if row.Body != nil {
			body = strings.TrimSpace(*row.Body)
		}
		if body == "" {
			s.markFailed(ctx, row.ID, row.Attempts, errors.New("isi pesan kosong"))
			failed++
			continue
		}
		result, err := s.send(ctx, row.Provider, row.Phone, body)
		if err != nil {
			s.markFailed(ctx, row.ID, row.Attempts, err)
			failed++
			continue
		}
		if err := s.db.WithContext(ctx).Exec(`
			UPDATE whatsapp_messages
			SET status = 'sent', sent_at = now(), last_error = NULL,
			    provider_message_id = NULLIF(?, '')
			WHERE id = ?`, result.MessageID, row.ID).Error; err != nil {
			return sent, failed, err
		}
		sent++
	}
	return sent, failed, nil
}

func (s *Service) markFailed(ctx context.Context, id uuid.UUID, attempts int, cause error) {
	status := WAStatusFailed
	scheduled := time.Now().UTC()
	if attempts < 3 {
		status = WAStatusQueued
		scheduled = scheduled.Add(time.Duration(attempts) * 2 * time.Minute)
	}
	_ = s.db.WithContext(ctx).Exec(`
		UPDATE whatsapp_messages
		SET status = ?, last_error = ?, scheduled_at = ?
		WHERE id = ?`, status, truncateError(cause), scheduled, id).Error
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	if len(text) > 500 {
		text = text[:500]
	}
	return text
}
