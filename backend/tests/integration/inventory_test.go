package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"fixphone/backend/internal/database"

	"github.com/google/uuid"
)

type sparepartData struct {
	ID           string  `json:"id"`
	SKU          string  `json:"sku"`
	Barcode      *string `json:"barcode"`
	Name         string  `json:"name"`
	CategoryID   *string `json:"category_id"`
	CategoryName *string `json:"category_name"`
	SupplierID   *string `json:"supplier_id"`
	Stock        int     `json:"stock"`
	MinStock     int     `json:"min_stock"`
	CostPrice    int64   `json:"cost_price"`
	SellPrice    int64   `json:"sell_price"`
	StockValue   int64   `json:"stock_value"`
	IsActive     bool    `json:"is_active"`
	LowStock     bool    `json:"low_stock"`
}

type movementData struct {
	ID            string  `json:"id"`
	SparepartID   string  `json:"sparepart_id"`
	MovementType  string  `json:"movement_type"`
	Qty           int     `json:"qty"`
	StockBefore   int     `json:"stock_before"`
	StockAfter    int     `json:"stock_after"`
	UnitCost      int64   `json:"unit_cost"`
	ReferenceType string  `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
}

type pageMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type stockAlertData struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Stock    int    `json:"stock"`
	MinStock int    `json:"min_stock"`
	Shortage int    `json:"shortage"`
}

type valuationData struct {
	TotalItems int64 `json:"total_items"`
	TotalStock int64 `json:"total_stock"`
	TotalValue int64 `json:"total_value"`
}

type topUsedData struct {
	SparepartID   string `json:"sparepart_id"`
	UsedQty       int64  `json:"used_qty"`
	MovementCount int64  `json:"movement_count"`
}

func decodeData[T any](t *testing.T, result *apiResult) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(result.body.Data, &out); err != nil {
		t.Fatalf("decode data: %v\n%s", err, result.response.Body.String())
	}
	return out
}

func decodeMeta(t *testing.T, result *apiResult) pageMeta {
	t.Helper()
	var out pageMeta
	if err := json.Unmarshal(result.body.Meta, &out); err != nil {
		t.Fatalf("decode meta: %v\n%s", err, result.response.Body.String())
	}
	return out
}

func mustStatus(t *testing.T, result *apiResult, want int, label string) {
	t.Helper()
	if result.response.Code != want {
		t.Fatalf("%s status = %d, want %d: %s", label, result.response.Code, want, result.response.Body.String())
	}
}

func createSparepart(t *testing.T, app *authTestApp, token string, body map[string]any) sparepartData {
	t.Helper()
	result := app.request(t, http.MethodPost, "/api/v1/spareparts", body, token)
	mustStatus(t, result, http.StatusCreated, "create sparepart")
	return decodeData[sparepartData](t, result)
}

func TestInventoryLifecycle(t *testing.T) {
	app := setupAuthApp(t)
	token, _ := app.loginOwner(t)

	categoryResult := app.request(t, http.MethodPost, "/api/v1/sparepart-categories",
		map[string]any{"name": "LCD"}, token)
	mustStatus(t, categoryResult, http.StatusCreated, "create category")
	category := decodeData[struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}](t, categoryResult)

	supplierResult := app.request(t, http.MethodPost, "/api/v1/suppliers",
		map[string]any{"name": "PT Sumber Part", "phone": "08123456789"}, token)
	mustStatus(t, supplierResult, http.StatusCreated, "create supplier")
	supplier := decodeData[struct {
		ID string `json:"id"`
	}](t, supplierResult)

	part := createSparepart(t, app, token, map[string]any{
		"sku": "LCD-IP11", "barcode": "899123", "name": "LCD iPhone 11",
		"category_id": category.ID, "supplier_id": supplier.ID,
		"stock": 10, "min_stock": 5, "cost_price": 100000, "sell_price": 150000,
		"location": "Rak A1",
	})
	if part.Stock != 10 || part.CostPrice != 100000 || part.StockValue != 1000000 {
		t.Fatalf("created part = %+v", part)
	}
	if part.CategoryName == nil || *part.CategoryName != "LCD" {
		t.Fatalf("category name = %v", part.CategoryName)
	}

	initial := app.request(t, http.MethodGet, "/api/v1/spareparts/"+part.ID+"/movements", nil, token)
	mustStatus(t, initial, http.StatusOK, "initial movements")
	initialItems := decodeData[[]movementData](t, initial)
	if len(initialItems) != 1 || initialItems[0].MovementType != "initial" ||
		initialItems[0].Qty != 10 || initialItems[0].ReferenceType != "opname" {
		t.Fatalf("initial movements = %+v", initialItems)
	}

	list := app.request(t, http.MethodGet, "/api/v1/spareparts?q=LCD", nil, token)
	mustStatus(t, list, http.StatusOK, "list spareparts")
	if meta := decodeMeta(t, list); meta.Total != 1 || meta.Page != 1 {
		t.Fatalf("list meta = %+v", meta)
	}

	barcode := app.request(t, http.MethodGet, "/api/v1/spareparts/barcode/899123", nil, token)
	mustStatus(t, barcode, http.StatusOK, "barcode lookup")
	if found := decodeData[sparepartData](t, barcode); found.ID != part.ID {
		t.Fatalf("barcode found %s, want %s", found.ID, part.ID)
	}

	update := app.request(t, http.MethodPatch, "/api/v1/spareparts/"+part.ID,
		map[string]any{"min_stock": 8, "sell_price": 175000}, token)
	mustStatus(t, update, http.StatusOK, "update sparepart")
	if updated := decodeData[sparepartData](t, update); updated.MinStock != 8 || updated.SellPrice != 175000 {
		t.Fatalf("updated part = %+v", updated)
	}

	categoryUpdate := app.request(t, http.MethodPatch, "/api/v1/sparepart-categories/"+category.ID,
		map[string]any{"name": "Layar"}, token)
	mustStatus(t, categoryUpdate, http.StatusOK, "update category")

	supplierUpdate := app.request(t, http.MethodPatch, "/api/v1/suppliers/"+supplier.ID,
		map[string]any{"contact_name": "Andi"}, token)
	mustStatus(t, supplierUpdate, http.StatusOK, "update supplier")

	purchase := app.request(t, http.MethodPost, "/api/v1/inventory/purchases", map[string]any{
		"supplier_id": supplier.ID, "reference_no": "INV-SUP-001",
		"items": []map[string]any{{"sparepart_id": part.ID, "qty": 5, "unit_cost": 120000}},
	}, token)
	mustStatus(t, purchase, http.StatusCreated, "purchase")
	purchaseData := decodeData[struct {
		ReferenceID string         `json:"reference_id"`
		TotalCost   int64          `json:"total_cost"`
		Items       []movementData `json:"items"`
	}](t, purchase)
	if purchaseData.ReferenceID == "" || purchaseData.TotalCost != 600000 || len(purchaseData.Items) != 1 {
		t.Fatalf("purchase = %+v", purchaseData)
	}
	if item := purchaseData.Items[0]; item.MovementType != "purchase_in" || item.StockBefore != 10 || item.StockAfter != 15 {
		t.Fatalf("purchase movement = %+v", item)
	}

	detail := app.request(t, http.MethodGet, "/api/v1/spareparts/"+part.ID, nil, token)
	mustStatus(t, detail, http.StatusOK, "detail after purchase")
	afterPurchase := decodeData[sparepartData](t, detail)
	if afterPurchase.Stock != 15 || afterPurchase.CostPrice != 120000 || afterPurchase.StockValue != 1800000 {
		t.Fatalf("detail after purchase = %+v", afterPurchase)
	}

	adjustment := app.request(t, http.MethodPost, "/api/v1/inventory/adjustments", map[string]any{
		"sparepart_id": part.ID, "new_stock": 12, "reason": "Hasil opname mingguan",
	}, token)
	mustStatus(t, adjustment, http.StatusCreated, "adjustment")
	adjustmentData := decodeData[movementData](t, adjustment)
	if adjustmentData.MovementType != "adjustment_out" || adjustmentData.Qty != -3 ||
		adjustmentData.StockBefore != 15 || adjustmentData.StockAfter != 12 ||
		adjustmentData.ReferenceType != "opname" {
		t.Fatalf("adjustment movement = %+v", adjustmentData)
	}

	second := createSparepart(t, app, token, map[string]any{
		"sku": "BAT-IP11", "name": "Baterai iPhone 11",
		"stock": 1, "min_stock": 5, "cost_price": 50000, "sell_price": 90000,
	})

	alerts := app.request(t, http.MethodGet, "/api/v1/inventory/alerts", nil, token)
	mustStatus(t, alerts, http.StatusOK, "alerts")
	alertItems := decodeData[[]stockAlertData](t, alerts)
	if len(alertItems) != 1 || alertItems[0].ID != second.ID || alertItems[0].Shortage != 4 {
		t.Fatalf("alerts = %+v", alertItems)
	}

	valuation := app.request(t, http.MethodGet, "/api/v1/inventory/valuation", nil, token)
	mustStatus(t, valuation, http.StatusOK, "valuation")
	valuationData := decodeData[valuationData](t, valuation)
	if valuationData.TotalItems != 2 || valuationData.TotalStock != 13 || valuationData.TotalValue != 1490000 {
		t.Fatalf("valuation = %+v", valuationData)
	}

	now := time.Now().UTC()
	if err := app.db.Exec(`
		INSERT INTO inventory_movements (
			id, tenant_id, outlet_id, sparepart_id, movement_type, qty,
			stock_before, stock_after, unit_cost, reference_type, reference_id,
			notes, created_at
		) VALUES (?, ?, ?, ?, 'service_out', -2, 12, 10, 120000, 'service_order', ?, 'Pemakaian servis', ?)`,
		uuid.New(), app.tenantID, app.outletID, part.ID, uuid.New(), now).Error; err != nil {
		t.Fatalf("insert service movement: %v", err)
	}

	topUsed := app.request(t, http.MethodGet, "/api/v1/inventory/top-used", nil, token)
	mustStatus(t, topUsed, http.StatusOK, "top used")
	topItems := decodeData[[]topUsedData](t, topUsed)
	if len(topItems) != 1 || topItems[0].SparepartID != part.ID || topItems[0].UsedQty != 2 {
		t.Fatalf("top used = %+v", topItems)
	}

	allMovements := app.request(t, http.MethodGet, "/api/v1/inventory/movements", nil, token)
	mustStatus(t, allMovements, http.StatusOK, "all movements")
	if meta := decodeMeta(t, allMovements); meta.Total < 4 {
		t.Fatalf("movement total = %d, want at least 4", meta.Total)
	}

	deleteResult := app.request(t, http.MethodDelete, "/api/v1/spareparts/"+part.ID, nil, token)
	mustStatus(t, deleteResult, http.StatusOK, "delete sparepart")
	gone := app.request(t, http.MethodGet, "/api/v1/spareparts/"+part.ID, nil, token)
	mustStatus(t, gone, http.StatusNotFound, "deleted sparepart detail")
}

func TestInventoryPermissionsAndTenantIsolation(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	part := createSparepart(t, app, ownerToken, map[string]any{
		"sku": "CAM-IP11", "name": "Kamera iPhone 11",
		"stock": 4, "min_stock": 2, "cost_price": 250000, "sell_price": 400000,
	})

	app.createUser(t, ownerToken, "Teknisi Satu", "tech@fixphone.local", "technician",
		[]string{app.outletID.String()})
	app.allowLogin(t, "tech@fixphone.local")
	techToken := accessToken(t, app.login(t, "tech@fixphone.local", "Password#123"))

	techDetail := app.request(t, http.MethodGet, "/api/v1/spareparts/"+part.ID, nil, techToken)
	mustStatus(t, techDetail, http.StatusOK, "technician detail")
	masked := decodeData[sparepartData](t, techDetail)
	if masked.CostPrice != 0 || masked.StockValue != 0 {
		t.Fatalf("technician saw cost fields: %+v", masked)
	}

	techMovements := app.request(t, http.MethodGet, "/api/v1/spareparts/"+part.ID+"/movements", nil, techToken)
	mustStatus(t, techMovements, http.StatusOK, "technician movements")
	for _, movement := range decodeData[[]movementData](t, techMovements) {
		if movement.UnitCost != 0 {
			t.Fatalf("technician saw movement unit_cost: %+v", movement)
		}
	}

	denied := app.request(t, http.MethodPost, "/api/v1/inventory/purchases", map[string]any{
		"items": []map[string]any{{"sparepart_id": part.ID, "qty": 1, "unit_cost": 250000}},
	}, techToken)
	mustStatus(t, denied, http.StatusForbidden, "technician purchase")
	if denied.body.Error.Code != "FORBIDDEN" {
		t.Fatalf("technician purchase code = %s", denied.body.Error.Code)
	}

	techAlerts := app.request(t, http.MethodGet, "/api/v1/inventory/alerts", nil, techToken)
	mustStatus(t, techAlerts, http.StatusOK, "technician alerts")

	techValuation := app.request(t, http.MethodGet, "/api/v1/inventory/valuation", nil, techToken)
	mustStatus(t, techValuation, http.StatusForbidden, "technician valuation")

	if _, err := database.CreateTenant(t.Context(), app.db, database.CreateTenantInput{
		Name: "Toko Kedua", Slug: "toko-kedua", OwnerName: "Siti Owner",
		OwnerEmail: "owner2@fixphone.local", OwnerPassword: "Owner#12345",
		OutletName: "Cabang", OutletCode: "CBG",
	}); err != nil {
		t.Fatalf("create second tenant: %v", err)
	}
	otherToken := accessToken(t, app.login(t, "owner2@fixphone.local", "Owner#12345"))

	otherDetail := app.request(t, http.MethodGet, "/api/v1/spareparts/"+part.ID, nil, otherToken)
	mustStatus(t, otherDetail, http.StatusNotFound, "cross-tenant detail")

	otherList := app.request(t, http.MethodGet, "/api/v1/spareparts", nil, otherToken)
	mustStatus(t, otherList, http.StatusOK, "cross-tenant list")
	if meta := decodeMeta(t, otherList); meta.Total != 0 {
		t.Fatalf("cross-tenant list total = %d, want 0", meta.Total)
	}
}
