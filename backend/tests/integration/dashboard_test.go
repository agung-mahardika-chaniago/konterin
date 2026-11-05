package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type dashboardSummary struct {
	ServicesToday    int64 `json:"services_today"`
	WaitingCount     int64 `json:"waiting_count"`
	InProgressCount  int64 `json:"in_progress_count"`
	CompletedCount   int64 `json:"completed_count"`
	NotPickedUpCount int64 `json:"not_picked_up_count"`
	RevenueToday     int64 `json:"revenue_today"`
	RevenueMonth     int64 `json:"revenue_month"`
	TotalCustomers   int64 `json:"total_customers"`
	Alerts           struct {
		Overdue       int64 `json:"overdue"`
		NotPickedUp3d int64 `json:"not_picked_up_3d"`
		LowStock      int64 `json:"low_stock"`
	} `json:"alerts"`
}

type dashboardCharts struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days []struct {
		Date              string `json:"date"`
		Label             string `json:"label"`
		Revenue           int64  `json:"revenue"`
		ServicesReceived  int64  `json:"services_received"`
		ServicesCompleted int64  `json:"services_completed"`
	} `json:"days"`
}

type dashboardAlerts struct {
	Overdue []struct {
		TicketNo     string `json:"ticket_no"`
		CustomerName string `json:"customer_name"`
		Days         int64  `json:"days"`
	} `json:"overdue"`
	NotPickedUp []struct {
		TicketNo string `json:"ticket_no"`
		Days     int64  `json:"days"`
	} `json:"not_picked_up"`
	LowStock []struct {
		SKU      string `json:"sku"`
		Stock    int64  `json:"stock"`
		MinStock int64  `json:"min_stock"`
	} `json:"low_stock"`
}

type dashboardTechnicians []struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CompletedCount int64  `json:"completed_count"`
	ActiveCount    int64  `json:"active_count"`
	AvgMinutes     int64  `json:"avg_minutes"`
}

func TestDashboardSummaryChartsAlertsAndTechnicians(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	tech := app.createUser(t, ownerToken, "Teknisi Dashboard", "tech-dashboard@fixphone.local", "technician", []string{app.outletID.String()})
	app.allowLogin(t, "tech-dashboard@fixphone.local")
	techID, err := uuid.Parse(tech["id"].(string))
	if err != nil {
		t.Fatalf("parse technician id: %v", err)
	}

	now := time.Now().UTC()
	customerID := seedDashboardCustomer(t, app, "Siti Pelanggan", "081200000001")

	insertServiceOrder(t, app, customerID, &techID, "DASH-001", "in_progress", now, nil, nil, pointerTime(now.Add(time.Hour)), pointerTime(now.Add(-20*time.Minute)))
	insertServiceOrder(t, app, customerID, nil, "DASH-002", "waiting_part", now, nil, nil, pointerTime(now.Add(-2*time.Hour)), nil)
	insertServiceOrder(t, app, customerID, nil, "DASH-003", "ready_for_pickup", now.AddDate(0, 0, -4), pointerTime(now.AddDate(0, 0, -4)), nil, nil, nil)
	completedID := insertServiceOrder(t, app, customerID, &techID, "DASH-004", "completed", now, pointerTime(now), nil, nil, pointerTime(now.Add(-90*time.Minute)))
	insertServiceOrder(t, app, customerID, nil, "DASH-005", "picked_up", now.AddDate(0, 0, -1), pointerTime(now.AddDate(0, 0, -1)), pointerTime(now.AddDate(0, 0, -1)), nil, nil)

	invoiceID := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO invoices (id, tenant_id, outlet_id, service_order_id, invoice_no, customer_id, status,
		                      subtotal_service, total_amount, paid_amount)
		VALUES (?, ?, ?, ?, ?, ?, 'partial', 500000, 500000, 500000)`,
		invoiceID, app.tenantID, app.outletID, completedID, "INV-DASH-001", customerID).Error; err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
	if err := app.db.Exec(`
		INSERT INTO payments (id, tenant_id, outlet_id, invoice_id, payment_no, method, amount, paid_at, status)
		VALUES (?, ?, ?, ?, ?, 'cash', 500000, ?, 'posted')`,
		uuid.New(), app.tenantID, app.outletID, invoiceID, "PAY-DASH-001", now).Error; err != nil {
		t.Fatalf("insert payment: %v", err)
	}

	if err := app.db.Exec(`
		INSERT INTO spareparts (id, tenant_id, sku, name, stock, min_stock, is_active)
		VALUES (?, ?, 'LCD-DASH', 'LCD Demo', 1, 3, true),
		       (?, ?, 'BAT-DASH', 'Baterai Demo', 10, 2, true)`,
		uuid.New(), app.tenantID, uuid.New(), app.tenantID).Error; err != nil {
		t.Fatalf("insert spareparts: %v", err)
	}

	summaryResult := app.request(t, http.MethodGet, "/api/v1/dashboard/summary", nil, ownerToken)
	if summaryResult.response.Code != http.StatusOK {
		t.Fatalf("summary status = %d: %s", summaryResult.response.Code, summaryResult.response.Body.String())
	}
	var summary dashboardSummary
	if err := json.Unmarshal(summaryResult.body.Data, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.ServicesToday != 3 || summary.WaitingCount != 1 || summary.InProgressCount != 1 ||
		summary.CompletedCount != 1 || summary.NotPickedUpCount != 2 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.RevenueToday != 500000 || summary.RevenueMonth != 500000 || summary.TotalCustomers != 1 {
		t.Fatalf("unexpected summary money/customer: %+v", summary)
	}
	if summary.Alerts.Overdue != 1 || summary.Alerts.NotPickedUp3d != 1 || summary.Alerts.LowStock != 1 {
		t.Fatalf("unexpected summary alerts: %+v", summary.Alerts)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	today := time.Now().In(loc)
	from := today.AddDate(0, 0, -6).Format("2006-01-02")
	to := today.Format("2006-01-02")
	chartsResult := app.request(t, http.MethodGet, "/api/v1/dashboard/charts?from="+from+"&to="+to, nil, ownerToken)
	if chartsResult.response.Code != http.StatusOK {
		t.Fatalf("charts status = %d: %s", chartsResult.response.Code, chartsResult.response.Body.String())
	}
	var charts dashboardCharts
	if err := json.Unmarshal(chartsResult.body.Data, &charts); err != nil {
		t.Fatalf("decode charts: %v", err)
	}
	if len(charts.Days) != 7 || charts.From != from || charts.To != to {
		t.Fatalf("unexpected chart range: %+v", charts)
	}
	last := charts.Days[len(charts.Days)-1]
	if last.Date != to || last.Revenue != 500000 || last.ServicesReceived != 3 || last.ServicesCompleted != 1 {
		t.Fatalf("unexpected today chart point: %+v", last)
	}

	monthFrom := today.AddDate(0, 0, -29).Format("2006-01-02")
	monthChartsResult := app.request(t, http.MethodGet, "/api/v1/dashboard/charts?from="+monthFrom+"&to="+to, nil, ownerToken)
	if monthChartsResult.response.Code != http.StatusOK {
		t.Fatalf("30-day charts status = %d: %s", monthChartsResult.response.Code, monthChartsResult.response.Body.String())
	}
	var monthCharts dashboardCharts
	if err := json.Unmarshal(monthChartsResult.body.Data, &monthCharts); err != nil {
		t.Fatalf("decode 30-day charts: %v", err)
	}
	if len(monthCharts.Days) != 30 || monthCharts.Days[0].Date != monthFrom || monthCharts.Days[29].Date != to {
		t.Fatalf("unexpected 30-day chart range: from=%s to=%s points=%d", monthCharts.From, monthCharts.To, len(monthCharts.Days))
	}

	alertsResult := app.request(t, http.MethodGet, "/api/v1/dashboard/alerts", nil, ownerToken)
	if alertsResult.response.Code != http.StatusOK {
		t.Fatalf("alerts status = %d: %s", alertsResult.response.Code, alertsResult.response.Body.String())
	}
	var alerts dashboardAlerts
	if err := json.Unmarshal(alertsResult.body.Data, &alerts); err != nil {
		t.Fatalf("decode alerts: %v", err)
	}
	if len(alerts.Overdue) != 1 || alerts.Overdue[0].TicketNo != "DASH-002" {
		t.Fatalf("unexpected overdue alerts: %+v", alerts.Overdue)
	}
	// Every uncollected device is listed (DASH-004 completed today is also
	// waiting for pickup), ordered by completion date ascending.
	if len(alerts.NotPickedUp) != 2 || alerts.NotPickedUp[0].TicketNo != "DASH-003" ||
		alerts.NotPickedUp[0].Days != 4 || alerts.NotPickedUp[1].TicketNo != "DASH-004" {
		t.Fatalf("unexpected pickup alerts: %+v", alerts.NotPickedUp)
	}
	if len(alerts.LowStock) != 1 || alerts.LowStock[0].SKU != "LCD-DASH" || alerts.LowStock[0].Stock != 1 {
		t.Fatalf("unexpected stock alerts: %+v", alerts.LowStock)
	}

	techResult := app.request(t, http.MethodGet, "/api/v1/dashboard/technicians?from="+from+"&to="+to, nil, ownerToken)
	if techResult.response.Code != http.StatusOK {
		t.Fatalf("technicians status = %d: %s", techResult.response.Code, techResult.response.Body.String())
	}
	var technicians dashboardTechnicians
	if err := json.Unmarshal(techResult.body.Data, &technicians); err != nil {
		t.Fatalf("decode technicians: %v", err)
	}
	if len(technicians) != 1 {
		t.Fatalf("technicians = %+v", technicians)
	}
	activity := technicians[0]
	if activity.ID != techID.String() || activity.CompletedCount != 1 || activity.ActiveCount != 1 || activity.AvgMinutes != 90 {
		t.Fatalf("unexpected technician activity: %+v", activity)
	}

	techLogin := app.login(t, "tech-dashboard@fixphone.local", "Password#123")
	if techLogin.response.Code != http.StatusOK {
		t.Fatalf("technician login = %d: %s", techLogin.response.Code, techLogin.response.Body.String())
	}
	if result := app.request(t, http.MethodGet, "/api/v1/dashboard/summary", nil, accessToken(t, techLogin)); result.response.Code != http.StatusOK {
		t.Fatalf("technician dashboard = %d: %s", result.response.Code, result.response.Body.String())
	}

	if result := app.request(t, http.MethodGet, "/api/v1/dashboard/summary?outlet_id="+uuid.NewString(), nil, ownerToken); result.response.Code != http.StatusForbidden {
		t.Fatalf("foreign outlet summary = %d, want 403", result.response.Code)
	}
	if result := app.request(t, http.MethodGet, "/api/v1/dashboard/summary?date=not-a-date", nil, ownerToken); result.response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid date = %d, want 422", result.response.Code)
	}
	if result := app.request(t, http.MethodGet, "/api/v1/dashboard/charts?from=2026-02-10&to=2026-02-01", nil, ownerToken); result.response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reversed range = %d, want 422", result.response.Code)
	}
}

func TestDashboardOutletFilterIsolatesData(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	outletB := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO outlets (id, tenant_id, code, name, timezone, is_active)
		VALUES (?, ?, 'CBG', 'Cabang B', 'Asia/Jakarta', true)`,
		outletB, app.tenantID).Error; err != nil {
		t.Fatalf("insert outlet B: %v", err)
	}
	customerB := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO customers (id, tenant_id, name, phone, phone_normalized)
		VALUES (?, ?, 'Pelanggan Cabang', '081200000099', '081200000099')`,
		customerB, app.tenantID).Error; err != nil {
		t.Fatalf("insert customer B: %v", err)
	}
	if err := app.db.Exec(`
		INSERT INTO service_orders (id, tenant_id, outlet_id, ticket_no, customer_id, status,
		                            device_brand, device_model, complaint, received_at)
		VALUES (?, ?, ?, 'CBG-001', ?, 'in_progress', 'Samsung', 'A54', 'Layar rusak', now())`,
		uuid.New(), app.tenantID, outletB, customerB).Error; err != nil {
		t.Fatalf("insert service B: %v", err)
	}

	base := app.request(t, http.MethodGet, "/api/v1/dashboard/summary", nil, ownerToken)
	if base.response.Code != http.StatusOK {
		t.Fatalf("base summary = %d: %s", base.response.Code, base.response.Body.String())
	}
	var baseSummary dashboardSummary
	if err := json.Unmarshal(base.body.Data, &baseSummary); err != nil {
		t.Fatalf("decode base summary: %v", err)
	}
	// Without X-Outlet-Id the middleware selects the first accessible outlet
	// ("Cabang B" sorts before "Pusat"), so branch B is included.
	if baseSummary.ServicesToday != 1 {
		t.Fatalf("default outlet services today = %d, want 1", baseSummary.ServicesToday)
	}

	pusat := app.request(t, http.MethodGet, "/api/v1/dashboard/summary?outlet_id="+app.outletID.String(), nil, ownerToken)
	if pusat.response.Code != http.StatusOK {
		t.Fatalf("pusat summary = %d: %s", pusat.response.Code, pusat.response.Body.String())
	}
	var pusatSummary dashboardSummary
	if err := json.Unmarshal(pusat.body.Data, &pusatSummary); err != nil {
		t.Fatalf("decode pusat summary: %v", err)
	}
	if pusatSummary.ServicesToday != 0 || pusatSummary.InProgressCount != 0 {
		t.Fatalf("pusat outlet leaked branch data: %+v", pusatSummary)
	}

	branch := app.request(t, http.MethodGet, "/api/v1/dashboard/summary?outlet_id="+outletB.String(), nil, ownerToken)
	if branch.response.Code != http.StatusOK {
		t.Fatalf("branch summary = %d: %s", branch.response.Code, branch.response.Body.String())
	}
	var branchSummary dashboardSummary
	if err := json.Unmarshal(branch.body.Data, &branchSummary); err != nil {
		t.Fatalf("decode branch summary: %v", err)
	}
	if branchSummary.ServicesToday != 1 || branchSummary.InProgressCount != 1 {
		t.Fatalf("unexpected branch summary: %+v", branchSummary)
	}
	if branchSummary.TotalCustomers != 1 {
		t.Fatalf("tenant-wide customer count = %d, want 1", branchSummary.TotalCustomers)
	}

	headerResult := dashboardRequestWithOutlet(t, app, ownerToken, outletB)
	if headerResult.response.Code != http.StatusOK {
		t.Fatalf("header outlet summary = %d: %s", headerResult.response.Code, headerResult.response.Body.String())
	}
	var headerSummary dashboardSummary
	if err := json.Unmarshal(headerResult.body.Data, &headerSummary); err != nil {
		t.Fatalf("decode header summary: %v", err)
	}
	if headerSummary.ServicesToday != 1 || headerSummary.InProgressCount != 1 {
		t.Fatalf("unexpected header outlet summary: %+v", headerSummary)
	}
}

// requestWithHeader is used for cases where the browser-style outlet header
// must be exercised instead of the query parameter.
func dashboardRequestWithOutlet(t *testing.T, app *authTestApp, token string, outletID uuid.UUID) *apiResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Outlet-Id", outletID.String())
	rec := httptest.NewRecorder()
	app.engine.ServeHTTP(rec, req)
	result := &apiResult{response: rec, cookies: rec.Result().Cookies()}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result.body); err != nil {
			t.Fatalf("decode header summary: %v", err)
		}
	}
	return result
}

func seedDashboardCustomer(t *testing.T, app *authTestApp, name, phone string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO customers (id, tenant_id, name, phone, phone_normalized)
		VALUES (?, ?, ?, ?, ?)`, id, app.tenantID, name, phone, phone).Error; err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	return id
}

func insertServiceOrder(t *testing.T, app *authTestApp, customerID uuid.UUID, technicianID *uuid.UUID, ticket, status string, receivedAt time.Time, completedAt, pickedUpAt, estimatedDoneAt, startedAt *time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := app.db.Exec(`
		INSERT INTO service_orders (
			id, tenant_id, outlet_id, ticket_no, customer_id, assigned_technician_id, status,
			device_brand, device_model, complaint, received_at, completed_at, picked_up_at,
			estimated_done_at, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'Samsung', 'Galaxy A54', 'Layar rusak', ?, ?, ?, ?, ?)`,
		id, app.tenantID, app.outletID, ticket, customerID, technicianID, status,
		receivedAt, completedAt, pickedUpAt, estimatedDoneAt, startedAt).Error; err != nil {
		t.Fatalf("insert service order %s: %v", ticket, err)
	}
	return id
}

func pointerTime(value time.Time) *time.Time {
	return &value
}
