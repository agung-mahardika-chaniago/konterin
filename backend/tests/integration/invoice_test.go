package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"fixphone/backend/internal/database"

	"github.com/google/uuid"
)

type invoiceData struct {
	ID              string  `json:"id"`
	InvoiceNo       string  `json:"invoice_no"`
	ServiceOrderID  string  `json:"service_order_id"`
	TicketNo        string  `json:"ticket_no"`
	Status          string  `json:"status"`
	SubtotalService int64   `json:"subtotal_service"`
	SubtotalPart    int64   `json:"subtotal_part"`
	DiscountType    string  `json:"discount_type"`
	DiscountValue   int64   `json:"discount_value"`
	DiscountAmount  int64   `json:"discount_amount"`
	TotalAmount     int64   `json:"total_amount"`
	PaidAmount      int64   `json:"paid_amount"`
	RemainingAmount int64   `json:"remaining_amount"`
	Notes           *string `json:"notes"`
	Items           []struct {
		ID          string  `json:"id"`
		ItemType    string  `json:"item_type"`
		Description string  `json:"description"`
		Qty         float64 `json:"qty"`
		Subtotal    int64   `json:"subtotal"`
	} `json:"items"`
	Payments []struct {
		ID        string `json:"id"`
		PaymentNo string `json:"payment_no"`
		Status    string `json:"status"`
		Amount    int64  `json:"amount"`
	} `json:"payments"`
	Store struct {
		Name string `json:"name"`
	} `json:"store"`
	Order struct {
		TicketNo string `json:"ticket_no"`
	} `json:"service_order"`
}

type paymentMutation struct {
	Payment struct {
		ID        string `json:"id"`
		PaymentNo string `json:"payment_no"`
		Status    string `json:"status"`
		Amount    int64  `json:"amount"`
		Method    string `json:"method"`
	} `json:"payment"`
	Invoice struct {
		ID              string `json:"id"`
		Status          string `json:"status"`
		TotalAmount     int64  `json:"total_amount"`
		PaidAmount      int64  `json:"paid_amount"`
		RemainingAmount int64  `json:"remaining_amount"`
	} `json:"invoice"`
}

type paymentListItem struct {
	ID        string `json:"id"`
	PaymentNo string `json:"payment_no"`
	InvoiceID string `json:"invoice_id"`
	Method    string `json:"method"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"`
	PaidAt    string `json:"paid_at"`
}

type dailySummaryData struct {
	Date         string `json:"date"`
	TotalAmount  int64  `json:"total_amount"`
	PostedCount  int64  `json:"posted_count"`
	InvoiceCount int64  `json:"invoice_count"`
	VoidedAmount int64  `json:"voided_amount"`
	VoidedCount  int64  `json:"voided_count"`
	Methods      []struct {
		Method string `json:"method"`
		Amount int64  `json:"amount"`
		Count  int64  `json:"count"`
	} `json:"methods"`
}

type expenseData struct {
	ID           string  `json:"id"`
	ExpenseNo    string  `json:"expense_no"`
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Description  string  `json:"description"`
	Amount       int64   `json:"amount"`
	ExpenseDate  string  `json:"expense_date"`
	Method       string  `json:"method"`
	Notes        *string `json:"notes"`
}

type expenseListData struct {
	Items []expenseData `json:"items"`
	Total int64         `json:"total_amount"`
}

type categoryData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsSystem bool   `json:"is_system"`
}

var expenseNoPattern = regexp.MustCompile(`^EXP-\d{6}-\d{4}$`)

func addServiceLine(t *testing.T, app *authTestApp, token, orderID string, price int64) {
	t.Helper()
	result := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type": "service", "description": "Jasa servis", "qty": 1,
		"unit_price": price, "warranty_days": 30,
	}, token)
	mustStatus(t, result, http.StatusCreated, "add service item")
}

func invoiceForOrder(t *testing.T, app *authTestApp, token, orderID string) invoiceData {
	t.Helper()
	result := app.request(t, http.MethodGet, "/api/v1/service-orders/"+orderID+"/invoice", nil, token)
	mustStatus(t, result, http.StatusOK, "get invoice")
	return decodeData[invoiceData](t, result)
}

func issueInvoice(t *testing.T, app *authTestApp, token, invoiceID string) invoiceData {
	t.Helper()
	result := app.request(t, http.MethodPost, "/api/v1/invoices/"+invoiceID+"/issue", nil, token)
	mustStatus(t, result, http.StatusOK, "issue invoice")
	return decodeData[invoiceData](t, result)
}

func createPayment(t *testing.T, app *authTestApp, token, invoiceID, key string, body map[string]any) *apiResult {
	t.Helper()
	path := "/api/v1/invoices/" + invoiceID + "/payments"
	if key == "" {
		return app.request(t, http.MethodPost, path, body, token)
	}
	// The shared request helper has no header argument; use a small inline
	// recorder so the Idempotency-Key header is exercised end to end.
	return app.requestWithHeaders(t, http.MethodPost, path, body, token,
		map[string]string{"Idempotency-Key": key})
}

func TestInvoiceLifecycleAndPayments(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081300001111", "name": "Dewi Invoice"},
		"device":    map[string]any{"brand": "Samsung", "model": "A54"},
		"complaint": "Layar retak",
	})
	orderID := created["id"].(string)
	addServiceLine(t, app, ownerToken, orderID, 200000)

	initial := invoiceForOrder(t, app, ownerToken, orderID)
	if initial.Status != "draft" || initial.TotalAmount != 200000 || initial.InvoiceNo == "" {
		t.Fatalf("unexpected draft invoice: %+v", initial)
	}

	// Discount is recalculated by the server.
	patched := app.request(t, http.MethodPatch, "/api/v1/invoices/"+initial.ID, map[string]any{
		"discount_type": "percent", "discount_value": 10, "notes": "Pelanggan langganan",
	}, ownerToken)
	mustStatus(t, patched, http.StatusOK, "patch invoice")
	draft := decodeData[invoiceData](t, patched)
	if draft.DiscountAmount != 20000 || draft.TotalAmount != 180000 {
		t.Fatalf("discount totals = %d/%d, want 20000/180000", draft.DiscountAmount, draft.TotalAmount)
	}
	if draft.Notes == nil || *draft.Notes != "Pelanggan langganan" {
		t.Fatalf("invoice notes not saved: %+v", draft.Notes)
	}

	issued := issueInvoice(t, app, ownerToken, initial.ID)
	if issued.Status != "unpaid" || len(issued.Items) != 1 || issued.Payments == nil {
		t.Fatalf("unexpected issued invoice: %+v", issued)
	}
	if issued.Items[0].Subtotal != 200000 {
		t.Fatalf("snapshot subtotal = %d, want 200000", issued.Items[0].Subtotal)
	}
	if issued.Store.Name == "" {
		t.Fatal("invoice store header is empty")
	}

	// A finalised invoice may not be edited or itemised again.
	locked := app.request(t, http.MethodPatch, "/api/v1/invoices/"+initial.ID, map[string]any{
		"discount_value": 0,
	}, ownerToken)
	assertCode(t, locked, http.StatusConflict, "INVOICE_LOCKED")
	lockedItem := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type": "service", "description": "Tambahan", "qty": 1, "unit_price": 10000,
	}, ownerToken)
	assertCode(t, lockedItem, http.StatusConflict, "INVOICE_LOCKED")

	// DP: the first payment makes the invoice partial.
	first := createPayment(t, app, ownerToken, initial.ID, "idem-dp-1", map[string]any{
		"method": "cash", "amount": 50000, "notes": "DP",
	})
	mustStatus(t, first, http.StatusCreated, "create DP")
	firstData := decodeData[paymentMutation](t, first)
	if firstData.Invoice.Status != "partial" || firstData.Invoice.PaidAmount != 50000 ||
		firstData.Invoice.RemainingAmount != 130000 {
		t.Fatalf("unexpected DP invoice state: %+v", firstData.Invoice)
	}
	if firstData.Payment.PaymentNo == "" || firstData.Payment.Status != "posted" {
		t.Fatalf("unexpected DP payment: %+v", firstData.Payment)
	}

	// Retrying the same Idempotency-Key must not create a second payment.
	retry := createPayment(t, app, ownerToken, initial.ID, "idem-dp-1", map[string]any{
		"method": "cash", "amount": 50000,
	})
	mustStatus(t, retry, http.StatusCreated, "retry DP")
	retryData := decodeData[paymentMutation](t, retry)
	if retryData.Payment.ID != firstData.Payment.ID || retryData.Payment.PaymentNo != firstData.Payment.PaymentNo {
		t.Fatalf("idempotent retry created a new payment: %+v vs %+v", retryData.Payment, firstData.Payment)
	}

	overpay := createPayment(t, app, ownerToken, initial.ID, "idem-overpay", map[string]any{
		"method": "cash", "amount": 200000,
	})
	assertCode(t, overpay, http.StatusConflict, "OVERPAYMENT")

	settle := createPayment(t, app, ownerToken, initial.ID, "idem-settle", map[string]any{
		"method": "qris", "amount": 130000, "reference_no": "QRIS-2026-001",
	})
	mustStatus(t, settle, http.StatusCreated, "settle invoice")
	settled := decodeData[paymentMutation](t, settle)
	if settled.Invoice.Status != "paid" || settled.Invoice.RemainingAmount != 0 {
		t.Fatalf("unexpected settled invoice: %+v", settled.Invoice)
	}

	var paymentCount int64
	if err := app.db.Raw(`SELECT count(*) FROM payments WHERE invoice_id = ?`, initial.ID).Scan(&paymentCount).Error; err != nil {
		t.Fatalf("count payments: %v", err)
	}
	if paymentCount != 2 {
		t.Fatalf("payment count = %d, want 2", paymentCount)
	}

	// Void the settlement first: the invoice returns to partial.
	voidSettle := app.request(t, http.MethodPost, "/api/v1/payments/"+settled.Payment.ID+"/void",
		map[string]any{"reason": "QRIS gagal settle"}, ownerToken)
	mustStatus(t, voidSettle, http.StatusOK, "void settlement")
	voidData := decodeData[paymentMutation](t, voidSettle)
	if voidData.Invoice.Status != "partial" || voidData.Invoice.PaidAmount != 50000 {
		t.Fatalf("unexpected invoice after settlement void: %+v", voidData.Invoice)
	}

	voidDP := app.request(t, http.MethodPost, "/api/v1/payments/"+firstData.Payment.ID+"/void",
		map[string]any{"reason": "DP dibatalkan pelanggan"}, ownerToken)
	mustStatus(t, voidDP, http.StatusOK, "void DP")
	voidDPData := decodeData[paymentMutation](t, voidDP)
	if voidDPData.Invoice.Status != "unpaid" || voidDPData.Invoice.PaidAmount != 0 {
		t.Fatalf("unexpected invoice after DP void: %+v", voidDPData.Invoice)
	}

	doubleVoid := app.request(t, http.MethodPost, "/api/v1/payments/"+firstData.Payment.ID+"/void",
		map[string]any{"reason": "lagi"}, ownerToken)
	assertCode(t, doubleVoid, http.StatusConflict, "PAYMENT_ALREADY_VOID")

	// Only money-free invoices can be voided; after void no payment is allowed.
	voidInvoice := app.request(t, http.MethodPost, "/api/v1/invoices/"+initial.ID+"/void",
		map[string]any{"reason": "Salah input item"}, ownerToken)
	mustStatus(t, voidInvoice, http.StatusOK, "void invoice")
	voidInvoiceData := decodeData[invoiceData](t, voidInvoice)
	if voidInvoiceData.Status != "void" {
		t.Fatalf("invoice status after void = %s", voidInvoiceData.Status)
	}
	payVoid := createPayment(t, app, ownerToken, initial.ID, "", map[string]any{
		"method": "cash", "amount": 10000,
	})
	assertCode(t, payVoid, http.StatusConflict, "INVOICE_VOID")

	var auditRows []struct {
		Action string `gorm:"column:action"`
		Count  int64  `gorm:"column:count"`
	}
	if err := app.db.Raw(`
		SELECT action, count(*) AS count FROM audit_logs
		WHERE tenant_id = ?
		GROUP BY action ORDER BY action`, app.tenantID).Scan(&auditRows).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	audits := int64(0)
	for _, row := range auditRows {
		switch row.Action {
		case "payment_create", "payment_void", "invoice_update", "invoice_issue", "invoice_void":
			audits += row.Count
		}
	}
	if audits < 7 {
		t.Fatalf("stage 8 audit rows = %d, want at least 7: %+v", audits, auditRows)
	}
}

func TestCashierDailySummaryAndExpenses(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	// Cashier may not view expenses or void payments.
	app.createUser(t, ownerToken, "Kasir Pembayaran", "kasir8@fixphone.local", "cashier", []string{app.outletID.String()})
	app.allowLogin(t, "kasir8@fixphone.local")
	cashierToken := accessToken(t, app.login(t, "kasir8@fixphone.local", "Password#123"))

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081300002222", "name": "Eko Kasir"},
		"device":    map[string]any{"brand": "Oppo", "model": "Reno 10"},
		"complaint": "Baterai cepat habis",
	})
	orderID := created["id"].(string)
	addServiceLine(t, app, ownerToken, orderID, 150000)
	invoice := issueInvoice(t, app, ownerToken, invoiceForOrder(t, app, ownerToken, orderID).ID)

	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	paidAt := time.Now().In(jakarta)
	today := paidAt.Format("2006-01-02")
	cashPayment := createPayment(t, app, cashierToken, invoice.ID, "", map[string]any{
		"method": "cash", "amount": 150000, "paid_at": paidAt.Format(time.RFC3339),
	})
	mustStatus(t, cashPayment, http.StatusCreated, "cashier payment")
	cashData := decodeData[paymentMutation](t, cashPayment)
	if cashData.Invoice.Status != "paid" {
		t.Fatalf("cashier payment did not settle invoice: %+v", cashData.Invoice)
	}

	// Daily summary is available to the cashier and aggregates by method.
	summaryResult := app.request(t, http.MethodGet,
		"/api/v1/payments/daily-summary?date="+today, nil, cashierToken)
	mustStatus(t, summaryResult, http.StatusOK, "daily summary")
	summary := decodeData[dailySummaryData](t, summaryResult)
	if summary.TotalAmount != 150000 || summary.PostedCount != 1 || summary.InvoiceCount != 1 {
		t.Fatalf("unexpected daily summary: %+v", summary)
	}
	if len(summary.Methods) != 1 || summary.Methods[0].Method != "cash" || summary.Methods[0].Amount != 150000 {
		t.Fatalf("unexpected method breakdown: %+v", summary.Methods)
	}

	// Payment list and invoice payment list are scoped and paginated.
	list := app.request(t, http.MethodGet,
		"/api/v1/payments?from="+today+"&to="+today+"&method=cash", nil, cashierToken)
	mustStatus(t, list, http.StatusOK, "payment list")
	payments := decodeData[[]paymentListItem](t, list)
	meta := decodeMeta(t, list)
	if len(payments) != 1 || meta.Total != 1 || payments[0].InvoiceID != invoice.ID {
		t.Fatalf("unexpected payment list: %+v meta=%+v", payments, meta)
	}
	invoicePayments := app.request(t, http.MethodGet, "/api/v1/invoices/"+invoice.ID+"/payments", nil, cashierToken)
	mustStatus(t, invoicePayments, http.StatusOK, "invoice payments")
	if got := decodeData[[]paymentListItem](t, invoicePayments); len(got) != 1 {
		t.Fatalf("invoice payment count = %d, want 1", len(got))
	}

	// Cashier lacks payment.void and expense permissions.
	cashierVoid := app.request(t, http.MethodPost, "/api/v1/payments/"+cashData.Payment.ID+"/void",
		map[string]any{"reason": "tes"}, cashierToken)
	assertCode(t, cashierVoid, http.StatusForbidden, "FORBIDDEN")
	expenseDenied := app.request(t, http.MethodGet, "/api/v1/expenses", nil, cashierToken)
	assertCode(t, expenseDenied, http.StatusForbidden, "FORBIDDEN")

	// Expenses: categories, create, list, update, delete.
	categoryResult := app.request(t, http.MethodGet, "/api/v1/expense-categories", nil, ownerToken)
	mustStatus(t, categoryResult, http.StatusOK, "list categories")
	categories := decodeData[[]categoryData](t, categoryResult)
	if len(categories) < 6 {
		t.Fatalf("default categories = %d, want at least 6", len(categories))
	}
	var categoryID string
	for _, category := range categories {
		if category.Name == "Operasional" {
			categoryID = category.ID
		}
	}
	if categoryID == "" {
		categoryID = categories[0].ID
	}

	customResult := app.request(t, http.MethodPost, "/api/v1/expense-categories",
		map[string]any{"name": "Konsumsi"}, ownerToken)
	mustStatus(t, customResult, http.StatusCreated, "create category")
	custom := decodeData[categoryData](t, customResult)
	duplicate := app.request(t, http.MethodPost, "/api/v1/expense-categories",
		map[string]any{"name": "konsumsi"}, ownerToken)
	assertCode(t, duplicate, http.StatusConflict, "CATEGORY_EXISTS")

	expenseResult := app.request(t, http.MethodPost, "/api/v1/expenses", map[string]any{
		"category_id": custom.ID, "description": "Air minum teknisi",
		"amount": 75000, "expense_date": today, "method": "cash", "notes": "Harian",
	}, ownerToken)
	mustStatus(t, expenseResult, http.StatusCreated, "create expense")
	expense := decodeData[expenseData](t, expenseResult)
	if !expenseNoPattern.MatchString(expense.ExpenseNo) {
		t.Fatalf("expense number = %q, want EXP-YYMMDD-SEQ", expense.ExpenseNo)
	}
	if expense.CategoryName != "Konsumsi" || expense.Amount != 75000 {
		t.Fatalf("unexpected expense: %+v", expense)
	}

	expenseList := app.request(t, http.MethodGet,
		"/api/v1/expenses?from="+today+"&to="+today+"&category_id="+custom.ID, nil, ownerToken)
	mustStatus(t, expenseList, http.StatusOK, "list expenses")
	listData := decodeData[expenseListData](t, expenseList)
	expenseMeta := decodeMeta(t, expenseList)
	if listData.Total != 75000 || expenseMeta.Total != 1 || len(listData.Items) != 1 {
		t.Fatalf("unexpected expense list: %+v meta=%+v", listData, expenseMeta)
	}
	if categoryID == "" {
		t.Fatal("default expense category missing")
	}

	updated := app.request(t, http.MethodPatch, "/api/v1/expenses/"+expense.ID, map[string]any{
		"amount": 50000, "description": "Air minum dan snack",
	}, ownerToken)
	mustStatus(t, updated, http.StatusOK, "update expense")
	updatedData := decodeData[expenseData](t, updated)
	if updatedData.Amount != 50000 || updatedData.Description != "Air minum dan snack" {
		t.Fatalf("expense not updated: %+v", updatedData)
	}

	deleted := app.request(t, http.MethodDelete, "/api/v1/expenses/"+expense.ID, nil, ownerToken)
	mustStatus(t, deleted, http.StatusOK, "delete expense")
	afterDelete := app.request(t, http.MethodGet, "/api/v1/expenses", nil, ownerToken)
	mustStatus(t, afterDelete, http.StatusOK, "list after delete")
	if got := decodeData[expenseListData](t, afterDelete); len(got.Items) != 0 {
		t.Fatalf("deleted expense still listed: %+v", got.Items)
	}

	// Cross-tenant access is rejected with a generic 404.
	other, err := database.CreateTenant(t.Context(), app.db, database.CreateTenantInput{
		Name: "Toko Lain", Slug: "toko-lain", OwnerName: "Siti Owner",
		OwnerEmail: "owner2@fixphone.local", OwnerPassword: "Owner#12345",
		OutletName: "Cabang", OutletCode: "CBG",
	})
	if err != nil {
		t.Fatalf("create second tenant: %v", err)
	}
	if other.TenantID == uuid.Nil {
		t.Fatal("second tenant id is empty")
	}
	loginOther := app.login(t, "owner2@fixphone.local", "Owner#12345")
	mustStatus(t, loginOther, http.StatusOK, "login second owner")
	otherToken := accessToken(t, loginOther)
	crossTenant := app.request(t, http.MethodGet, "/api/v1/invoices/"+invoice.ID, nil, otherToken)
	assertCode(t, crossTenant, http.StatusNotFound, "NOT_FOUND")
	crossExpense := app.request(t, http.MethodPatch, "/api/v1/expenses/"+expense.ID,
		map[string]any{"amount": 1000}, otherToken)
	assertCode(t, crossExpense, http.StatusNotFound, "NOT_FOUND")
}

func TestPickupRequiresSettledInvoiceOrOwnerOverride(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)
	app.createUser(t, ownerToken, "Kasir Pickup", "kasir-pickup@fixphone.local", "cashier", []string{app.outletID.String()})
	app.allowLogin(t, "kasir-pickup@fixphone.local")
	cashierToken := accessToken(t, app.login(t, "kasir-pickup@fixphone.local", "Password#123"))

	prepareOrder := func(phone, name string, price int64) (string, invoiceData) {
		created := createServiceOrder(t, app, ownerToken, map[string]any{
			"customer":  map[string]any{"phone": phone, "name": name},
			"device":    map[string]any{"brand": "Vivo", "model": "V29"},
			"complaint": "Mati total",
		})
		orderID := created["id"].(string)
		if price > 0 {
			addServiceLine(t, app, ownerToken, orderID, price)
		}
		for _, status := range []string{"waiting_diagnosis", "in_progress", "testing", "completed", "ready_for_pickup"} {
			assertCode(t, changeStatus(t, app, ownerToken, orderID, status, nil), http.StatusOK, "")
		}
		return orderID, invoiceForOrder(t, app, ownerToken, orderID)
	}

	// Unpaid: cashier is blocked even when supplying an override reason.
	orderA, invoiceA := prepareOrder("081300003333", "Fajar Belum Lunas", 120000)
	issueInvoice(t, app, ownerToken, invoiceA.ID)
	blocked := changeStatus(t, app, cashierToken, orderA, "picked_up", nil)
	assertCode(t, blocked, http.StatusConflict, "UNPAID_PICKUP")
	cashierOverride := changeStatus(t, app, cashierToken, orderA, "picked_up",
		map[string]any{"override_reason": "Pelanggan janji bayar besok"})
	assertCode(t, cashierOverride, http.StatusConflict, "UNPAID_PICKUP")

	// Paying in full unlocks the hand-over.
	settle := createPayment(t, app, cashierToken, invoiceA.ID, "", map[string]any{
		"method": "cash", "amount": 120000,
	})
	mustStatus(t, settle, http.StatusCreated, "settle order A")
	assertCode(t, changeStatus(t, app, cashierToken, orderA, "picked_up", nil), http.StatusOK, "")

	// Owner override requires a reason and is audited.
	orderB, invoiceB := prepareOrder("081300004444", "Gita Override", 90000)
	issueInvoice(t, app, ownerToken, invoiceB.ID)
	noReason := changeStatus(t, app, ownerToken, orderB, "picked_up", nil)
	assertCode(t, noReason, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	override := changeStatus(t, app, ownerToken, orderB, "picked_up",
		map[string]any{"override_reason": "Barang milik perusahaan, dijamin owner"})
	assertCode(t, override, http.StatusOK, "")

	var overrideAudit int64
	if err := app.db.Raw(`
		SELECT count(*) FROM audit_logs
		WHERE action = 'service_pickup_override' AND entity_id = ?`, orderB).Scan(&overrideAudit).Error; err != nil {
		t.Fatalf("count override audit: %v", err)
	}
	if overrideAudit != 1 {
		t.Fatalf("override audit rows = %d, want 1", overrideAudit)
	}

	// Zero-total orders stay pickup-able without an override.
	orderC, invoiceC := prepareOrder("081300005555", "Hana Gratis", 0)
	if invoiceC.TotalAmount != 0 {
		t.Fatalf("zero-total invoice has total %d", invoiceC.TotalAmount)
	}
	assertCode(t, changeStatus(t, app, cashierToken, orderC, "picked_up", nil), http.StatusOK, "")
}

func TestZeroTotalInvoiceIssuesAsPaid(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081300006666", "name": "Iwan Gratis"},
		"device":    map[string]any{"brand": "Realme", "model": "C55"},
		"complaint": "Cek saja",
	})
	invoice := invoiceForOrder(t, app, ownerToken, created["id"].(string))
	issued := issueInvoice(t, app, ownerToken, invoice.ID)
	if issued.Status != "paid" || issued.TotalAmount != 0 || issued.RemainingAmount != 0 {
		t.Fatalf("zero-total invoice = %+v", issued)
	}
}

// requestWithHeaders mirrors request() while allowing extra headers.
func (a *authTestApp) requestWithHeaders(t *testing.T, method, path string, body any, token string,
	headers map[string]string) *apiResult {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
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
