package integration_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/jobs"
	"fixphone/backend/internal/modules/notification"

	"github.com/google/uuid"
)

type publicTrackingData struct {
	TicketNo    string `json:"ticket_no"`
	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Device      string `json:"device"`
	Complaint   string `json:"complaint"`
	Diagnosis   *string `json:"diagnosis"`
	Store       struct {
		Name string `json:"name"`
	} `json:"store"`
	InvoiceStatus      *string `json:"invoice_status"`
	PaymentStatusLabel string  `json:"payment_status_label"`
	PickupStatus       string  `json:"pickup_status"`
	Warranty           struct {
		Active  bool    `json:"active"`
		Status  string  `json:"status"`
		Days    int     `json:"days"`
		EndDate *string `json:"end_date"`
	} `json:"warranty"`
}

type publicLinkData struct {
	ServiceOrderID string `json:"service_order_id"`
	TicketNo       string `json:"ticket_no"`
	PublicToken    string `json:"public_token"`
	TrackingURL    string `json:"tracking_url"`
	Regenerated    bool   `json:"regenerated"`
}

type warrantyData struct {
	ID         string  `json:"id"`
	WarrantyNo string  `json:"warranty_no"`
	Type       string  `json:"type"`
	Status     string  `json:"status"`
	StartDate  string  `json:"start_date"`
	EndDate    string  `json:"end_date"`
	Days       int     `json:"days"`
	Terms      *string `json:"terms"`
	ClaimCount int64   `json:"claim_count"`
	Customer   struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"customer"`
	ServiceOrder struct {
		ID       string `json:"id"`
		TicketNo string `json:"ticket_no"`
	} `json:"service_order"`
	Claims []claimData `json:"claims"`
}

type claimData struct {
	ID         string `json:"id"`
	ClaimNo    string `json:"claim_no"`
	WarrantyID string `json:"warranty_id"`
	Status     string `json:"status"`
	Complaint  string `json:"complaint"`
	Resolution *string `json:"resolution"`
	ReworkServiceOrder *struct {
		ID       string `json:"id"`
		TicketNo string `json:"ticket_no"`
		Status   string `json:"status"`
	} `json:"rework_service_order"`
}

type notificationData struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	IsRead  bool   `json:"is_read"`
	IsBroadcast bool `json:"is_broadcast"`
}

type outboxData struct {
	ID          string  `json:"id"`
	Phone       string  `json:"phone"`
	TemplateKey string  `json:"template_key"`
	Status      string  `json:"status"`
	Attempts    int     `json:"attempts"`
	LastError   *string `json:"last_error"`
}

func TestPublicTrackingPrivacyAndTokenExpiry(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer": map[string]any{
			"phone": "081234567890", "name": "Rahasia Pelanggan",
			"address": "Jl. Rahasia No. 99",
		},
		"device": map[string]any{
			"brand": "Samsung", "model": "Galaxy S22", "imei": "356789012345678",
		},
		"complaint":      "Layar tidak menyala",
		"internal_notes": "Catatan internal rahasia",
		"credential":     map[string]any{"type": "pin", "value": "9876"},
		"estimated_cost": 500000,
	})
	orderID := created["id"].(string)
	ticketNo := created["ticket_no"].(string)

	linkResult := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID+"/public-link", nil, ownerToken)
	mustStatus(t, linkResult, http.StatusOK, "public link")
	link := decodeData[publicLinkData](t, linkResult)
	if len(link.PublicToken) != 32 || link.TicketNo != ticketNo {
		t.Fatalf("public link = %+v", link)
	}
	if !strings.Contains(link.TrackingURL, "/track/"+link.PublicToken) {
		t.Fatalf("tracking url = %q", link.TrackingURL)
	}

	tracked := app.request(t, http.MethodGet,
		"/api/v1/public/track?ticket="+ticketNo+"&phone_last4=7890", nil, "")
	mustStatus(t, tracked, http.StatusOK, "public track")
	tracking := decodeData[publicTrackingData](t, tracked)
	if tracking.TicketNo != ticketNo || tracking.Device != "Samsung Galaxy S22" {
		t.Fatalf("tracking = %+v", tracking)
	}
	if tracking.Store.Name == "" || tracking.StatusLabel == "" || tracking.PickupStatus != "Belum Diambil" {
		t.Fatalf("tracking display fields = %+v", tracking)
	}

	raw := tracked.response.Body.String()
	for _, forbidden := range []string{
		"Rahasia Pelanggan", "081234567890", "Jl. Rahasia", "356789012345678",
		"Catatan internal rahasia", "9876", "cost_price", "technician",
		"credential", "internal_notes", "customer_name", "phone",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("public tracking leaked %q: %s", forbidden, raw)
		}
	}

	byToken := app.request(t, http.MethodGet, "/api/v1/public/track/"+link.PublicToken, nil, "")
	mustStatus(t, byToken, http.StatusOK, "public track by token")
	if decoded := decodeData[publicTrackingData](t, byToken); decoded.TicketNo != ticketNo {
		t.Fatalf("token tracking ticket = %q", decoded.TicketNo)
	}

	// Wrong digits return a generic 404 without confirming the ticket exists.
	wrong := app.request(t, http.MethodGet,
		"/api/v1/public/track?ticket="+ticketNo+"&phone_last4=0000", nil, "")
	assertCode(t, wrong, http.StatusNotFound, "NOT_FOUND")
	if strings.Contains(wrong.response.Body.String(), "ticket") && !strings.Contains(wrong.response.Body.String(), "tidak ditemukan") {
		t.Fatalf("wrong digits response not generic: %s", wrong.response.Body.String())
	}

	if err := app.db.Exec(`
		UPDATE service_orders SET public_token_expires_at = now() - interval '1 minute'
		WHERE id = ?`, orderID).Error; err != nil {
		t.Fatalf("expire token: %v", err)
	}
	expired := app.request(t, http.MethodGet, "/api/v1/public/track/"+link.PublicToken, nil, "")
	assertCode(t, expired, http.StatusNotFound, "NOT_FOUND")

	// A staff request regenerates an expired token.
	regenerated := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID+"/public-link", nil, ownerToken)
	mustStatus(t, regenerated, http.StatusOK, "regenerate public link")
	fresh := decodeData[publicLinkData](t, regenerated)
	if !fresh.Regenerated || fresh.PublicToken == link.PublicToken {
		t.Fatalf("token was not regenerated: %+v", fresh)
	}
}

func TestPublicTrackingRateLimit(t *testing.T) {
	app := setupAuthApp(t)

	for i := 1; i <= 10; i++ {
		result := app.request(t, http.MethodGet,
			"/api/v1/public/track?ticket=TIDAK-ADA&phone_last4=1234", nil, "")
		if result.response.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was rate limited too early", i)
		}
	}
	blocked := app.request(t, http.MethodGet,
		"/api/v1/public/track?ticket=TIDAK-ADA&phone_last4=1234", nil, "")
	if blocked.response.Code != http.StatusTooManyRequests {
		t.Fatalf("11th request = %d, want 429: %s", blocked.response.Code, blocked.response.Body.String())
	}
	if blocked.response.Header().Get("Retry-After") == "" {
		t.Fatal("rate limited response missing Retry-After")
	}
}

func TestWarrantyClaimReworkAndNotifications(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081255556666", "name": "Wati Garansi"},
		"device":    map[string]any{"brand": "Oppo", "model": "Reno 10"},
		"complaint": "Baterai cepat habis",
	})
	orderID := created["id"].(string)

	item := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type": "service", "description": "Ganti baterai", "qty": 1,
		"unit_price": 0, "warranty_days": 45,
	}, ownerToken)
	mustStatus(t, item, http.StatusCreated, "add warranty item")

	for _, status := range []string{"waiting_diagnosis", "in_progress", "testing", "completed", "ready_for_pickup", "picked_up"} {
		mustStatus(t, changeStatus(t, app, ownerToken, orderID, status, nil), http.StatusOK, "transition "+status)
	}

	list := app.request(t, http.MethodGet, "/api/v1/warranties", nil, ownerToken)
	mustStatus(t, list, http.StatusOK, "list warranties")
	warranties := decodeData[[]warrantyData](t, list)
	if len(warranties) != 1 {
		t.Fatalf("warranties = %+v", warranties)
	}
	warranty := warranties[0]
	if warranty.Type != "service" || warranty.Days != 45 || warranty.Status != "active" {
		t.Fatalf("warranty = %+v", warranty)
	}
	start, err := time.Parse("2006-01-02", warranty.StartDate)
	if err != nil {
		t.Fatalf("parse start: %v", err)
	}
	end, err := time.Parse("2006-01-02", warranty.EndDate)
	if err != nil {
		t.Fatalf("parse end: %v", err)
	}
	if int(end.Sub(start).Hours()/24) != 45 {
		t.Fatalf("warranty range %s..%s is not 45 days", warranty.StartDate, warranty.EndDate)
	}

	var warrantyCount int64
	if err := app.db.Raw(`SELECT count(*) FROM warranties WHERE service_order_id = ?`, orderID).Scan(&warrantyCount).Error; err != nil {
		t.Fatalf("count warranties: %v", err)
	}
	if warrantyCount != 1 {
		t.Fatalf("warranty rows = %d, want 1", warrantyCount)
	}

	claimResult := app.request(t, http.MethodPost, "/api/v1/warranties/"+warranty.ID+"/claims",
		map[string]any{"complaint": "Baterai kembali drop setelah 2 hari"}, ownerToken)
	mustStatus(t, claimResult, http.StatusCreated, "create claim")
	claim := decodeData[claimData](t, claimResult)
	if claim.Status != "submitted" || !strings.HasPrefix(claim.ClaimNo, "CLM-") {
		t.Fatalf("claim = %+v", claim)
	}

	// Claim acknowledgement creates both an inbox entry and a WhatsApp outbox row.
	notifications := app.request(t, http.MethodGet, "/api/v1/notifications", nil, ownerToken)
	mustStatus(t, notifications, http.StatusOK, "list notifications")
	inbox := decodeData[[]notificationData](t, notifications)
	foundClaim := false
	for _, item := range inbox {
		if item.Type == "warranty_claim" {
			foundClaim = true
		}
	}
	if !foundClaim {
		t.Fatalf("claim notification missing: %+v", inbox)
	}

	outbox := app.request(t, http.MethodGet, "/api/v1/whatsapp-messages?status=queued", nil, ownerToken)
	mustStatus(t, outbox, http.StatusOK, "list outbox")
	messages := decodeData[[]outboxData](t, outbox)
	if len(messages) == 0 || messages[0].TemplateKey != "claim_received" || messages[0].Status != "queued" {
		t.Fatalf("outbox = %+v", messages)
	}

	// Unread count and mark-all-read work for the current user.
	unread := app.request(t, http.MethodGet, "/api/v1/notifications/unread-count", nil, ownerToken)
	mustStatus(t, unread, http.StatusOK, "unread count")
	var unreadData struct {
		Count int64 `json:"count"`
	}
	if err := json.Unmarshal(unread.body.Data, &unreadData); err != nil {
		t.Fatalf("decode unread: %v", err)
	}
	if unreadData.Count == 0 {
		t.Fatal("expected unread notifications")
	}
	readAll := app.request(t, http.MethodPost, "/api/v1/notifications/read-all", nil, ownerToken)
	mustStatus(t, readAll, http.StatusOK, "mark all read")
	unreadAgain := app.request(t, http.MethodGet, "/api/v1/notifications/unread-count", nil, ownerToken)
	mustStatus(t, unreadAgain, http.StatusOK, "unread count after read-all")
	if err := json.Unmarshal(unreadAgain.body.Data, &unreadData); err != nil {
		t.Fatalf("decode unread after: %v", err)
	}
	if unreadData.Count != 0 {
		t.Fatalf("unread after read-all = %d", unreadData.Count)
	}

	rework := app.request(t, http.MethodPost, "/api/v1/claims/"+claim.ID+"/rework", nil, ownerToken)
	mustStatus(t, rework, http.StatusCreated, "create rework")
	reworkData := decodeData[struct {
		ServiceOrderID string `json:"service_order_id"`
		TicketNo       string `json:"ticket_no"`
		ClaimStatus    string `json:"claim_status"`
		InvoiceNo      string `json:"invoice_no"`
	}](t, rework)
	if reworkData.ServiceOrderID == "" || reworkData.ClaimStatus != "in_progress" || reworkData.InvoiceNo == "" {
		t.Fatalf("rework = %+v", reworkData)
	}

	reworkDetail := app.request(t, http.MethodGet, "/api/v1/service-orders/"+reworkData.ServiceOrderID, nil, ownerToken)
	mustStatus(t, reworkDetail, http.StatusOK, "rework detail")
	detail := decodeData[serviceOrderDetail](t, reworkDetail)
	if detail.Customer.Name != "Wati Garansi" || detail.Device.Model != "Reno 10" || detail.Status != "received" {
		t.Fatalf("rework detail = %+v", detail)
	}
	var reworkLink struct {
		OrderType    string     `gorm:"column:order_type"`
		ReworkOf     *uuid.UUID `gorm:"column:rework_of_service_order_id"`
		WarrantyClaim *uuid.UUID `gorm:"column:warranty_claim_id"`
	}
	if err := app.db.Raw(`
		SELECT order_type, rework_of_service_order_id, warranty_claim_id
		FROM service_orders WHERE id = ?`, reworkData.ServiceOrderID).Scan(&reworkLink).Error; err != nil {
		t.Fatalf("read rework link: %v", err)
	}
	if reworkLink.OrderType != "rework" || reworkLink.ReworkOf == nil || reworkLink.WarrantyClaim == nil {
		t.Fatalf("rework linkage = %+v", reworkLink)
	}

	second := app.request(t, http.MethodPost, "/api/v1/claims/"+claim.ID+"/rework", nil, ownerToken)
	assertCode(t, second, http.StatusConflict, "REWORK_EXISTS")

	// Cross-tenant warranty access is a generic 404.
	other, err := database.CreateTenant(t.Context(), app.db, database.CreateTenantInput{
		Name: "Toko Lain", Slug: "toko-lain-garansi", OwnerName: "Siti Lain",
		OwnerEmail: "owner-garansi@fixphone.local", OwnerPassword: "Owner#12345",
		OutletName: "Cabang", OutletCode: "CBG",
	})
	if err != nil || other.TenantID == uuid.Nil {
		t.Fatalf("create second tenant: %v", err)
	}
	otherLogin := app.login(t, "owner-garansi@fixphone.local", "Owner#12345")
	mustStatus(t, otherLogin, http.StatusOK, "login other tenant")
	otherToken := accessToken(t, otherLogin)
	crossTenant := app.request(t, http.MethodGet, "/api/v1/warranties/"+warranty.ID, nil, otherToken)
	assertCode(t, crossTenant, http.StatusNotFound, "NOT_FOUND")
}

func TestWorkerRemindersOutboxAndCredentialPurge(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081277778888", "name": "Budi Worker"},
		"device":    map[string]any{"brand": "Asus", "model": "Zenfone 10"},
		"complaint": "Cek konektor charger",
		"credential": map[string]any{"type": "pin", "value": "4321"},
	})
	orderID := created["id"].(string)
	assertCode(t, changeStatus(t, app, ownerToken, orderID, "waiting_diagnosis", nil), http.StatusOK, "")
	assertCode(t, changeStatus(t, app, ownerToken, orderID, "in_progress", nil), http.StatusOK, "")

	if err := app.db.Exec(`
		UPDATE service_orders SET estimated_done_at = now() - interval '2 hours',
		       credential_purge_at = now() - interval '1 day'
		WHERE id = ?`, orderID).Error; err != nil {
		t.Fatalf("prepare worker data: %v", err)
	}

	notifier := notification.NewService(app.db, notification.ServiceDeps{AppBaseURL: "http://localhost:5173"})
	runner := jobs.NewRunner(app.db, notifier, "http://localhost:5173")

	first, err := runner.RunReminders(t.Context())
	if err != nil {
		t.Fatalf("run reminders: %v", err)
	}
	if first.OverdueServices != 1 {
		t.Fatalf("overdue reminders = %+v", first)
	}
	second, err := runner.RunReminders(t.Context())
	if err != nil {
		t.Fatalf("run reminders again: %v", err)
	}
	if second.OverdueServices != 1 {
		t.Fatalf("second reminders = %+v", second)
	}

	var overdueNotifications int64
	if err := app.db.Raw(`
		SELECT count(*) FROM notifications
		WHERE type = 'service_overdue' AND entity_id = ?`, orderID).Scan(&overdueNotifications).Error; err != nil {
		t.Fatalf("count overdue notifications: %v", err)
	}
	if overdueNotifications != 1 {
		t.Fatalf("overdue notifications = %d, want 1", overdueNotifications)
	}
	var reminderMessages int64
	if err := app.db.Raw(`
		SELECT count(*) FROM whatsapp_messages
		WHERE service_order_id = ? AND template_key = 'reminder'`, orderID).Scan(&reminderMessages).Error; err != nil {
		t.Fatalf("count reminder messages: %v", err)
	}
	if reminderMessages != 1 {
		t.Fatalf("reminder messages = %d, want 1", reminderMessages)
	}

	sent, failed, err := runner.ProcessOutbox(t.Context())
	if err != nil || sent == 0 || failed != 0 {
		t.Fatalf("process outbox: sent=%d failed=%d err=%v", sent, failed, err)
	}
	sentAgain, failedAgain, err := runner.ProcessOutbox(t.Context())
	if err != nil || sentAgain != 0 || failedAgain != 0 {
		t.Fatalf("second outbox pass: sent=%d failed=%d err=%v", sentAgain, failedAgain, err)
	}
	var messageStatus string
	if err := app.db.Raw(`
		SELECT status FROM whatsapp_messages
		WHERE service_order_id = ? AND template_key = 'reminder'`, orderID).Scan(&messageStatus).Error; err != nil {
		t.Fatalf("read message status: %v", err)
	}
	if messageStatus != "sent" {
		t.Fatalf("message status = %q, want sent", messageStatus)
	}

	purged, err := runner.PurgeCredentials(t.Context())
	if err != nil || purged != 1 {
		t.Fatalf("purge credentials: count=%d err=%v", purged, err)
	}
	var credential struct {
		Type string  `gorm:"column:credential_type"`
		Enc  *[]byte `gorm:"column:credential_enc"`
		Set  *time.Time `gorm:"column:credential_set_at"`
	}
	if err := app.db.Raw(`
		SELECT credential_type, credential_enc, credential_set_at
		FROM service_orders WHERE id = ?`, orderID).Scan(&credential).Error; err != nil {
		t.Fatalf("read credential: %v", err)
	}
	if credential.Type != "none" || credential.Enc != nil || credential.Set != nil {
		t.Fatalf("credential not purged: %+v", credential)
	}
	purgedAgain, err := runner.PurgeCredentials(t.Context())
	if err != nil || purgedAgain != 0 {
		t.Fatalf("second purge: count=%d err=%v", purgedAgain, err)
	}
}
