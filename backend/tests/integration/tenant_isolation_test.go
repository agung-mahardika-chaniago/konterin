package integration_test

import (
	"net/http"
	"testing"
	"time"

	"fixphone/backend/internal/database"
)

// TestTenantIDORMatrix provisions a full set of tenant A resources and then
// proves that a tenant B owner cannot read or mutate any of them by guessing
// IDs. Cross-tenant lookups must look exactly like missing rows (404), never
// leak existence and never modify data.
func TestTenantIDORMatrix(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	// --- Tenant A resources -------------------------------------------------
	customerResult := app.request(t, http.MethodPost, "/api/v1/customers", map[string]any{
		"name": "Pelanggan Tenant A", "phone": "081255556666",
	}, ownerToken)
	mustStatus(t, customerResult, http.StatusCreated, "create customer")
	customerID := decodeData[map[string]any](t, customerResult)["id"].(string)

	createdOrder := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":       map[string]any{"phone": "081255556666", "name": "Pelanggan Tenant A"},
		"device":         map[string]any{"brand": "Apple", "model": "iPhone 13"},
		"complaint":      "Baterai cepat habis",
		"credential":     map[string]any{"type": "pin", "value": "123456"},
		"estimated_cost": 350000,
	})
	orderID := createdOrder["id"].(string)

	diagnostic := app.request(t, http.MethodPut, "/api/v1/service-orders/"+orderID+"/diagnostic", map[string]any{
		"diagnosis":    "Baterai aus",
		"findings":     "Health 72%",
		"action_taken": "Ganti baterai",
	}, ownerToken)
	mustStatus(t, diagnostic, http.StatusOK, "upsert diagnostic")

	addServiceLine(t, app, ownerToken, orderID, 200000)
	invoice := issueInvoice(t, app, ownerToken, invoiceForOrder(t, app, ownerToken, orderID).ID)
	payment := createPayment(t, app, ownerToken, invoice.ID, "idor-payment-1", map[string]any{
		"method": "cash", "amount": 50000,
	})
	mustStatus(t, payment, http.StatusCreated, "create payment")
	paymentID := decodeData[paymentMutation](t, payment).Payment.ID

	part := createSparepart(t, app, ownerToken, map[string]any{
		"sku": "IDOR-BAT-13", "name": "Baterai iPhone 13",
		"stock": 5, "min_stock": 1, "cost_price": 150000, "sell_price": 300000,
	})

	categoryResult := app.request(t, http.MethodGet, "/api/v1/expense-categories", nil, ownerToken)
	mustStatus(t, categoryResult, http.StatusOK, "list categories")
	categoryID := decodeData[[]categoryData](t, categoryResult)[0].ID
	expenseResult := app.request(t, http.MethodPost, "/api/v1/expenses", map[string]any{
		"category_id": categoryID, "description": "Konsumsi tenant A",
		"amount": 25000, "expense_date": time.Now().Format("2006-01-02"), "method": "cash",
	}, ownerToken)
	mustStatus(t, expenseResult, http.StatusCreated, "create expense")
	expenseID := decodeData[expenseData](t, expenseResult).ID

	backupResult := app.request(t, http.MethodPost, "/api/v1/settings/backups", nil, ownerToken)
	mustStatus(t, backupResult, http.StatusCreated, "create backup")
	backupID := decodeData[backupData](t, backupResult).ID

	// Warranty + claim are seeded directly: provisioning them through the full
	// pickup flow is covered by the stage 9 tests and irrelevant here.
	var warrantyID, claimID string
	if err := app.db.Raw(`
		INSERT INTO warranties (tenant_id, outlet_id, service_order_id, invoice_id, customer_id,
			warranty_no, type, start_date, end_date, days, status)
		VALUES (?, ?, ?, ?, ?, 'WAR-IDOR-0001', 'service', current_date, current_date + 30, 30, 'active')
		RETURNING id::text`,
		app.tenantID, app.outletID, orderID, invoice.ID, customerID).Scan(&warrantyID).Error; err != nil {
		t.Fatalf("seed warranty: %v", err)
	}
	if err := app.db.Raw(`
		INSERT INTO warranty_claims (tenant_id, outlet_id, warranty_id, claim_no, complaint, status)
		VALUES (?, ?, ?, 'CLM-IDOR-0001', 'Baterai kembali drop', 'submitted')
		RETURNING id::text`,
		app.tenantID, app.outletID, warrantyID).Scan(&claimID).Error; err != nil {
		t.Fatalf("seed warranty claim: %v", err)
	}

	// --- Tenant B -----------------------------------------------------------
	other, err := database.CreateTenant(t.Context(), app.db, database.CreateTenantInput{
		Name: "Toko Penantang", Slug: "toko-penantang", OwnerName: "Siti Owner",
		OwnerEmail: "owner-b@fixphone.local", OwnerPassword: "Owner#12345",
		OutletName: "Cabang", OutletCode: "CBG",
	})
	if err != nil {
		t.Fatalf("create second tenant: %v", err)
	}
	otherToken := accessToken(t, app.login(t, "owner-b@fixphone.local", "Owner#12345"))

	// Lists never include another tenant's rows.
	listCases := []struct {
		path  string
		total int64
	}{
		{"/api/v1/customers", 0},
		{"/api/v1/service-orders", 0},
		{"/api/v1/invoices", 0},
		{"/api/v1/spareparts", 0},
		{"/api/v1/users", 1}, // only the tenant B owner
		{"/api/v1/expenses", 0},
		{"/api/v1/warranties", 0},
		{"/api/v1/claims", 0},
	}
	for _, tc := range listCases {
		result := app.request(t, http.MethodGet, tc.path, nil, otherToken)
		mustStatus(t, result, http.StatusOK, "cross-tenant list "+tc.path)
		if meta := decodeMeta(t, result); meta.Total != tc.total {
			t.Errorf("%s total = %d, want %d", tc.path, meta.Total, tc.total)
		}
	}

	// Reads by guessed ID must be indistinguishable from missing rows.
	readCases := []struct{ path string }{
		{"/api/v1/customers/" + customerID},
		{"/api/v1/service-orders/" + orderID},
		{"/api/v1/service-orders/" + orderID + "/diagnostic"},
		{"/api/v1/service-orders/" + orderID + "/items"},
		{"/api/v1/service-orders/" + orderID + "/timeline"},
		{"/api/v1/service-orders/" + orderID + "/invoice"},
		{"/api/v1/service-orders/" + orderID + "/public-link"},
		{"/api/v1/invoices/" + invoice.ID},
		{"/api/v1/invoices/" + invoice.ID + "/payments"},
		{"/api/v1/spareparts/" + part.ID},
		{"/api/v1/spareparts/" + part.ID + "/movements"},
		{"/api/v1/users/" + app.ownerID.String()},
		{"/api/v1/warranties/" + warrantyID},
		{"/api/v1/claims/" + claimID},
		{"/api/v1/settings/backups/" + backupID + "/download"},
	}
	for _, tc := range readCases {
		result := app.request(t, http.MethodGet, tc.path, nil, otherToken)
		assertCode(t, result, http.StatusNotFound, "NOT_FOUND")
	}
	// Payment-level reads are not exposed by ID directly; verify through void.
	assertCode(t, app.request(t, http.MethodPost, "/api/v1/payments/"+paymentID+"/void",
		map[string]any{"reason": "cross tenant"}, otherToken), http.StatusNotFound, "NOT_FOUND")

	// Mutations must not touch another tenant's rows.
	writeCases := []struct {
		method, path string
		body         any
		label        string
	}{
		{http.MethodPatch, "/api/v1/customers/" + customerID, map[string]any{"name": "Diretas"}, "patch customer"},
		{http.MethodDelete, "/api/v1/customers/" + customerID, nil, "delete customer"},
		{http.MethodPatch, "/api/v1/service-orders/" + orderID, map[string]any{"complaint": "Diretas"}, "patch service order"},
		{http.MethodPatch, "/api/v1/service-orders/" + orderID + "/status", map[string]any{"status": "in_progress"}, "change status"},
		{http.MethodPost, "/api/v1/service-orders/" + orderID + "/items", map[string]any{
			"item_type": "service", "description": "Injeksi", "qty": 1, "unit_price": 1000,
		}, "add item"},
		{http.MethodPut, "/api/v1/service-orders/" + orderID + "/diagnostic", map[string]any{"diagnosis": "Diretas"}, "write diagnostic"},
		{http.MethodPut, "/api/v1/service-orders/" + orderID + "/credential", map[string]any{"type": "pin", "value": "999999"}, "overwrite credential"},
		{http.MethodPost, "/api/v1/service-orders/" + orderID + "/credential/reveal", map[string]any{"reason": "mencoba mencuri"}, "reveal credential"},
		{http.MethodPatch, "/api/v1/invoices/" + invoice.ID, map[string]any{"notes": "Diretas"}, "patch invoice"},
		{http.MethodPost, "/api/v1/invoices/" + invoice.ID + "/payments", map[string]any{"method": "cash", "amount": 1000}, "pay foreign invoice"},
		{http.MethodPost, "/api/v1/invoices/" + invoice.ID + "/void", map[string]any{"reason": "Diretas"}, "void foreign invoice"},
		{http.MethodPatch, "/api/v1/spareparts/" + part.ID, map[string]any{"name": "Diretas"}, "patch sparepart"},
		{http.MethodDelete, "/api/v1/spareparts/" + part.ID, nil, "delete sparepart"},
		{http.MethodPatch, "/api/v1/users/" + app.ownerID.String(), map[string]any{"name": "Diretas"}, "patch user"},
		{http.MethodDelete, "/api/v1/users/" + app.ownerID.String(), nil, "delete user"},
		{http.MethodPatch, "/api/v1/outlets/" + app.outletID.String(), map[string]any{"name": "Diretas"}, "patch outlet"},
		{http.MethodDelete, "/api/v1/expenses/" + expenseID, nil, "delete expense"},
		{http.MethodPatch, "/api/v1/claims/" + claimID, map[string]any{"status": "approved"}, "patch claim"},
	}
	for _, tc := range writeCases {
		result := app.request(t, tc.method, tc.path, tc.body, otherToken)
		if result.response.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404: %s", tc.label, result.response.Code, result.response.Body.String())
		}
	}

	// The original rows must be intact when checked as tenant A / superuser.
	var customerName, orderComplaint, invoiceNotes, partName string
	if err := app.db.Raw(`SELECT name FROM customers WHERE id = ?`, customerID).Scan(&customerName).Error; err != nil {
		t.Fatalf("read customer: %v", err)
	}
	if customerName != "Pelanggan Tenant A" {
		t.Errorf("customer was modified: %q", customerName)
	}
	if err := app.db.Raw(`SELECT complaint FROM service_orders WHERE id = ?`, orderID).Scan(&orderComplaint).Error; err != nil {
		t.Fatalf("read order: %v", err)
	}
	if orderComplaint != "Baterai cepat habis" {
		t.Errorf("service order was modified: %q", orderComplaint)
	}
	if err := app.db.Raw(`SELECT coalesce(notes, '') FROM invoices WHERE id = ?`, invoice.ID).Scan(&invoiceNotes).Error; err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if invoiceNotes != "" {
		t.Errorf("invoice notes were modified: %q", invoiceNotes)
	}
	if err := app.db.Raw(`SELECT name FROM spareparts WHERE id = ?`, part.ID).Scan(&partName).Error; err != nil {
		t.Fatalf("read sparepart: %v", err)
	}
	if partName != "Baterai iPhone 13" {
		t.Errorf("sparepart was modified: %q", partName)
	}
	var status string
	if err := app.db.Raw(`SELECT status FROM service_orders WHERE id = ?`, orderID).Scan(&status).Error; err != nil {
		t.Fatalf("read order status: %v", err)
	}
	if status != "received" {
		t.Errorf("service order status changed to %q", status)
	}
	if other.TenantID.String() == app.tenantID.String() {
		t.Fatal("second tenant reused tenant A id")
	}
}
