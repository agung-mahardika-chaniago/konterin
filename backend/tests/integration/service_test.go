package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"fixphone/backend/internal/database"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Helpers shared by the service test suite.
// ---------------------------------------------------------------------------

// multipartRequest posts a multipart/form-data request (photo upload).
func (a *authTestApp) multipartRequest(t *testing.T, method, path, token string,
	fields map[string]string, fileField, fileName string, fileContent []byte) *apiResult {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("write multipart field %s: %v", name, err)
		}
	}
	if fileContent != nil {
		part, err := writer.CreateFormFile(fileField, fileName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.RemoteAddr = "127.0.0.1:12345"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.engine.ServeHTTP(rec, req)
	result := &apiResult{response: rec, cookies: rec.Result().Cookies()}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result.body); err != nil {
			t.Fatalf("decode %s %s response (%d): %v\n%s", method, path, rec.Code, err, rec.Body.String())
		}
	}
	return result
}

// createServiceOrder posts a valid intake and fails the test on any error.
func createServiceOrder(t *testing.T, app *authTestApp, token string, body map[string]any) map[string]any {
	t.Helper()
	result := app.request(t, http.MethodPost, "/api/v1/service-orders", body, token)
	if result.response.Code != http.StatusCreated {
		t.Fatalf("create service order = %d: %s", result.response.Code, result.response.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(result.body.Data, &data); err != nil {
		t.Fatalf("decode created order: %v", err)
	}
	return data
}

// ticketNoPattern matches the default FP-{YYMMDD}-{SEQ:4} document format.
var ticketNoPattern = regexp.MustCompile(`^FP-\d{6}-\d{4}$`)

type serviceOrderDetail struct {
	ID                   string  `json:"id"`
	TicketNo             string  `json:"ticket_no"`
	Status               string  `json:"status"`
	Priority             string  `json:"priority"`
	EstimatedCost        int64   `json:"estimated_cost"`
	Complaint            string  `json:"complaint"`
	CredentialConfigured bool    `json:"credential_configured"`
	PhotoCount           int64   `json:"photo_count"`
	PublicToken          *string `json:"public_token"`
	TrackingURL          *string `json:"tracking_url"`
	Customer             struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Device struct {
		ID    string  `json:"id"`
		Brand string  `json:"brand"`
		Model string  `json:"model"`
		IMEI  *string `json:"imei"`
	} `json:"device"`
	Timeline []struct {
		ToStatus   string  `json:"to_status"`
		FromStatus *string `json:"from_status"`
		ChangedBy  *string `json:"changed_by"`
	} `json:"timeline"`
	Invoice *struct {
		ID          string `json:"id"`
		InvoiceNo   string `json:"invoice_no"`
		Status      string `json:"status"`
		TotalAmount int64  `json:"total_amount"`
		PaidAmount  int64  `json:"paid_amount"`
	} `json:"invoice"`
	Store struct {
		Name string `json:"name"`
	} `json:"store"`
	Photos []struct {
		ID        string `json:"id"`
		Stage     string `json:"stage"`
		MimeType  string `json:"mime_type"`
		FilePath  string `json:"file_path"`
		IsPublic  bool   `json:"is_public"`
		SizeBytes int64  `json:"size_bytes"`
	} `json:"photos"`
}

type serviceOrderItem struct {
	ID                   string `json:"id"`
	TicketNo             string `json:"ticket_no"`
	Status               string `json:"status"`
	Priority             string `json:"priority"`
	CredentialConfigured bool   `json:"credential_configured"`
	Customer             struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Device struct {
		Brand string `json:"brand"`
		Model string `json:"model"`
	} `json:"device"`
}

type listMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// assertCode checks both the HTTP status and the error envelope code.
func assertCode(t *testing.T, result *apiResult, status int, wantCode string) {
	t.Helper()
	if result.response.Code != status {
		t.Fatalf("status = %d, want %d: %s", result.response.Code, status, result.response.Body.String())
	}
	if wantCode != "" && result.body.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q (status %d)", result.body.Error.Code, wantCode, status)
	}
}

// ---------------------------------------------------------------------------
// 1. Customer CRUD, duplicate prevention, search, devices.
// ---------------------------------------------------------------------------

func TestServiceCustomersLifecycleAndDuplicates(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := app.request(t, http.MethodPost, "/api/v1/customers", map[string]any{
		"name": "Rina Wulandari", "phone": "081311112222", "address": "Jl. Melati 12",
	}, ownerToken)
	assertCode(t, created, http.StatusCreated, "")
	var customer map[string]any
	if err := json.Unmarshal(created.body.Data, &customer); err != nil {
		t.Fatalf("decode customer: %v", err)
	}
	customerID := customer["id"].(string)

	// Same phone digits, different formatting → still a duplicate.
	duplicate := app.request(t, http.MethodPost, "/api/v1/customers", map[string]any{
		"name": "Rina Kembar", "phone": "+62 813-1111-2222",
	}, ownerToken)
	assertCode(t, duplicate, http.StatusConflict, "DUPLICATE_PHONE")
	if !strings.Contains(duplicate.body.Error.Message, "Nomor WhatsApp") {
		t.Fatalf("duplicate message unexpected: %s", duplicate.body.Error.Message)
	}

	// Normalized phone is stored in 62-prefixed form.
	var normalized string
	if err := app.db.Raw(`SELECT phone_normalized FROM customers WHERE id = ?`, customerID).
		Scan(&normalized).Error; err != nil {
		t.Fatalf("read phone_normalized: %v", err)
	}
	if normalized != "6281311112222" {
		t.Fatalf("phone_normalized = %q, want 6281311112222", normalized)
	}

	// Patch rename.
	updated := app.request(t, http.MethodPatch, "/api/v1/customers/"+customerID, map[string]any{
		"name": "Rina W. Santoso", "notes": "pelanggan setia",
	}, ownerToken)
	if updated.response.Code != http.StatusOK {
		t.Fatalf("update customer = %d: %s", updated.response.Code, updated.response.Body.String())
	}

	// Device + duplicate IMEI guard.
	device := app.request(t, http.MethodPost, "/api/v1/customers/"+customerID+"/devices", map[string]any{
		"brand": "Xiaomi", "model": "Redmi Note 11", "imei": "356938035643809",
	}, ownerToken)
	if device.response.Code != http.StatusCreated {
		t.Fatalf("create device = %d: %s", device.response.Code, device.response.Body.String())
	}
	deviceAgain := app.request(t, http.MethodPost, "/api/v1/customers/"+customerID+"/devices", map[string]any{
		"brand": "Xiaomi", "model": "Redmi Note 11", "imei": "356938035643809",
	}, ownerToken)
	assertCode(t, deviceAgain, http.StatusConflict, "DEVICE_IMEI_EXISTS")

	detail := app.request(t, http.MethodGet, "/api/v1/customers/"+customerID, nil, ownerToken)
	if detail.response.Code != http.StatusOK {
		t.Fatalf("customer detail = %d: %s", detail.response.Code, detail.response.Body.String())
	}
	var detailData struct {
		Name    string           `json:"name"`
		Devices []map[string]any `json:"devices"`
		Stats   struct {
			TotalServices int64 `json:"total_services"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(detail.body.Data, &detailData); err != nil {
		t.Fatalf("decode customer detail: %v", err)
	}
	if len(detailData.Devices) != 1 || detailData.Stats.TotalServices != 0 {
		t.Fatalf("unexpected customer detail: %+v", detailData)
	}

	// Search matches name and normalized phone fragment.
	search := app.request(t, http.MethodGet, "/api/v1/customers?q=wulandari", nil, ownerToken)
	var searchItems []map[string]any
	if err := json.Unmarshal(search.body.Data, &searchItems); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if len(searchItems) != 1 || searchItems[0]["id"] != customerID {
		t.Fatalf("unexpected search result: %+v", searchItems)
	}
	searchPhone := app.request(t, http.MethodGet, "/api/v1/customers?q=6281311112222", nil, ownerToken)
	var phoneItems []map[string]any
	if err := json.Unmarshal(searchPhone.body.Data, &phoneItems); err != nil {
		t.Fatalf("decode search phone: %v", err)
	}
	if len(phoneItems) != 1 {
		t.Fatalf("phone search found %d customers, want 1", len(phoneItems))
	}

	// Soft delete frees the number; re-creating the same phone succeeds.
	deleted := app.request(t, http.MethodDelete, "/api/v1/customers/"+customerID, nil, ownerToken)
	if deleted.response.Code != http.StatusOK && deleted.response.Code != http.StatusNoContent {
		t.Fatalf("delete customer = %d: %s", deleted.response.Code, deleted.response.Body.String())
	}
	recreated := app.request(t, http.MethodPost, "/api/v1/customers", map[string]any{
		"name": "Rina Baru", "phone": "081311112222",
	}, ownerToken)
	if recreated.response.Code != http.StatusCreated {
		t.Fatalf("recreate deleted phone = %d: %s", recreated.response.Code, recreated.response.Body.String())
	}

	// Invalid phone format is rejected with a clear message.
	invalid := app.request(t, http.MethodPost, "/api/v1/customers", map[string]any{
		"name": "Nomor Aneh", "phone": "abcd1234",
	}, ownerToken)
	assertCode(t, invalid, http.StatusUnprocessableEntity, "")
}

// ---------------------------------------------------------------------------
// 2. Intake flow: ticket number, invoice draft, timeline, credential, audit.
// ---------------------------------------------------------------------------

func TestServiceOrderIntakeCredentialAndAudit(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)
	app.createUser(t, ownerToken, "Admin Servis", "service-admin@fixphone.local", "admin", []string{app.outletID.String()})
	app.allowLogin(t, "service-admin@fixphone.local")
	adminToken := accessToken(t, app.login(t, "service-admin@fixphone.local", "Password#123"))

	// Customer is created once; intake must reuse it by normalized phone.
	created := app.request(t, http.MethodPost, "/api/v1/customers", map[string]any{
		"name": "Dewi Lestari", "phone": "081233334444",
	}, ownerToken)
	assertCode(t, created, http.StatusCreated, "")
	var customer map[string]any
	if err := json.Unmarshal(created.body.Data, &customer); err != nil {
		t.Fatalf("decode customer: %v", err)
	}

	createdOrder := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":           map[string]any{"phone": "081233334444", "name": "Dewi Lestari"},
		"device":             map[string]any{"brand": "Samsung", "model": "A34", "color": "Hitam"},
		"complaint":          "Layar berkedip setelah jatuh",
		"completeness":       "Unit + charger",
		"physical_condition": "Scratch belakang, layar retak",
		"credential":         map[string]any{"type": "pin", "value": "987654321"},
		"estimated_cost":     250000,
		"priority":           "normal",
		"notes":              "Simpan di rak 3",
		"internal_notes":     "Pelanggan menunggu dengan sabar",
	})
	orderID := createdOrder["id"].(string)
	ticketNo := createdOrder["ticket_no"].(string)

	// Ticket number format + public tracking token.
	if !ticketNoPattern.MatchString(ticketNo) {
		t.Fatalf("ticket_no = %q does not match FP-YYMMDD-SEQ4", ticketNo)
	}
	publicToken := createdOrder["public_token"].(string)
	if len(publicToken) != 32 {
		t.Fatalf("public_token length = %d, want 32", len(publicToken))
	}
	if !strings.Contains(createdOrder["tracking_url"].(string), "/track/"+publicToken) {
		t.Fatalf("tracking_url unexpected: %v", createdOrder["tracking_url"])
	}
	if createdOrder["status"] != "received" {
		t.Fatalf("initial status = %v", createdOrder["status"])
	}

	// Customer reuse: intake did NOT create a second customer row.
	var customerCount int64
	if err := app.db.Raw(`SELECT count(*) FROM customers WHERE phone_normalized = '6281233334444'`).
		Scan(&customerCount).Error; err != nil {
		t.Fatalf("count customers: %v", err)
	}
	if customerCount != 1 {
		t.Fatalf("customer reuse failed: %d rows", customerCount)
	}

	// Draft invoice was created automatically.
	var invoiceNo, invoiceStatus string
	if err := app.db.Raw(`SELECT invoice_no, status FROM invoices WHERE service_order_id = ?`, orderID).
		Row().Scan(&invoiceNo, &invoiceStatus); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if invoiceStatus != "draft" || !strings.HasPrefix(invoiceNo, "INV-") {
		t.Fatalf("invoice = %s/%s, want draft INV-*", invoiceStatus, invoiceNo)
	}

	// Detail: device, timeline, invoice, store; credential must stay secret.
	detail := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID, nil, ownerToken)
	assertCode(t, detail, http.StatusOK, "")
	var detailData serviceOrderDetail
	if err := json.Unmarshal(detail.body.Data, &detailData); err != nil {
		t.Fatalf("decode order detail: %v", err)
	}
	if detailData.Customer.ID != customer["id"].(string) || detailData.Customer.Phone != "081233334444" {
		t.Fatalf("unexpected customer ref: %+v", detailData.Customer)
	}
	if detailData.Device.Brand != "Samsung" || detailData.Device.IMEI != nil {
		t.Fatalf("unexpected device ref: %+v", detailData.Device)
	}
	if !detailData.CredentialConfigured {
		t.Fatalf("credential_configured = false, want true")
	}
	if len(detailData.Timeline) != 1 || detailData.Timeline[0].ToStatus != "received" ||
		detailData.Timeline[0].FromStatus != nil {
		t.Fatalf("unexpected initial timeline: %+v", detailData.Timeline)
	}
	if detailData.Invoice == nil || detailData.Invoice.Status != "draft" {
		t.Fatalf("unexpected invoice ref: %+v", detailData.Invoice)
	}
	if detailData.Store.Name == "" {
		t.Fatalf("store ref missing for label printing")
	}
	if strings.Contains(detail.response.Body.String(), "987654321") {
		t.Fatalf("credential clear text leaked in detail response")
	}
	if strings.Contains(detail.response.Body.String(), "credential_enc") {
		t.Fatalf("encrypted credential blob leaked in detail response")
	}
	if detailData.PublicToken == nil || *detailData.PublicToken != publicToken {
		t.Fatalf("detail public token mismatch")
	}
	if detailData.EstimatedCost != 250000 {
		t.Fatalf("estimated cost = %d, want 250000", detailData.EstimatedCost)
	}

	// Reveal requires a reason; the admin role lacks the reveal permission.
	revealNoReason := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/credential/reveal",
		map[string]any{"reason": ""}, ownerToken)
	assertCode(t, revealNoReason, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	revealDenied := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/credential/reveal",
		map[string]any{"reason": "kelalaian admin"}, adminToken)
	assertCode(t, revealDenied, http.StatusForbidden, "FORBIDDEN")

	reveal := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/credential/reveal",
		map[string]any{"reason": "Perlu cek pola kunci pelanggan"}, ownerToken)
	assertCode(t, reveal, http.StatusOK, "")
	var revealData struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(reveal.body.Data, &revealData); err != nil {
		t.Fatalf("decode reveal: %v", err)
	}
	if revealData.Type != "pin" || revealData.Value != "987654321" {
		t.Fatalf("reveal = %s/%s", revealData.Type, revealData.Value)
	}
	if reveal.response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reveal response missing Cache-Control no-store")
	}

	// The reveal attempt must be audited.
	var auditCount int64
	if err := app.db.Raw(`SELECT count(*) FROM audit_logs WHERE action = 'service.credential_reveal' AND entity_id = ?`,
		orderID).Scan(&auditCount).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("credential reveal audit rows = %d, want 1", auditCount)
	}

	// Intake update: invalid priority rejected, valid patch applied.
	patchBad := app.request(t, http.MethodPatch, "/api/v1/service-orders/"+orderID, map[string]any{
		"priority": "kilat",
	}, ownerToken)
	assertCode(t, patchBad, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	patch := app.request(t, http.MethodPatch, "/api/v1/service-orders/"+orderID, map[string]any{
		"priority": "express", "estimated_cost": 350000,
	}, ownerToken)
	if patch.response.Code != http.StatusOK {
		t.Fatalf("patch order = %d: %s", patch.response.Code, patch.response.Body.String())
	}

	detailAfter := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID, nil, ownerToken)
	if detailAfter.response.Code != http.StatusOK {
		t.Fatalf("detail after patch = %d", detailAfter.response.Code)
	}
	var after serviceOrderDetail
	if err := json.Unmarshal(detailAfter.body.Data, &after); err != nil {
		t.Fatalf("decode order detail after patch: %v", err)
	}
	if after.Priority != "express" || after.EstimatedCost != 350000 {
		t.Fatalf("patch not applied: priority=%s cost=%d", after.Priority, after.EstimatedCost)
	}

	// Credential clearing restores the "not configured" state.
	deleteCred := app.request(t, http.MethodDelete, "/api/v1/service-orders/"+orderID+"/credential", nil, ownerToken)
	if deleteCred.response.Code != http.StatusOK {
		t.Fatalf("delete credential = %d: %s", deleteCred.response.Code, deleteCred.response.Body.String())
	}
	detailCleared := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID, nil, ownerToken)
	var cleared serviceOrderDetail
	if err := json.Unmarshal(detailCleared.body.Data, &cleared); err != nil {
		t.Fatalf("decode cleared detail: %v", err)
	}
	if cleared.CredentialConfigured {
		t.Fatalf("credential_configured still true after delete")
	}
}

// ---------------------------------------------------------------------------
// 3. Concurrent intake: unique ticket numbers under 20 parallel requests.
// ---------------------------------------------------------------------------

func TestServiceTicketNumbersUniqueUnderConcurrency(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	const workers = 20
	type outcome struct {
		code int
		body string
	}
	results := make([]outcome, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result := app.request(t, http.MethodPost, "/api/v1/service-orders", map[string]any{
				"customer": map[string]any{
					"name":  fmt.Sprintf("Pelanggan Paralel %02d", i),
					"phone": fmt.Sprintf("08150011%04d", i),
				},
				"device":    map[string]any{"brand": "Vivo", "model": "Y17s"},
				"complaint": "Boot loop paralel",
			}, ownerToken)
			results[i] = outcome{code: result.response.Code, body: result.response.Body.String()}
		}(i)
	}
	wg.Wait()

	tickets := make(map[string]struct{}, workers)
	for i, result := range results {
		if result.code != http.StatusCreated {
			t.Fatalf("worker %d create = %d: %s", i, result.code, result.body)
		}
		var payload struct {
			Data struct {
				TicketNo string `json:"ticket_no"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(result.body), &payload); err != nil {
			t.Fatalf("worker %d decode: %v", i, err)
		}
		ticket := payload.Data.TicketNo
		if !ticketNoPattern.MatchString(ticket) {
			t.Fatalf("worker %d ticket format: %s", i, ticket)
		}
		if _, exists := tickets[ticket]; exists {
			t.Fatalf("duplicate ticket number under concurrency: %s", ticket)
		}
		tickets[ticket] = struct{}{}
	}
	if len(tickets) != workers {
		t.Fatalf("unique tickets = %d, want %d", len(tickets), workers)
	}

	var orderCount int64
	if err := app.db.Raw(`SELECT count(*) FROM service_orders`).Scan(&orderCount).Error; err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != workers {
		t.Fatalf("orders persisted = %d, want %d", orderCount, workers)
	}
}

// ---------------------------------------------------------------------------
// 4. List filters, pagination, outlet guard, tenant isolation.
// ---------------------------------------------------------------------------

func TestServiceOrderListFiltersAndTenantIsolation(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	tech := app.createUser(t, ownerToken, "Teknisi Filter", "tech-filter@fixphone.local", "technician", []string{app.outletID.String()})
	app.allowLogin(t, "tech-filter@fixphone.local")
	techToken := accessToken(t, app.login(t, "tech-filter@fixphone.local", "Password#123"))
	techID, err := uuid.Parse(tech["id"].(string))
	if err != nil {
		t.Fatalf("parse technician id: %v", err)
	}

	customerA := seedDashboardCustomer(t, app, "Ayu Filter", "081244445555")
	customerB := seedDashboardCustomer(t, app, "Bagas Surya", "081244446666")
	now := time.Now().UTC()
	orderA := insertServiceOrder(t, app, customerA, &techID, "FIL-001", "received", now, nil, nil, nil, nil)
	orderOther := insertServiceOrder(t, app, customerA, nil, "FIL-002", "in_progress", now, nil, nil, nil, nil)
	insertServiceOrder(t, app, customerB, nil, "FIL-003", "received", now.AddDate(0, 0, -5), nil, nil, nil, nil)
	insertServiceOrder(t, app, customerB, nil, "FIL-004", "picked_up", now, nil, nil, nil, nil)

	fetchList := func(query, token string) ([]serviceOrderItem, listMeta) {
		t.Helper()
		result := app.request(t, http.MethodGet, "/api/v1/service-orders"+query, nil, token)
		if result.response.Code != http.StatusOK {
			t.Fatalf("list %s = %d: %s", query, result.response.Code, result.response.Body.String())
		}
		var items []serviceOrderItem
		if err := json.Unmarshal(result.body.Data, &items); err != nil {
			t.Fatalf("decode list %s: %v", query, err)
		}
		var meta listMeta
		if result.body.Meta != nil {
			if err := json.Unmarshal(result.body.Meta, &meta); err != nil {
				t.Fatalf("decode meta %s: %v", query, err)
			}
		}
		return items, meta
	}

	_, allMeta := fetchList("", ownerToken)
	if allMeta.Total != 4 || allMeta.Page != 1 || allMeta.PerPage != 20 {
		t.Fatalf("unexpected all-meta: %+v", allMeta)
	}

	received, _ := fetchList("?status=received", ownerToken)
	if len(received) != 2 {
		t.Fatalf("status filter received count = %d, want 2", len(received))
	}
	progress, _ := fetchList("?status=received,in_progress", ownerToken)
	if len(progress) != 3 {
		t.Fatalf("multi-status filter count = %d, want 3", len(progress))
	}
	byTech, _ := fetchList("?technician_id="+techID.String(), ownerToken)
	if len(byTech) != 1 || byTech[0].TicketNo != "FIL-001" {
		t.Fatalf("technician filter broken: %+v", byTech)
	}
	byCustomer, _ := fetchList("?customer_id="+customerB.String(), ownerToken)
	if len(byCustomer) != 2 {
		t.Fatalf("customer filter count = %d, want 2", len(byCustomer))
	}
	byName, _ := fetchList("?q=ayu", ownerToken)
	if len(byName) != 2 {
		t.Fatalf("customer-name search count = %d, want 2", len(byName))
	}
	byBrand, _ := fetchList("?q=samsung", ownerToken)
	if len(byBrand) != 4 {
		t.Fatalf("brand search count = %d, want 4", len(byBrand))
	}
	byTicket, _ := fetchList("?q=FIL-004", ownerToken)
	if len(byTicket) != 1 || byTicket[0].TicketNo != "FIL-004" {
		t.Fatalf("ticket search broken: %+v", byTicket)
	}
	invalidFilter := app.request(t, http.MethodGet, "/api/v1/service-orders?technician_id=bukan-uuid", nil, ownerToken)
	assertCode(t, invalidFilter, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	// Date window uses the outlet timezone; received "now" is always today WIB.
	today := time.Now().In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02")
	dated, _ := fetchList("?from="+today+"&to="+today+"&status=received", ownerToken)
	if len(dated) != 1 || dated[0].TicketNo != "FIL-001" {
		t.Fatalf("date filter broken: %+v", dated)
	}
	future, _ := fetchList("?from=2030-01-01&to=2030-01-31", ownerToken)
	if len(future) != 0 {
		t.Fatalf("future window should be empty, got %d", len(future))
	}
	badDate := app.request(t, http.MethodGet, "/api/v1/service-orders?from=01-01-2026", nil, ownerToken)
	assertCode(t, badDate, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	// Pagination: 4 rows, per_page=2 → page 2 full, total_pages 2.
	_, pagedMeta := fetchList("?per_page=2&page=2", ownerToken)
	if pagedMeta.Total != 4 || pagedMeta.Page != 2 || pagedMeta.TotalPages != 2 {
		t.Fatalf("unexpected paged-meta: %+v", pagedMeta)
	}

	// Technician with the default "assigned only" setting sees just their row.
	techList, techMeta := fetchList("", techToken)
	if techMeta.Total != 1 || len(techList) != 1 || techList[0].TicketNo != "FIL-001" {
		t.Fatalf("technician scope broken: total=%d items=%+v", techMeta.Total, techList)
	}
	techDetail := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderOther.String(), nil, techToken)
	assertCode(t, techDetail, http.StatusNotFound, "")

	// Foreign outlet → 403 OUTLET_FORBIDDEN.
	foreign := uuid.New()
	foreignResult := app.request(t, http.MethodGet, "/api/v1/service-orders?outlet_id="+foreign.String(), nil, ownerToken)
	assertCode(t, foreignResult, http.StatusForbidden, "OUTLET_FORBIDDEN")

	// Tenant B must not see or touch tenant A rows.
	tenantB, err := database.CreateTenant(t.Context(), app.db, database.CreateTenantInput{
		Name: "Toko B Servis", Slug: "toko-b-servis", OwnerName: "Owner B Servis",
		OwnerEmail: "ownerb-servis@fixphone.local", OwnerPassword: "OwnerB#123",
		OutletName: "Cabang B Servis", OutletCode: "B2S",
	})
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}
	if tenantB.TenantID == app.tenantID {
		t.Fatal("tenant IDs collided")
	}
	tokenB := accessToken(t, app.login(t, "ownerb-servis@fixphone.local", "OwnerB#123"))
	_, metaB := fetchList("", tokenB)
	if metaB.Total != 0 {
		t.Fatalf("tenant B sees %d tenant A orders", metaB.Total)
	}
	isolatedRead := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderA.String(), nil, tokenB)
	assertCode(t, isolatedRead, http.StatusNotFound, "")
	isolatedWrite := app.request(t, http.MethodPut, "/api/v1/service-orders/"+orderA.String()+"/credential",
		map[string]any{"type": "pin", "value": "hack"}, tokenB)
	assertCode(t, isolatedWrite, http.StatusNotFound, "NOT_FOUND")
}

// ---------------------------------------------------------------------------
// 5. Photo upload validation + delete.
// ---------------------------------------------------------------------------

var pngHeader = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

func TestServicePhotoUploadValidationAndDelete(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	order := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"name": "Foto Customer", "phone": "081355556666"},
		"device":    map[string]any{"brand": "Oppo", "model": "A18"},
		"complaint": "Kamera buram",
	})
	orderID := order["id"].(string)

	// Valid PNG intake photo.
	valid := append(append([]byte{}, pngHeader...), bytes.Repeat([]byte{0}, 1500)...)
	upload := app.multipartRequest(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/photos",
		ownerToken, map[string]string{"stage": "intake", "is_public": "true"}, "file", "intake.png", valid)
	if upload.response.Code != http.StatusCreated {
		t.Fatalf("photo upload = %d: %s", upload.response.Code, upload.response.Body.String())
	}
	var photo struct {
		ID       string `json:"id"`
		Stage    string `json:"stage"`
		MimeType string `json:"mime_type"`
		FilePath string `json:"file_path"`
		IsPublic bool   `json:"is_public"`
	}
	if err := json.Unmarshal(upload.body.Data, &photo); err != nil {
		t.Fatalf("decode photo: %v", err)
	}
	if photo.Stage != "intake" || photo.MimeType != "image/png" || !photo.IsPublic {
		t.Fatalf("unexpected photo fields: %+v", photo)
	}
	if !strings.HasPrefix(photo.FilePath, "/uploads/services/"+orderID+"/") {
		t.Fatalf("file path unexpected: %s", photo.FilePath)
	}
	diskPath := filepath.Join(app.cfg.UploadDir, filepath.FromSlash(strings.TrimPrefix(photo.FilePath, "/uploads")))
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("uploaded file missing on disk: %v", err)
	}

	detail := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID, nil, ownerToken)
	var detailData serviceOrderDetail
	if err := json.Unmarshal(detail.body.Data, &detailData); err != nil {
		t.Fatalf("decode detail with photos: %v", err)
	}
	if detailData.PhotoCount != 1 || len(detailData.Photos) != 1 {
		t.Fatalf("photos not listed in detail: count=%d len=%d", detailData.PhotoCount, len(detailData.Photos))
	}
	if detailData.Photos[0].FilePath != photo.FilePath {
		t.Fatalf("detail photo path mismatch: %s", detailData.Photos[0].FilePath)
	}

	// Rejections: content sniffed as text, invalid stage, missing file.
	textUpload := app.multipartRequest(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/photos",
		ownerToken, nil, "file", "notes.txt", []byte("ini teks biasa bukan foto"))
	assertCode(t, textUpload, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	stageUpload := app.multipartRequest(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/photos",
		ownerToken, map[string]string{"stage": "bengkel"}, "file", "pic.png", pngHeader)
	assertCode(t, stageUpload, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	missing := app.multipartRequest(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/photos",
		ownerToken, nil, "file", "pic.png", nil)
	assertCode(t, missing, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	// Photo exceeding the 5 MB limit (but small enough to pass the 6 MB
	// multipart body limit) → 413 PAYLOAD_TOO_LARGE.
	oversize := 5 * 1024 * 1024 + (900 * 1024)
	big := app.multipartRequest(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/photos",
		ownerToken, nil, "file", "big.png",
		append(append([]byte{}, pngHeader...), bytes.Repeat([]byte{0}, oversize)...))
	assertCode(t, big, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")

	// Delete removes the row and the backing file.
	deleted := app.request(t, http.MethodDelete, "/api/v1/service-orders/"+orderID+"/photos/"+photo.ID, nil, ownerToken)
	if deleted.response.Code != http.StatusOK {
		t.Fatalf("delete photo = %d: %s", deleted.response.Code, deleted.response.Body.String())
	}
	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Fatalf("disk file not removed after delete: %v", err)
	}
	missingDelete := app.request(t, http.MethodDelete, "/api/v1/service-orders/"+orderID+"/photos/"+photo.ID, nil, ownerToken)
	assertCode(t, missingDelete, http.StatusNotFound, "NOT_FOUND")
}
