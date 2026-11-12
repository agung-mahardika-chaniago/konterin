package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type revenueReportData struct {
	Period string `json:"period"`
	From   string `json:"from"`
	To     string `json:"to"`
	Items  []struct {
		Label         string `json:"label"`
		InvoiceCount  int64  `json:"invoice_count"`
		ServiceCount  int64  `json:"service_count"`
		ServiceAmount int64  `json:"service_amount"`
		PartAmount    int64  `json:"part_amount"`
		DiscountAmount int64 `json:"discount_amount"`
		TotalRevenue  int64  `json:"total_revenue"`
		PaidAmount    int64  `json:"paid_amount"`
		UnpaidAmount  int64  `json:"unpaid_amount"`
	} `json:"items"`
	Total struct {
		InvoiceCount   int64 `json:"invoice_count"`
		ServiceCount   int64 `json:"service_count"`
		ServiceAmount  int64 `json:"service_amount"`
		PartAmount     int64 `json:"part_amount"`
		DiscountAmount int64 `json:"discount_amount"`
		TotalRevenue   int64 `json:"total_revenue"`
		PaidAmount     int64 `json:"paid_amount"`
		UnpaidAmount   int64 `json:"unpaid_amount"`
	} `json:"total"`
}

type servicesReportData struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Total   int64  `json:"total"`
	Completed int64 `json:"completed"`
	InProgress int64 `json:"in_progress"`
	StatusBreakdown []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	} `json:"status_breakdown"`
	TechnicianRank []struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	} `json:"technician_rank"`
}

type brandsReportData struct {
	Items []struct {
		Brand   string `json:"brand"`
		Count   int64  `json:"count"`
		Revenue int64  `json:"revenue"`
	} `json:"items"`
}

type serviceVsPartsReportData struct {
	Items []struct {
		ItemType string `json:"item_type"`
		Qty      int64  `json:"qty"`
		Amount   int64  `json:"amount"`
	} `json:"items"`
}

type profitReportData struct {
	Revenue      int64 `json:"revenue"`
	Cost         int64 `json:"cost"`
	GrossProfit  int64 `json:"gross_profit"`
	ExpenseTotal int64 `json:"expense_total"`
	NetProfit    int64 `json:"net_profit"`
	InvoiceCount int64 `json:"invoice_count"`
	AvgInvoice   int64 `json:"avg_invoice"`
}

type expensesReportData struct {
	Items []struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
		Amount   int64  `json:"amount"`
	} `json:"items"`
	Total int64 `json:"total"`
}

type techniciansReportData struct {
	Items []struct {
		Name           string  `json:"name"`
		AssignedCount  int64   `json:"assigned_count"`
		CompletedCount int64   `json:"completed_count"`
		ActiveCount    int64   `json:"active_count"`
		AvgRepairHours float64 `json:"avg_repair_hours"`
		RevenueContrib int64   `json:"revenue_contribution"`
	} `json:"items"`
}

type stockReportData struct {
	Items []struct {
		SKU        string `json:"sku"`
		Name       string `json:"name"`
		Stock      int64  `json:"stock"`
		MinStock   int64  `json:"min_stock"`
		CostPrice  int64  `json:"cost_price"`
		StockValue int64  `json:"stock_value"`
		IsLow      bool   `json:"is_low"`
	} `json:"items"`
	TotalValue    int64 `json:"total_value"`
	TotalItems    int64 `json:"total_items"`
	LowStockRows  int64 `json:"low_stock_rows"`
}

type settingsData struct {
	StoreName                  string `json:"store_name"`
	Timezone                   string `json:"timezone"`
	TicketPrefix               string `json:"ticket_prefix"`
	InvoiceFooter              *string `json:"invoice_footer"`
	ReceiptPaperWidth          int    `json:"receipt_paper_width"`
	DefaultServiceWarrantyDays int    `json:"default_service_warranty_days"`
	WAProvider                 string `json:"wa_provider"`
	UpdatedAt                  string `json:"updated_at"`
}

type backupData struct {
	ID         string `json:"id"`
	FileName   string `json:"file_name"`
	SizeBytes  int64  `json:"size_bytes"`
	Status     string `json:"status"`
	TableCount int    `json:"table_count"`
	RowCount    int64  `json:"row_count"`
	CreatedBy  *string `json:"created_by"`
}

type auditListData struct {
	Items []struct {
		ID         int64           `json:"id"`
		ActorName  string          `json:"actor_name"`
		Action     string          `json:"action"`
		Module     string          `json:"module"`
		EntityType string          `json:"entity_type"`
		After      json.RawMessage `json:"after_data"`
		CreatedAt  time.Time       `json:"created_at"`
	} `json:"items"`
}

// stage10Fixture seeds one complete repair + payment + expense flow so every
// Stage 10 report has non-trivial numbers to aggregate.
type stage10Fixture struct {
	orderID    string
	invoiceID  string
	sparepartID string
	techID     string
}

// requestBinary performs a request whose response body is not JSON (xlsx,
// gzip) and therefore must skip the envelope decoder.
func (a *authTestApp) requestBinary(t *testing.T, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.engine.ServeHTTP(rec, req)
	return rec
}

func seedStage10Flow(t *testing.T, app *authTestApp, ownerToken string) stage10Fixture {
	t.Helper()

	tech := app.createUser(t, ownerToken, "Tono Teknisi", "teknisi.rpt@fixphone.local",
		"technician", []string{app.outletID.String()})
	app.allowLogin(t, "teknisi.rpt@fixphone.local")

	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081300008888", "name": "Rina Laporan"},
		"device":    map[string]any{"brand": "Samsung", "model": "A54"},
		"complaint": "Layar retak",
	})
	orderID := created["id"].(string)

	assigned := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/assign",
		map[string]any{"technician_id": tech["id"], "priority": "normal"}, ownerToken)
	mustStatus(t, assigned, http.StatusOK, "assign technician")

	sparepartID := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO spareparts (id, tenant_id, sku, name, stock, min_stock, cost_price, sell_price, is_active)
		VALUES (?, ?, 'LCD-A54-RPT', 'LCD Samsung A54', 5, 2, 100000, 250000, true)`,
		sparepartID, app.tenantID).Error; err != nil {
		t.Fatalf("insert sparepart: %v", err)
	}

	addServiceLine(t, app, ownerToken, orderID, 150000)
	partItem := app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderID+"/items", map[string]any{
		"item_type": "sparepart", "sparepart_id": sparepartID.String(), "qty": 1,
	}, ownerToken)
	mustStatus(t, partItem, http.StatusCreated, "add sparepart item")

	invoice := invoiceForOrder(t, app, ownerToken, orderID)
	if invoice.TotalAmount != 400000 {
		t.Fatalf("invoice total = %d, want 400000", invoice.TotalAmount)
	}
	issueInvoice(t, app, ownerToken, invoice.ID)
	paid := createPayment(t, app, ownerToken, invoice.ID, "stage10-pay-1", map[string]any{
		"method": "cash", "amount": 400000,
	})
	mustStatus(t, paid, http.StatusCreated, "pay invoice")

	categories := app.request(t, http.MethodGet, "/api/v1/expense-categories", nil, ownerToken)
	mustStatus(t, categories, http.StatusOK, "list expense categories")
	var categoryList []categoryData
	if err := json.Unmarshal(categories.body.Data, &categoryList); err != nil || len(categoryList) == 0 {
		t.Fatalf("expense categories missing: %v", err)
	}
	expense := app.request(t, http.MethodPost, "/api/v1/expenses", map[string]any{
		"category_id":  categoryList[0].ID,
		"description":  "Listrik bulan ini",
		"amount":       50000,
		"expense_date": time.Now().Format("2006-01-02"),
		"method":       "cash",
	}, ownerToken)
	mustStatus(t, expense, http.StatusCreated, "create expense")

	return stage10Fixture{orderID: orderID, invoiceID: invoice.ID, sparepartID: sparepartID.String(), techID: tech["id"].(string)}
}

func TestStage10ReportsAggregatesAndExcelExport(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)
	seedStage10Flow(t, app, ownerToken)

	// Revenue: one invoice, service + part lines, fully paid.
	revenue := app.request(t, http.MethodGet, "/api/v1/reports/revenue?period=daily", nil, ownerToken)
	mustStatus(t, revenue, http.StatusOK, "revenue report")
	revenueData := decodeData[revenueReportData](t, revenue)
	if revenueData.Period != "daily" || revenueData.Total.InvoiceCount != 1 || revenueData.Total.ServiceCount != 1 {
		t.Fatalf("unexpected revenue totals: %+v", revenueData.Total)
	}
	if revenueData.Total.ServiceAmount != 150000 || revenueData.Total.PartAmount != 250000 {
		t.Fatalf("revenue split = %d/%d, want 150000/250000",
			revenueData.Total.ServiceAmount, revenueData.Total.PartAmount)
	}
	if revenueData.Total.TotalRevenue != 400000 || revenueData.Total.PaidAmount != 400000 ||
		revenueData.Total.UnpaidAmount != 0 || revenueData.Total.DiscountAmount != 0 {
		t.Fatalf("unexpected revenue money totals: %+v", revenueData.Total)
	}
	if len(revenueData.Items) == 0 || revenueData.Items[0].Label == "" {
		t.Fatalf("revenue buckets empty: %+v", revenueData.Items)
	}

	// Services: one order, assigned to a technician, still in progress.
	services := app.request(t, http.MethodGet, "/api/v1/reports/services", nil, ownerToken)
	mustStatus(t, services, http.StatusOK, "services report")
	servicesData := decodeData[servicesReportData](t, services)
	if servicesData.Total != 1 || servicesData.InProgress != 1 || servicesData.Completed != 0 {
		t.Fatalf("unexpected service totals: %+v", servicesData)
	}
	if len(servicesData.TechnicianRank) != 1 || servicesData.TechnicianRank[0].Name != "Tono Teknisi" ||
		servicesData.TechnicianRank[0].Count != 1 {
		t.Fatalf("unexpected technician rank: %+v", servicesData.TechnicianRank)
	}

	// Brands.
	brands := app.request(t, http.MethodGet, "/api/v1/reports/brands", nil, ownerToken)
	mustStatus(t, brands, http.StatusOK, "brands report")
	brandsData := decodeData[brandsReportData](t, brands)
	if len(brandsData.Items) != 1 || brandsData.Items[0].Brand != "Samsung" ||
		brandsData.Items[0].Count != 1 || brandsData.Items[0].Revenue != 400000 {
		t.Fatalf("unexpected brand rows: %+v", brandsData.Items)
	}

	// Service vs parts.
	vs := app.request(t, http.MethodGet, "/api/v1/reports/service-vs-parts", nil, ownerToken)
	mustStatus(t, vs, http.StatusOK, "service vs parts report")
	vsData := decodeData[serviceVsPartsReportData](t, vs)
	amounts := map[string]int64{}
	qtys := map[string]int64{}
	for _, item := range vsData.Items {
		amounts[item.ItemType] = item.Amount
		qtys[item.ItemType] = item.Qty
	}
	if amounts["service"] != 150000 || amounts["sparepart"] != 250000 || qtys["service"] != 1 || qtys["sparepart"] != 1 {
		t.Fatalf("unexpected service-vs-parts rows: %+v", vsData.Items)
	}

	// Profit: revenue 400k, COGS 100k, expenses 50k -> net 250k.
	profit := app.request(t, http.MethodGet, "/api/v1/reports/profit", nil, ownerToken)
	mustStatus(t, profit, http.StatusOK, "profit report")
	profitData := decodeData[profitReportData](t, profit)
	if profitData.Revenue != 400000 || profitData.Cost != 100000 || profitData.GrossProfit != 300000 {
		t.Fatalf("unexpected profit core: %+v", profitData)
	}
	if profitData.ExpenseTotal != 50000 || profitData.NetProfit != 250000 {
		t.Fatalf("unexpected profit net: %+v", profitData)
	}
	if profitData.InvoiceCount != 1 || profitData.AvgInvoice != 400000 {
		t.Fatalf("unexpected invoice stats: %+v", profitData)
	}

	// Expenses.
	expenses := app.request(t, http.MethodGet, "/api/v1/reports/expenses", nil, ownerToken)
	mustStatus(t, expenses, http.StatusOK, "expenses report")
	expensesData := decodeData[expensesReportData](t, expenses)
	if expensesData.Total != 50000 || len(expensesData.Items) != 1 || expensesData.Items[0].Count != 1 {
		t.Fatalf("unexpected expense rows: %+v", expensesData)
	}

	// Technicians productivity.
	techs := app.request(t, http.MethodGet, "/api/v1/reports/technicians", nil, ownerToken)
	mustStatus(t, techs, http.StatusOK, "technicians report")
	techsData := decodeData[techniciansReportData](t, techs)
	if len(techsData.Items) != 1 {
		t.Fatalf("technician rows = %d, want 1", len(techsData.Items))
	}
	row := techsData.Items[0]
	if row.Name != "Tono Teknisi" || row.AssignedCount != 1 || row.ActiveCount != 1 || row.CompletedCount != 0 {
		t.Fatalf("unexpected technician row: %+v", row)
	}
	if row.RevenueContrib != 400000 {
		t.Fatalf("technician revenue contribution = %d, want 400000", row.RevenueContrib)
	}

	// Stock: one part consumed -> 4 left at 100k each.
	stock := app.request(t, http.MethodGet, "/api/v1/reports/stock", nil, ownerToken)
	mustStatus(t, stock, http.StatusOK, "stock report")
	stockData := decodeData[stockReportData](t, stock)
	if stockData.TotalItems != 1 || stockData.TotalValue != 400000 || stockData.LowStockRows != 0 {
		t.Fatalf("unexpected stock summary: %+v", stockData)
	}
	if stockData.Items[0].Stock != 4 || stockData.Items[0].StockValue != 400000 || stockData.Items[0].IsLow {
		t.Fatalf("unexpected stock row: %+v", stockData.Items[0])
	}

	// Inventory valuation mirrors the stock report for profit viewers.
	valuation := app.request(t, http.MethodGet, "/api/v1/reports/inventory-valuation", nil, ownerToken)
	mustStatus(t, valuation, http.StatusOK, "inventory valuation")
	valuationData := decodeData[stockReportData](t, valuation)
	if valuationData.TotalValue != 400000 {
		t.Fatalf("valuation total = %d, want 400000", valuationData.TotalValue)
	}

	// Excel export streams a real xlsx (zip magic bytes).
	xlsx := app.requestBinary(t, http.MethodGet, "/api/v1/reports/revenue?period=daily&format=xlsx", ownerToken)
	if xlsx.Code != http.StatusOK {
		t.Fatalf("revenue xlsx status = %d", xlsx.Code)
	}
	if got := xlsx.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("xlsx content type = %q", got)
	}
	if !bytes.HasPrefix(xlsx.Body.Bytes(), []byte("PK\x03\x04")) {
		t.Fatalf("xlsx body is not a zip container")
	}

	stockXlsx := app.requestBinary(t, http.MethodGet, "/api/v1/reports/stock?format=xlsx", ownerToken)
	if stockXlsx.Code != http.StatusOK {
		t.Fatalf("stock xlsx status = %d", stockXlsx.Code)
	}
	if !bytes.HasPrefix(stockXlsx.Body.Bytes(), []byte("PK\x03\x04")) {
		t.Fatal("stock xlsx body is not a zip container")
	}

	// Invalid period and reversed ranges are rejected.
	assertCode(t, app.request(t, http.MethodGet, "/api/v1/reports/revenue?period=weekly", nil, ownerToken),
		http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	assertCode(t, app.request(t, http.MethodGet,
		"/api/v1/reports/revenue?from="+today+"&to="+yesterday, nil, ownerToken),
		http.StatusUnprocessableEntity, "VALIDATION_ERROR")
}

func TestStage10ReportPermissionGates(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)
	seedStage10Flow(t, app, ownerToken)

	app.createUser(t, ownerToken, "Ani Admin", "admin.rpt@fixphone.local", "admin", []string{app.outletID.String()})
	app.allowLogin(t, "admin.rpt@fixphone.local")
	adminToken := accessToken(t, app.login(t, "admin.rpt@fixphone.local", "Password#123"))

	// Admin keeps the operational reports...
	mustStatus(t, app.request(t, http.MethodGet, "/api/v1/reports/revenue", nil, adminToken),
		http.StatusOK, "admin revenue report")

	// ...but not the profit report or the audit trail.
	assertCode(t, app.request(t, http.MethodGet, "/api/v1/reports/profit", nil, adminToken),
		http.StatusForbidden, "FORBIDDEN")
	assertCode(t, app.request(t, http.MethodGet, "/api/v1/reports/inventory-valuation", nil, adminToken),
		http.StatusForbidden, "FORBIDDEN")
	assertCode(t, app.request(t, http.MethodGet, "/api/v1/audit-logs", nil, adminToken),
		http.StatusForbidden, "FORBIDDEN")

	// Super admin is platform-scoped and never sees tenant reports.
	app.allowLogin(t, "superadmin@fixphone.local")
	superToken := accessToken(t, app.login(t, "superadmin@fixphone.local", "Superadmin#2026"))
	assertCode(t, app.request(t, http.MethodGet, "/api/v1/reports/revenue", nil, superToken),
		http.StatusForbidden, "FORBIDDEN")
}

func TestStage10SettingsAuditAndManualBackup(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)
	seedStage10Flow(t, app, ownerToken)

	// Settings are readable and partially updatable.
	initial := app.request(t, http.MethodGet, "/api/v1/settings", nil, ownerToken)
	mustStatus(t, initial, http.StatusOK, "get settings")
	initialData := decodeData[settingsData](t, initial)
	if initialData.StoreName != "Toko Demo" || initialData.TicketPrefix != "FP" {
		t.Fatalf("unexpected initial settings: %+v", initialData)
	}

	patched := app.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"store_name":                   "Toko Demo Utama",
		"invoice_footer":               "Terima kasih atas kepercayaan Anda",
		"receipt_paper_width":          80,
		"default_service_warranty_days": 14,
		"wa_provider":                  "noop",
	}, ownerToken)
	mustStatus(t, patched, http.StatusOK, "patch settings")
	patchedData := decodeData[settingsData](t, patched)
	if patchedData.StoreName != "Toko Demo Utama" || patchedData.ReceiptPaperWidth != 80 ||
		patchedData.DefaultServiceWarrantyDays != 14 || patchedData.WAProvider != "noop" {
		t.Fatalf("unexpected patched settings: %+v", patchedData)
	}
	if patchedData.InvoiceFooter == nil || *patchedData.InvoiceFooter != "Terima kasih atas kepercayaan Anda" {
		t.Fatalf("invoice footer not stored: %+v", patchedData.InvoiceFooter)
	}

	// Invalid values are rejected before any write.
	assertCode(t, app.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"timezone": "Mars/Olympus",
	}, ownerToken), http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	assertCode(t, app.request(t, http.MethodPatch, "/api/v1/settings", map[string]any{
		"ticket_prefix": "  ",
	}, ownerToken), http.StatusUnprocessableEntity, "VALIDATION_ERROR")

	// The updated store identity shows up on new invoices.
	created := createServiceOrder(t, app, ownerToken, map[string]any{
		"customer":  map[string]any{"phone": "081300007777", "name": "Joko Invoice"},
		"device":    map[string]any{"brand": "Vivo", "model": "Y22"},
		"complaint": "Charging rusak",
	})
	addServiceLine(t, app, ownerToken, created["id"].(string), 50000)
	invoice := invoiceForOrder(t, app, ownerToken, created["id"].(string))
	issueInvoice(t, app, ownerToken, invoice.ID)
	if invoice.Store.Name != "Toko Demo Utama" {
		t.Fatalf("invoice store name = %q, want updated name", invoice.Store.Name)
	}

	// Audit trail captures the settings change and is filterable.
	auditByModule := app.request(t, http.MethodGet, "/api/v1/audit-logs?module=settings&per_page=5", nil, ownerToken)
	mustStatus(t, auditByModule, http.StatusOK, "audit logs by module")
	auditData := decodeData[auditListData](t, auditByModule)
	if len(auditData.Items) == 0 {
		t.Fatal("settings audit entries missing")
	}
	found := false
	for _, item := range auditData.Items {
		if item.Action == "settings_update" && item.EntityType == "business_settings" && item.ActorName == "Budi Owner" {
			found = true
		}
	}
	if !found {
		t.Fatalf("settings_update audit entry missing: %+v", auditData.Items)
	}

	auditByEntity := app.request(t, http.MethodGet, "/api/v1/audit-logs?entity_type=business_settings", nil, ownerToken)
	mustStatus(t, auditByEntity, http.StatusOK, "audit logs by entity")
	if len(decodeData[auditListData](t, auditByEntity).Items) == 0 {
		t.Fatal("entity filter returned no rows")
	}

	modules := app.request(t, http.MethodGet, "/api/v1/audit-logs/modules", nil, ownerToken)
	mustStatus(t, modules, http.StatusOK, "audit modules")
	var moduleList []string
	if err := json.Unmarshal(modules.body.Data, &moduleList); err != nil {
		t.Fatalf("decode audit modules: %v", err)
	}
	hasSettings := false
	for _, module := range moduleList {
		if module == "settings" {
			hasSettings = true
		}
	}
	if !hasSettings {
		t.Fatalf("settings module missing from audit modules: %v", moduleList)
	}

	// Manual backup: create, list, download.
	createdBackup := app.request(t, http.MethodPost, "/api/v1/settings/backups", nil, ownerToken)
	mustStatus(t, createdBackup, http.StatusCreated, "create backup")
	backup := decodeData[backupData](t, createdBackup)
	if backup.Status != "completed" || backup.TableCount < 20 || backup.RowCount == 0 || backup.SizeBytes == 0 {
		t.Fatalf("unexpected backup metadata: %+v", backup)
	}
	if backup.CreatedBy == nil || *backup.CreatedBy != "Budi Owner" {
		t.Fatalf("backup creator missing: %+v", backup.CreatedBy)
	}

	list := app.request(t, http.MethodGet, "/api/v1/settings/backups", nil, ownerToken)
	mustStatus(t, list, http.StatusOK, "list backups")
	var backups []backupData
	if err := json.Unmarshal(list.body.Data, &backups); err != nil {
		t.Fatalf("decode backups: %v", err)
	}
	if len(backups) != 1 || backups[0].ID != backup.ID {
		t.Fatalf("backup list = %+v", backups)
	}

	download := app.requestBinary(t, http.MethodGet, "/api/v1/settings/backups/"+backup.ID+"/download", ownerToken)
	if download.Code != http.StatusOK {
		t.Fatalf("download backup status = %d: %s", download.Code, download.Body.String())
	}
	if got := download.Header().Get("Content-Type"); got != "application/gzip" {
		t.Fatalf("backup content type = %q", got)
	}
	body := download.Body.Bytes()
	if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
		t.Fatal("downloaded backup is not gzip")
	}
	if int64(len(body)) != backup.SizeBytes {
		t.Fatalf("downloaded size = %d, metadata says %d", len(body), backup.SizeBytes)
	}

	// Unknown or foreign backups are rejected.
	assertCode(t, app.request(t, http.MethodGet, "/api/v1/settings/backups/"+uuid.New().String()+"/download", nil, ownerToken),
		http.StatusNotFound, "NOT_FOUND")
}
