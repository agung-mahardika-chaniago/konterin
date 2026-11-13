package integration_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// TestStockNeverOversellsUnderConcurrency reserves the last units of a
// sparepart from many service orders at once. Row locks must serialise the
// deduction so the stock never goes negative and the surplus requests are
// rejected with INSUFFICIENT_STOCK.
func TestStockNeverOversellsUnderConcurrency(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	const initialStock = 3
	const workers = 8

	part := createSparepart(t, app, ownerToken, map[string]any{
		"sku": "RACE-STOK-01", "name": "Konektor Charger",
		"stock": initialStock, "min_stock": 0, "cost_price": 10000, "sell_price": 25000,
	})

	orderIDs := make([]string, workers)
	for i := 0; i < workers; i++ {
		created := createServiceOrder(t, app, ownerToken, map[string]any{
			"customer":  map[string]any{"name": fmt.Sprintf("Pelanggan Stok %02d", i), "phone": fmt.Sprintf("08170000%04d", i)},
			"device":    map[string]any{"brand": "Xiaomi", "model": "Redmi 12"},
			"complaint": "Port charger rusak",
		})
		orderIDs[i] = created["id"].(string)
	}

	results := make([]*apiResult, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = app.request(t, http.MethodPost, "/api/v1/service-orders/"+orderIDs[i]+"/items",
				map[string]any{
					"item_type": "sparepart", "sparepart_id": part.ID, "qty": 1,
				}, ownerToken)
		}(i)
	}
	wg.Wait()

	success, insufficient := 0, 0
	for i, result := range results {
		switch result.response.Code {
		case http.StatusCreated:
			success++
		case http.StatusConflict:
			if result.body.Error.Code != "INSUFFICIENT_STOCK" {
				t.Fatalf("worker %d conflict code = %q: %s", i, result.body.Error.Code, result.response.Body.String())
			}
			insufficient++
		default:
			t.Fatalf("worker %d status = %d: %s", i, result.response.Code, result.response.Body.String())
		}
	}
	if success != initialStock || insufficient != workers-initialStock {
		t.Fatalf("stock race: success=%d insufficient=%d, want %d/%d",
			success, insufficient, initialStock, workers-initialStock)
	}

	var stock int
	if err := app.db.Raw(`SELECT stock FROM spareparts WHERE id = ?`, part.ID).Scan(&stock).Error; err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if stock != 0 {
		t.Fatalf("final stock = %d, want 0", stock)
	}

	var movements int64
	if err := app.db.Raw(`
		SELECT count(*) FROM inventory_movements
		WHERE sparepart_id = ? AND movement_type = 'service_out'`, part.ID).Scan(&movements).Error; err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if movements != initialStock {
		t.Fatalf("service_use movements = %d, want %d", movements, initialStock)
	}
}

// TestPaymentIdempotencyAndOverpaymentUnderConcurrency verifies two payment
// invariants under parallel traffic: one Idempotency-Key creates one payment,
// and distinct concurrent payments can never exceed the invoice total.
func TestPaymentIdempotencyAndOverpaymentUnderConcurrency(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	newInvoice := func(phone string) invoiceData {
		t.Helper()
		created := createServiceOrder(t, app, ownerToken, map[string]any{
			"customer":  map[string]any{"phone": phone, "name": "Pelanggan Bayar"},
			"device":    map[string]any{"brand": "Samsung", "model": "A15"},
			"complaint": "Layar bergaris",
		})
		orderID := created["id"].(string)
		addServiceLine(t, app, ownerToken, orderID, 200000)
		return issueInvoice(t, app, ownerToken, invoiceForOrder(t, app, ownerToken, orderID).ID)
	}

	// 1) Same Idempotency-Key fired in parallel.
	sameKeyInvoice := newInvoice("081711110001")
	const idempotentWorkers = 10
	results := make([]*apiResult, idempotentWorkers)
	var wg sync.WaitGroup
	for i := 0; i < idempotentWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = app.requestWithHeaders(t, http.MethodPost,
				"/api/v1/invoices/"+sameKeyInvoice.ID+"/payments",
				map[string]any{"method": "cash", "amount": 50000},
				ownerToken, map[string]string{"Idempotency-Key": "race-same-key"})
		}(i)
	}
	wg.Wait()

	paymentIDs := make(map[string]struct{}, idempotentWorkers)
	for i, result := range results {
		if result.response.Code != http.StatusCreated {
			t.Fatalf("idempotent worker %d status = %d: %s", i, result.response.Code, result.response.Body.String())
		}
		data := decodeData[paymentMutation](t, result)
		paymentIDs[data.Payment.ID] = struct{}{}
		if data.Invoice.PaidAmount != 50000 || data.Invoice.RemainingAmount != 150000 {
			t.Fatalf("idempotent worker %d invoice state: %+v", i, data.Invoice)
		}
	}
	if len(paymentIDs) != 1 {
		t.Fatalf("idempotency key produced %d payments, want 1", len(paymentIDs))
	}
	var paymentRows int64
	if err := app.db.Raw(`SELECT count(*) FROM payments WHERE invoice_id = ?`, sameKeyInvoice.ID).
		Scan(&paymentRows).Error; err != nil {
		t.Fatalf("count idempotent payments: %v", err)
	}
	if paymentRows != 1 {
		t.Fatalf("payment rows = %d, want 1", paymentRows)
	}

	// 2) Distinct keys, each 60000 on a 200000 invoice: exactly three fit.
	overpayInvoice := newInvoice("081711110002")
	const overpayWorkers = 8
	overpayResults := make([]*apiResult, overpayWorkers)
	for i := 0; i < overpayWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			overpayResults[i] = app.requestWithHeaders(t, http.MethodPost,
				"/api/v1/invoices/"+overpayInvoice.ID+"/payments",
				map[string]any{"method": "cash", "amount": 60000},
				ownerToken, map[string]string{"Idempotency-Key": fmt.Sprintf("race-overpay-%02d", i)})
		}(i)
	}
	wg.Wait()

	posted, overpayment := 0, 0
	for i, result := range overpayResults {
		switch result.response.Code {
		case http.StatusCreated:
			posted++
		case http.StatusConflict:
			if result.body.Error.Code != "OVERPAYMENT" {
				t.Fatalf("overpay worker %d conflict = %q: %s", i, result.body.Error.Code, result.response.Body.String())
			}
			overpayment++
		default:
			t.Fatalf("overpay worker %d status = %d: %s", i, result.response.Code, result.response.Body.String())
		}
	}
	if posted != 3 || overpayment != overpayWorkers-3 {
		t.Fatalf("overpayment race: posted=%d rejected=%d, want 3/%d", posted, overpayment, overpayWorkers-3)
	}

	var paid, remaining int64
	if err := app.db.Raw(`SELECT paid_amount FROM invoices WHERE id = ?`, overpayInvoice.ID).Scan(&paid).Error; err != nil {
		t.Fatalf("read paid amount: %v", err)
	}
	remaining = 200000 - paid
	if paid != 180000 || remaining != 20000 {
		t.Fatalf("paid=%d remaining=%d, want 180000/20000", paid, remaining)
	}
	var postedRows int64
	if err := app.db.Raw(`SELECT count(*) FROM payments WHERE invoice_id = ? AND status = 'posted'`,
		overpayInvoice.ID).Scan(&postedRows).Error; err != nil {
		t.Fatalf("count posted payments: %v", err)
	}
	if postedRows != 3 {
		t.Fatalf("posted payment rows = %d, want 3", postedRows)
	}
}
