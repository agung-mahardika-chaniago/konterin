package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// changeStatus sends a status transition and fails the test on an unexpected code.
func changeStatus(t *testing.T, app *authTestApp, token, orderID, status string, extra map[string]any) *apiResult {
	t.Helper()
	body := map[string]any{"status": status}
	for key, value := range extra {
		body[key] = value
	}
	return app.request(t, http.MethodPatch, "/api/v1/service-orders/"+orderID+"/status", body, token)
}

func TestServiceRepairStatusMachineAndDiagnosis(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":       map[string]any{"phone": "081200001111", "name": "Rina Status"},
		"device":         map[string]any{"brand": "Xiaomi", "model": "Redmi Note 12"},
		"complaint":      "Mati total setelah kena air",
		"priority":       "normal",
		"estimated_cost": 0,
	})
	orderID := created["id"].(string)

	// Illegal jump straight from received to in_progress.
	illegal := changeStatus(t, app, ownerToken, orderID, "in_progress", nil)
	assertCode(t, illegal, http.StatusConflict, "INVALID_STATUS_TRANSITION")

	// Received -> waiting_diagnosis.
	step := changeStatus(t, app, ownerToken, orderID, "waiting_diagnosis", map[string]any{"notes": "Masuk meja teknisi"})
	assertCode(t, step, http.StatusOK, "")

	// Save diagnosis findings and estimated completion.
	diagnostic := app.request(t, http.MethodPut, "/api/v1/service-orders/"+orderID+"/diagnostic", map[string]any{
		"diagnosis":         "Korsleting pada IC power",
		"findings":          "Board terlihat korosi di area charging IC",
		"action_taken":      "Pembersihan ultrasonik dan penggantian IC",
		"test_result":       "Belum dites ulang",
		"technician_notes":  "Perlu pengeringan 2 jam",
		"estimated_done_at": "2026-11-08T10:00:00+07:00",
	}, ownerToken)
	assertCode(t, diagnostic, http.StatusOK, "")

	var diagData struct {
		ID              string  `json:"id"`
		Diagnosis       *string `json:"diagnosis"`
		Findings        *string `json:"findings"`
		DiagnosedBy     *string `json:"diagnosed_by"`
		EstimatedDoneAt *string `json:"estimated_done_at"`
	}
	if err := json.Unmarshal(diagnostic.body.Data, &diagData); err != nil {
		t.Fatalf("decode diagnostic: %v", err)
	}
	if diagData.ID == "" || diagData.Diagnosis == nil || *diagData.Diagnosis != "Korsleting pada IC power" {
		t.Fatalf("unexpected diagnostic: %+v", diagData)
	}
	if diagData.DiagnosedBy == nil || *diagData.DiagnosedBy == "" {
		t.Fatalf("diagnosed_by missing: %+v", diagData)
	}
	if diagData.EstimatedDoneAt == nil {
		t.Fatalf("estimated_done_at missing: %+v", diagData)
	}

	// Estimated completion is synced to the service order.
	var orderEstimate *string
	if err := app.db.Raw(`SELECT estimated_done_at::text FROM service_orders WHERE id = ?`, orderID).
		Scan(&orderEstimate).Error; err != nil {
		t.Fatalf("read order estimate: %v", err)
	}
	if orderEstimate == nil {
		t.Fatal("service order estimated_done_at was not synced from the diagnosis")
	}

	// Full happy path to picked_up.
	for _, status := range []string{"in_progress", "testing", "completed", "ready_for_pickup", "picked_up"} {
		result := changeStatus(t, app, ownerToken, orderID, status, nil)
		assertCode(t, result, http.StatusOK, "")
	}

	var timestamps struct {
		StartedAt         *string `gorm:"column:started_at"`
		CompletedAt       *string `gorm:"column:completed_at"`
		PickedUpAt        *string `gorm:"column:picked_up_at"`
		CredentialPurgeAt *string `gorm:"column:credential_purge_at"`
		Status            string  `gorm:"column:status"`
	}
	if err := app.db.Raw(`
		SELECT status, started_at::text, completed_at::text, picked_up_at::text, credential_purge_at::text
		FROM service_orders WHERE id = ?`, orderID).Scan(&timestamps).Error; err != nil {
		t.Fatalf("read order timestamps: %v", err)
	}
	if timestamps.Status != "picked_up" || timestamps.StartedAt == nil ||
		timestamps.CompletedAt == nil || timestamps.PickedUpAt == nil || timestamps.CredentialPurgeAt == nil {
		t.Fatalf("unexpected pickup timestamps: %+v", timestamps)
	}

	// Terminal statuses cannot transition again.
	terminal := changeStatus(t, app, ownerToken, orderID, "cancelled", map[string]any{"cancel_reason": "tes"})
	assertCode(t, terminal, http.StatusConflict, "INVALID_STATUS_TRANSITION")

	// Detail embeds diagnosis and a complete timeline.
	detail := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID, nil, ownerToken)
	assertCode(t, detail, http.StatusOK, "")
	var detailData struct {
		Diagnostic *struct {
			Diagnosis *string `json:"diagnosis"`
		} `json:"diagnostic"`
		Timeline []struct {
			ToStatus string `json:"to_status"`
		} `json:"timeline"`
	}
	if err := json.Unmarshal(detail.body.Data, &detailData); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detailData.Diagnostic == nil || detailData.Diagnostic.Diagnosis == nil {
		t.Fatalf("detail missing diagnostic: %+v", detailData.Diagnostic)
	}
	if len(detailData.Timeline) != 7 {
		t.Fatalf("timeline entries = %d, want 7", len(detailData.Timeline))
	}
}

func TestServiceRepairPermissionsAndAssignment(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	app.createUser(t, ownerToken, "Teknisi Satu", "teknisi@fixphone.local", "technician", []string{app.outletID.String()})
	app.allowLogin(t, "teknisi@fixphone.local")
	techToken := accessToken(t, app.login(t, "teknisi@fixphone.local", "Password#123"))

	app.createUser(t, ownerToken, "Kasir Satu", "kasir@fixphone.local", "cashier", []string{app.outletID.String()})
	app.allowLogin(t, "kasir@fixphone.local")
	cashierToken := accessToken(t, app.login(t, "kasir@fixphone.local", "Password#123"))

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081200002222", "name": "Budi Izin"},
		"device":    map[string]any{"brand": "Oppo", "model": "A78"},
		"complaint": "Layar pecah",
	})
	orderID := created["id"].(string)

	// Cancellation requires a reason before any state change.
	cancelNoReason := changeStatus(t, app, ownerToken, orderID, "cancelled", nil)
	assertCode(t, cancelNoReason, http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	var techID string
	if err := app.db.Raw(`SELECT id::text FROM users WHERE email = 'teknisi@fixphone.local'`).Scan(&techID).Error; err != nil {
		t.Fatalf("read technician id: %v", err)
	}

	// Technician cannot assign or cancel.
	assignDenied := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/assign",
		map[string]any{"technician_id": techID, "priority": "express"}, techToken)
	assertCode(t, assignDenied, http.StatusForbidden, "FORBIDDEN")

	// Owner assigns the technician and sets express priority.
	assigned := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/assign",
		map[string]any{"technician_id": techID, "priority": "express"}, ownerToken)
	assertCode(t, assigned, http.StatusOK, "")
	var assignedData struct {
		Priority   string `json:"priority"`
		Technician *struct {
			ID string `json:"id"`
		} `json:"assigned_technician"`
	}
	if err := json.Unmarshal(assigned.body.Data, &assignedData); err != nil {
		t.Fatalf("decode assigned order: %v", err)
	}
	if assignedData.Priority != "express" || assignedData.Technician == nil || assignedData.Technician.ID != techID {
		t.Fatalf("assignment failed: %+v", assignedData)
	}

	// Technician can work the order they are assigned to but still cannot cancel.
	assertCode(t, changeStatus(t, app, ownerToken, orderID, "waiting_diagnosis", nil), http.StatusOK, "")
	cancelDenied := changeStatus(t, app, techToken, orderID, "cancelled", map[string]any{"cancel_reason": "tidak jadi"})
	assertCode(t, cancelDenied, http.StatusForbidden, "FORBIDDEN")
	assertCode(t, changeStatus(t, app, techToken, orderID, "in_progress", nil), http.StatusOK, "")

	// Cashier may only perform hand-over transitions.
	cashierDenied := changeStatus(t, app, cashierToken, orderID, "testing", nil)
	assertCode(t, cashierDenied, http.StatusForbidden, "FORBIDDEN")

	// Owner moves the order to ready_for_pickup, then the cashier hands it over.
	assertCode(t, changeStatus(t, app, ownerToken, orderID, "testing", nil), http.StatusOK, "")
	assertCode(t, changeStatus(t, app, ownerToken, orderID, "completed", nil), http.StatusOK, "")
	assertCode(t, changeStatus(t, app, ownerToken, orderID, "ready_for_pickup", nil), http.StatusOK, "")
	assertCode(t, changeStatus(t, app, cashierToken, orderID, "picked_up", nil), http.StatusOK, "")
}

func TestServiceRepairItemsStockAndInvoiceTotals(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081200003333", "name": "Sari Sparepart"},
		"device":    map[string]any{"brand": "Samsung", "model": "A34"},
		"complaint": "Layar blank",
	})
	orderID := created["id"].(string)

	sparepartID := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO spareparts (id, tenant_id, sku, name, stock, min_stock, cost_price, sell_price, is_active)
		VALUES (?, ?, 'LCD-A34', 'LCD Samsung A34 Original', 10, 2, 150000, 350000, true)`,
		sparepartID, app.tenantID).Error; err != nil {
		t.Fatalf("insert sparepart: %v", err)
	}

	// Add a service line with a line discount.
	serviceItem := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type":       "service",
		"description":     "Jasa ganti LCD",
		"qty":             1,
		"unit_price":      100000,
		"discount_amount": 10000,
		"warranty_days":   30,
	}, ownerToken)
	assertCode(t, serviceItem, http.StatusCreated, "")
	var serviceData struct {
		ID       string  `json:"id"`
		Subtotal int64   `json:"subtotal"`
		Qty      float64 `json:"qty"`
	}
	if err := json.Unmarshal(serviceItem.body.Data, &serviceData); err != nil {
		t.Fatalf("decode service item: %v", err)
	}
	if serviceData.Subtotal != 90000 {
		t.Fatalf("service subtotal = %d, want 90000", serviceData.Subtotal)
	}

	// Add two spareparts; unit price defaults to sell price and stock drops.
	sparepartItem := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type":    "sparepart",
		"sparepart_id": sparepartID.String(),
		"qty":          2,
	}, ownerToken)
	assertCode(t, sparepartItem, http.StatusCreated, "")
	var sparepartData struct {
		ID        string  `json:"id"`
		UnitPrice int64   `json:"unit_price"`
		Subtotal  int64   `json:"subtotal"`
		Qty       float64 `json:"qty"`
	}
	if err := json.Unmarshal(sparepartItem.body.Data, &sparepartData); err != nil {
		t.Fatalf("decode sparepart item: %v", err)
	}
	if sparepartData.UnitPrice != 350000 || sparepartData.Subtotal != 700000 {
		t.Fatalf("sparepart pricing = %d/%d", sparepartData.UnitPrice, sparepartData.Subtotal)
	}

	var stock int
	if err := app.db.Raw(`SELECT stock FROM spareparts WHERE id = ?`, sparepartID).Scan(&stock).Error; err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if stock != 8 {
		t.Fatalf("stock after adding 2 = %d, want 8", stock)
	}

	var totals struct {
		SubtotalService int64 `gorm:"column:subtotal_service"`
		SubtotalPart    int64 `gorm:"column:subtotal_part"`
		TotalAmount     int64 `gorm:"column:total_amount"`
	}
	if err := app.db.Raw(`
		SELECT subtotal_service, subtotal_part, total_amount FROM invoices WHERE service_order_id = ?`, orderID).
		Scan(&totals).Error; err != nil {
		t.Fatalf("read invoice totals: %v", err)
	}
	if totals.SubtotalService != 90000 || totals.SubtotalPart != 700000 || totals.TotalAmount != 790000 {
		t.Fatalf("invoice totals = %+v", totals)
	}

	// Insufficient stock is rejected atomically.
	insufficient := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type":    "sparepart",
		"sparepart_id": sparepartID.String(),
		"qty":          99,
	}, ownerToken)
	assertCode(t, insufficient, http.StatusConflict, "INSUFFICIENT_STOCK")

	var itemCount int64
	if err := app.db.Raw(`SELECT count(*) FROM service_order_items WHERE service_order_id = ?`, orderID).Scan(&itemCount).Error; err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 2 {
		t.Fatalf("item count after failed add = %d, want 2", itemCount)
	}

	// Editing the sparepart quantity adjusts stock by the delta only.
	updated := app.request(t, http.MethodPatch,
		"/api/v1/service-orders/"+orderID+"/items/"+sparepartData.ID,
		map[string]any{"qty": 3}, ownerToken)
	assertCode(t, updated, http.StatusOK, "")
	var updatedData struct {
		Qty      float64 `json:"qty"`
		Subtotal int64   `json:"subtotal"`
	}
	if err := json.Unmarshal(updated.body.Data, &updatedData); err != nil {
		t.Fatalf("decode updated item: %v", err)
	}
	if updatedData.Qty != 3 || updatedData.Subtotal != 1050000 {
		t.Fatalf("updated item = %+v", updatedData)
	}
	if err := app.db.Raw(`SELECT stock FROM spareparts WHERE id = ?`, sparepartID).Scan(&stock).Error; err != nil {
		t.Fatalf("read stock after edit: %v", err)
	}
	if stock != 7 {
		t.Fatalf("stock after editing 2 -> 3 = %d, want 7", stock)
	}

	// Updating a service line recomputes the line and invoice.
	serviceUpdated := app.request(t, http.MethodPatch,
		"/api/v1/service-orders/"+orderID+"/items/"+serviceData.ID,
		map[string]any{"qty": 2, "discount_amount": 10000}, ownerToken)
	assertCode(t, serviceUpdated, http.StatusOK, "")
	var serviceUpdatedData struct {
		Subtotal int64 `json:"subtotal"`
	}
	if err := json.Unmarshal(serviceUpdated.body.Data, &serviceUpdatedData); err != nil {
		t.Fatalf("decode updated service item: %v", err)
	}
	if serviceUpdatedData.Subtotal != 190000 {
		t.Fatalf("updated service subtotal = %d, want 190000", serviceUpdatedData.Subtotal)
	}

	// Deleting the sparepart line restocks it.
	deleted := app.request(t, http.MethodDelete,
		"/api/v1/service-orders/"+orderID+"/items/"+sparepartData.ID, nil, ownerToken)
	assertCode(t, deleted, http.StatusOK, "")
	if err := app.db.Raw(`SELECT stock FROM spareparts WHERE id = ?`, sparepartID).Scan(&stock).Error; err != nil {
		t.Fatalf("read stock after delete: %v", err)
	}
	if stock != 10 {
		t.Fatalf("stock after deleting sparepart = %d, want 10", stock)
	}
	if err := app.db.Raw(`
		SELECT subtotal_service, subtotal_part, total_amount FROM invoices WHERE service_order_id = ?`, orderID).
		Scan(&totals).Error; err != nil {
		t.Fatalf("read invoice totals after delete: %v", err)
	}
	if totals.TotalAmount != 190000 || totals.SubtotalPart != 0 {
		t.Fatalf("invoice totals after delete = %+v", totals)
	}

	// Finalized invoices freeze line items.
	if err := app.db.Exec(`UPDATE invoices SET status = 'unpaid' WHERE service_order_id = ?`, orderID).Error; err != nil {
		t.Fatalf("finalize invoice: %v", err)
	}
	locked := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type": "service", "description": "Tambahan", "qty": 1, "unit_price": 50000,
	}, ownerToken)
	assertCode(t, locked, http.StatusConflict, "INVOICE_LOCKED")

	// Cost price is only exposed to actors with inventory.cost.view.
	searchOwner := app.request(t, http.MethodGet, "/api/v1/spareparts/search?q=LCD", nil, ownerToken)
	assertCode(t, searchOwner, http.StatusOK, "")
	var ownerOptions []struct {
		CostPrice int64 `json:"cost_price"`
	}
	if err := json.Unmarshal(searchOwner.body.Data, &ownerOptions); err != nil {
		t.Fatalf("decode owner sparepart search: %v", err)
	}
	if len(ownerOptions) != 1 || ownerOptions[0].CostPrice != 150000 {
		t.Fatalf("owner cost price = %+v", ownerOptions)
	}

	app.createUser(t, ownerToken, "Teknisi Biaya", "teknisi-biaya@fixphone.local", "technician", []string{app.outletID.String()})
	app.allowLogin(t, "teknisi-biaya@fixphone.local")
	techToken := accessToken(t, app.login(t, "teknisi-biaya@fixphone.local", "Password#123"))
	searchTech := app.request(t, http.MethodGet, "/api/v1/spareparts/search?q=LCD", nil, techToken)
	assertCode(t, searchTech, http.StatusOK, "")
	var techOptions []struct {
		CostPrice int64 `json:"cost_price"`
	}
	if err := json.Unmarshal(searchTech.body.Data, &techOptions); err != nil {
		t.Fatalf("decode technician sparepart search: %v", err)
	}
	if len(techOptions) != 1 || techOptions[0].CostPrice != 0 {
		t.Fatalf("technician must not see cost price: %+v", techOptions)
	}
}

func TestServiceRepairCancellationVoidsUnpaidInvoiceAndRestocks(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081200004444", "name": "Dewi Batal"},
		"device":    map[string]any{"brand": "Vivo", "model": "Y36"},
		"complaint": "Tidak bisa charging",
	})
	orderID := created["id"].(string)

	sparepartID := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO spareparts (id, tenant_id, sku, name, stock, min_stock, cost_price, sell_price, is_active)
		VALUES (?, ?, 'IC-Y36', 'IC Charging Vivo Y36', 5, 1, 40000, 120000, true)`,
		sparepartID, app.tenantID).Error; err != nil {
		t.Fatalf("insert sparepart: %v", err)
	}
	item := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type":    "sparepart",
		"sparepart_id": sparepartID.String(),
		"qty":          2,
	}, ownerToken)
	assertCode(t, item, http.StatusCreated, "")

	// Issue the invoice but leave it unpaid. Cancellation must void it (BR-10).
	if err := app.db.Exec(`UPDATE invoices SET status = 'unpaid' WHERE service_order_id = ?`, orderID).Error; err != nil {
		t.Fatalf("issue invoice: %v", err)
	}
	cancelled := changeStatus(t, app, ownerToken, orderID, "cancelled", map[string]any{
		"cancel_reason": "Pelanggan membatalkan perbaikan",
	})
	assertCode(t, cancelled, http.StatusOK, "")

	var invoiceStatus string
	if err := app.db.Raw(`SELECT status FROM invoices WHERE service_order_id = ?`, orderID).Scan(&invoiceStatus).Error; err != nil {
		t.Fatalf("read invoice status: %v", err)
	}
	if invoiceStatus != "void" {
		t.Fatalf("invoice status after cancel = %q, want void", invoiceStatus)
	}

	var stock int
	if err := app.db.Raw(`SELECT stock FROM spareparts WHERE id = ?`, sparepartID).Scan(&stock).Error; err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if stock != 5 {
		t.Fatalf("stock after cancel = %d, want 5", stock)
	}

	var returns int64
	if err := app.db.Raw(`
		SELECT count(*) FROM inventory_movements
		WHERE reference_id = ? AND movement_type = 'service_return'`, orderID).Scan(&returns).Error; err != nil {
		t.Fatalf("count return movements: %v", err)
	}
	if returns != 1 {
		t.Fatalf("service_return movements = %d, want 1", returns)
	}

	// A paid invoice blocks cancellation.
	paidOrder := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081200005555", "name": "Eko Lunas"},
		"device":    map[string]any{"brand": "Realme", "model": "C55"},
		"complaint": "Layar bergaris",
	})
	paidOrderID := paidOrder["id"].(string)
	if err := app.db.Exec(`
		UPDATE invoices SET status = 'paid', total_amount = 100000, paid_amount = 100000
		WHERE service_order_id = ?`, paidOrderID).Error; err != nil {
		t.Fatalf("mark invoice paid: %v", err)
	}
	blocked := changeStatus(t, app, ownerToken, paidOrderID, "cancelled", map[string]any{
		"cancel_reason": "Pelanggan membatalkan perbaikan",
	})
	assertCode(t, blocked, http.StatusConflict, "INVOICE_LOCKED")
}
