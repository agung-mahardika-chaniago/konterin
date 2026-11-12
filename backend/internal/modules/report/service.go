package report

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultTimezone = "Asia/Jakarta"
	maxRangeDays    = 370
)

// Service builds report aggregates from live data.
type Service struct {
	db *gorm.DB
}

// NewService builds the report module.
func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// reportScope resolves the tenant/outlet filter for a request.
type reportScope struct {
	TenantID  uuid.UUID
	OutletID  *uuid.UUID
	Loc       *time.Location
	From      time.Time
	To        time.Time
	FromKey   string
	ToKey     string
}

// resolveScope validates the requested outlet, resolves the outlet timezone,
// and clamps the requested period. from/to are interpreted in the outlet
// timezone so day boundaries match what the outlet staff see.
func (s *Service) resolveScope(ctx context.Context, filter Filter, defaultDays int) (*reportScope, error) {
	if filter.TenantID == "" {
		return nil, apperror.Forbidden("FORBIDDEN", "Laporan hanya tersedia untuk pengguna tenant.")
	}
	tenantID, err := uuid.Parse(filter.TenantID)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"tenant": "Tenant tidak valid."})
	}

	var outletID *uuid.UUID
	if strings.TrimSpace(filter.OutletID) != "" {
		id, parseErr := uuid.Parse(strings.TrimSpace(filter.OutletID))
		if parseErr != nil {
			return nil, apperror.Validation(map[string]string{"outlet_id": "ID outlet tidak valid."})
		}
		outletID = &id
	}

	loc, err := s.location(ctx, tenantID, outletID)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	toDay, err := parseDay(filter.To, loc)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(filter.To) == "" {
		toDay = today
	}
	fromDay, err := parseDay(filter.From, loc)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(filter.From) == "" {
		fromDay = toDay.AddDate(0, 0, -defaultDays)
	}
	if fromDay.After(toDay) {
		return nil, apperror.Validation(map[string]string{"from": "Tanggal awal tidak boleh setelah tanggal akhir."})
	}
	if int(toDay.Sub(fromDay).Hours()/24)+1 > maxRangeDays {
		return nil, apperror.Validation(map[string]string{"from": "Rentang tanggal terlalu panjang (maksimal 1 tahun)."})
	}

	return &reportScope{
		TenantID: tenantID, OutletID: outletID, Loc: loc,
		From: fromDay, To: toDay,
		FromKey: fromDay.Format("2006-01-02"), ToKey: toDay.Format("2006-01-02"),
	}, nil
}

func (s *Service) location(ctx context.Context, tenantID uuid.UUID, outletID *uuid.UUID) (*time.Location, error) {
	var timezone string
	var err error
	if outletID != nil {
		err = s.db.WithContext(ctx).Raw(`
			SELECT timezone FROM outlets WHERE id = ? AND tenant_id = ?`,
			*outletID, tenantID).Scan(&timezone).Error
	} else {
		err = s.db.WithContext(ctx).Raw(`
			SELECT timezone FROM outlets WHERE tenant_id = ?
			ORDER BY is_active DESC, name LIMIT 1`, tenantID).Scan(&timezone).Error
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = defaultTimezone
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return loc, nil
}

func parseDay(value string, loc *time.Location) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), loc)
	if err != nil {
		return time.Time{}, apperror.Validation(map[string]string{"date": "Format tanggal harus YYYY-MM-DD."})
	}
	return day, nil
}

func (sc *reportScope) outletClause(alias string) (string, []any) {
	if sc.OutletID == nil {
		return "", nil
	}
	return " AND " + alias + ".outlet_id = ?", []any{*sc.OutletID}
}

func (sc *reportScope) periodExpr(column string, period Period) string {
	tz := sc.Loc.String()
	switch period {
	case PeriodYearly:
		return fmt.Sprintf("(date_trunc('year', (%s AT TIME ZONE %s)))", column, quoteLiteral(tz))
	case PeriodMonthly:
		return fmt.Sprintf("(date_trunc('month', (%s AT TIME ZONE %s)))", column, quoteLiteral(tz))
	default:
		return fmt.Sprintf("((%s AT TIME ZONE %s)::date)", column, quoteLiteral(tz))
	}
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func periodLabelFor(period Period, bucket time.Time) string {
	switch period {
	case PeriodYearly:
		return bucket.Format("2006")
	case PeriodMonthly:
		return bucket.Format("2006-01")
	default:
		return bucket.Format("2006-01-02")
	}
}

// Revenue aggregates issued (non-draft, non-void) invoices by period bucket.
func (s *Service) Revenue(ctx context.Context, filter Filter, period Period) (*RevenueReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	if period != PeriodDaily && period != PeriodMonthly && period != PeriodYearly {
		return nil, apperror.Validation(map[string]string{"period": "Periode harus daily, monthly, atau yearly."})
	}

	outletSQL, outletArgs := sc.outletClause("i")
	bucketExpr := sc.periodExpr("i.issued_at", period)

	type row struct {
		Bucket          time.Time `gorm:"column:bucket"`
		InvoiceCount    int64     `gorm:"column:invoice_count"`
		ServiceCount    int64     `gorm:"column:service_count"`
		ServiceAmount   int64     `gorm:"column:service_amount"`
		PartAmount      int64     `gorm:"column:part_amount"`
		DiscountAmount  int64     `gorm:"column:discount_amount"`
		TotalRevenue    int64     `gorm:"column:total_revenue"`
		PaidAmount      int64     `gorm:"column:paid_amount"`
		UnpaidAmount    int64     `gorm:"column:unpaid_amount"`
	}
	args := append([]any{sc.TenantID, sc.From, sc.To.AddDate(0, 0, 1)}, outletArgs...)
	var rows []row
	query := `
		SELECT ` + bucketExpr + ` AS bucket,
			count(*)::bigint AS invoice_count,
			count(*) FILTER (WHERE i.service_order_id IS NOT NULL)::bigint AS service_count,
			COALESCE(sum(i.subtotal_service), 0)::bigint AS service_amount,
			COALESCE(sum(i.subtotal_part), 0)::bigint AS part_amount,
			COALESCE(sum(i.discount_amount), 0)::bigint AS discount_amount,
			COALESCE(sum(i.total_amount), 0)::bigint AS total_revenue,
			COALESCE(sum(i.paid_amount), 0)::bigint AS paid_amount,
			COALESCE(sum(i.total_amount - i.paid_amount), 0)::bigint AS unpaid_amount
		FROM invoices i
		WHERE i.tenant_id = ? AND i.status <> 'draft' AND i.status <> 'void'
		  AND i.issued_at >= ? AND i.issued_at < ?` + outletSQL + `
		GROUP BY 1 ORDER BY 1`
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]RevenuePoint, 0, len(rows))
	var total RevenuePoint
	for _, r := range rows {
		items = append(items, RevenuePoint{
			Label: periodLabelFor(period, r.Bucket), InvoiceCount: r.InvoiceCount,
			ServiceCount: r.ServiceCount, ServiceAmount: r.ServiceAmount,
			PartAmount: r.PartAmount, DiscountAmount: r.DiscountAmount,
			TotalRevenue: r.TotalRevenue, PaidAmount: r.PaidAmount,
			UnpaidAmount: r.UnpaidAmount,
		})
		total.InvoiceCount += r.InvoiceCount
		total.ServiceCount += r.ServiceCount
		total.ServiceAmount += r.ServiceAmount
		total.PartAmount += r.PartAmount
		total.DiscountAmount += r.DiscountAmount
		total.TotalRevenue += r.TotalRevenue
		total.PaidAmount += r.PaidAmount
		total.UnpaidAmount += r.UnpaidAmount
	}
	return &RevenueReport{Period: period, From: sc.FromKey, To: sc.ToKey, Items: items, Total: total}, nil
}

// Services summarises service orders received in the period.
func (s *Service) Services(ctx context.Context, filter Filter) (*ServicesReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	outletSQL, outletArgs := sc.outletClause("so")

	var totals struct {
		Total     int64   `gorm:"column:total"`
		Cancelled int64   `gorm:"column:cancelled"`
		Completed int64   `gorm:"column:completed"`
		InProgress int64  `gorm:"column:in_progress"`
		AvgHours  float64 `gorm:"column:avg_hours"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*)::bigint AS total,
			count(*) FILTER (WHERE so.status = 'cancelled')::bigint AS cancelled,
			count(*) FILTER (WHERE so.completed_at IS NOT NULL AND so.status <> 'cancelled')::bigint AS completed,
			count(*) FILTER (WHERE so.status NOT IN ('cancelled','picked_up') AND so.completed_at IS NULL)::bigint AS in_progress,
			COALESCE(avg(extract(epoch FROM (so.completed_at - so.started_at)) / 3600.0)
				FILTER (WHERE so.completed_at IS NOT NULL AND so.started_at IS NOT NULL), 0) AS avg_hours
		FROM service_orders so
		WHERE so.tenant_id = ? AND so.received_at >= ? AND so.received_at < ?`+outletSQL,
		append([]any{sc.TenantID, sc.From, sc.To.AddDate(0, 0, 1)}, outletArgs...)...).Scan(&totals).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	var statusRows []struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT so.status, count(*)::bigint AS count
		FROM service_orders so
		WHERE so.tenant_id = ? AND so.received_at >= ? AND so.received_at < ?`+outletSQL+`
		GROUP BY so.status ORDER BY count DESC`,
		append([]any{sc.TenantID, sc.From, sc.To.AddDate(0, 0, 1)}, outletArgs...)...).Scan(&statusRows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	statuses := make([]ServiceStatusRow, 0, len(statusRows))
	for _, r := range statusRows {
		statuses = append(statuses, ServiceStatusRow{Status: r.Status, Count: r.Count})
	}

	var techRows []struct {
		Name  string `gorm:"column:name"`
		Count int64  `gorm:"column:count"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT u.name, count(*)::bigint AS count
		FROM service_orders so JOIN users u ON u.id = so.assigned_technician_id
		WHERE so.tenant_id = ? AND so.received_at >= ? AND so.received_at < ?
		  AND so.status <> 'cancelled'`+outletSQL+`
		GROUP BY u.name ORDER BY count DESC`,
		append([]any{sc.TenantID, sc.From, sc.To.AddDate(0, 0, 1)}, outletArgs...)...).Scan(&techRows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	techs := make([]ServiceTechnicianRow, 0, len(techRows))
	for _, r := range techRows {
		techs = append(techs, ServiceTechnicianRow{Name: r.Name, Count: r.Count})
	}

	return &ServicesReport{
		From: sc.FromKey, To: sc.ToKey,
		Total: totals.Total, Cancelled: totals.Cancelled,
		Completed: totals.Completed, InProgress: totals.InProgress,
		AvgHours: totals.AvgHours, StatusBreakdown: statuses, TechnicianRank: techs,
	}, nil
}

// Brands aggregates service orders and their invoice totals per device brand.
func (s *Service) Brands(ctx context.Context, filter Filter) (*BrandsReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	outletSQL, outletArgs := sc.outletClause("so")

	var rows []struct {
		Brand   string `gorm:"column:brand"`
		Count   int64  `gorm:"column:count"`
		Revenue int64  `gorm:"column:revenue"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT so.device_brand AS brand, count(*)::bigint AS count,
			COALESCE(sum(i.total_amount) FILTER (WHERE i.status NOT IN ('draft','void')), 0)::bigint AS revenue
		FROM service_orders so
		LEFT JOIN invoices i ON i.service_order_id = so.id
		WHERE so.tenant_id = ? AND so.received_at >= ? AND so.received_at < ?`+outletSQL+`
		GROUP BY so.device_brand
		ORDER BY count DESC`,
		append([]any{sc.TenantID, sc.From, sc.To.AddDate(0, 0, 1)}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]BrandRow, 0, len(rows))
	for _, r := range rows {
		items = append(items, BrandRow{Brand: r.Brand, Count: r.Count, Revenue: r.Revenue})
	}
	return &BrandsReport{From: sc.FromKey, To: sc.ToKey, Items: items}, nil
}

// ServiceVsParts compares service labour vs sparepart revenue on issued invoices.
func (s *Service) ServiceVsParts(ctx context.Context, filter Filter) (*ServiceVsPartsReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	outletSQL, outletArgs := sc.outletClause("i")

	var rows []struct {
		ItemType string `gorm:"column:item_type"`
		Qty      int64  `gorm:"column:qty"`
		Amount   int64  `gorm:"column:amount"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT ii.item_type, COALESCE(sum(ii.qty), 0)::bigint AS qty,
			COALESCE(sum(ii.subtotal), 0)::bigint AS amount
		FROM invoice_items ii
		JOIN invoices i ON i.id = ii.invoice_id
		WHERE i.tenant_id = ? AND i.status NOT IN ('draft','void')
		  AND i.issued_at >= ? AND i.issued_at < ?`+outletSQL+`
		GROUP BY ii.item_type`,
		append([]any{sc.TenantID, sc.From, sc.To.AddDate(0, 0, 1)}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]ItemTypeRow, 0, len(rows))
	for _, r := range rows {
		items = append(items, ItemTypeRow{ItemType: r.ItemType, Qty: r.Qty, Amount: r.Amount})
	}
	return &ServiceVsPartsReport{From: sc.FromKey, To: sc.ToKey, Items: items}, nil
}

// Profit computes revenue, COGS, gross profit, expenses, and net profit.
func (s *Service) Profit(ctx context.Context, filter Filter) (*ProfitReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	end := sc.To.AddDate(0, 0, 1)
	outletSQL, outletArgs := sc.outletClause("i")

	var revenue struct {
		Revenue      int64 `gorm:"column:revenue"`
		Cost         int64 `gorm:"column:cost"`
		InvoiceCount int64 `gorm:"column:invoice_count"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(sum(i.total_amount), 0)::bigint AS revenue,
			COALESCE((SELECT sum(ii.cost_price * ii.qty)
				FROM invoice_items ii
				WHERE ii.invoice_id = ANY(array_agg(i.id))
				  AND ii.item_type IN ('sparepart','custom')), 0)::bigint AS cost,
			count(*)::bigint AS invoice_count
		FROM invoices i
		WHERE i.tenant_id = ? AND i.status NOT IN ('draft','void')
		  AND i.issued_at >= ? AND i.issued_at < ?`+outletSQL,
		append([]any{sc.TenantID, sc.From, end}, outletArgs...)...).Scan(&revenue).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	var expenseTotal int64
	expenseOutletSQL, expenseOutletArgs := sc.outletClause("e")
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(sum(e.amount), 0)::bigint FROM expenses e
		WHERE e.tenant_id = ? AND e.deleted_at IS NULL
		  AND e.expense_date >= ? AND e.expense_date <= ?`+expenseOutletSQL,
		append([]any{sc.TenantID, sc.From, sc.To}, expenseOutletArgs...)...).Scan(&expenseTotal).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	var avgInvoice int64
	if revenue.InvoiceCount > 0 {
		avgInvoice = revenue.Revenue / revenue.InvoiceCount
	}
	return &ProfitReport{
		From: sc.FromKey, To: sc.ToKey,
		Revenue: revenue.Revenue, Cost: revenue.Cost,
		GrossProfit: revenue.Revenue - revenue.Cost,
		ExpenseTotal: expenseTotal,
		NetProfit:    revenue.Revenue - revenue.Cost - expenseTotal,
		InvoiceCount: revenue.InvoiceCount, AvgInvoice: avgInvoice,
	}, nil
}

// Expenses aggregates posted expenses by category.
func (s *Service) Expenses(ctx context.Context, filter Filter) (*ExpensesReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	outletSQL, outletArgs := sc.outletClause("e")

	var rows []struct {
		Category string `gorm:"column:category"`
		Count    int64  `gorm:"column:count"`
		Amount   int64  `gorm:"column:amount"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT ec.name AS category, count(*)::bigint AS count,
			COALESCE(sum(e.amount), 0)::bigint AS amount
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.tenant_id = ? AND e.deleted_at IS NULL
		  AND e.expense_date >= ? AND e.expense_date <= ?`+outletSQL+`
		GROUP BY ec.name ORDER BY amount DESC`,
		append([]any{sc.TenantID, sc.From, sc.To}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]ExpenseCategoryRow, 0, len(rows))
	var total int64
	for _, r := range rows {
		items = append(items, ExpenseCategoryRow{Category: r.Category, Count: r.Count, Amount: r.Amount})
		total += r.Amount
	}
	return &ExpensesReport{From: sc.FromKey, To: sc.ToKey, Items: items, Total: total}, nil
}

// Technicians measures technician productivity in the period.
func (s *Service) Technicians(ctx context.Context, filter Filter) (*TechniciansReport, error) {
	sc, err := s.resolveScope(ctx, filter, 29)
	if err != nil {
		return nil, err
	}
	outletSQL, outletArgs := sc.outletClause("so")
	end := sc.To.AddDate(0, 0, 1)

	var rows []struct {
		Name           string  `gorm:"column:name"`
		AssignedCount  int64   `gorm:"column:assigned_count"`
		CompletedCount int64   `gorm:"column:completed_count"`
		ActiveCount    int64   `gorm:"column:active_count"`
		AvgRepairHours float64 `gorm:"column:avg_repair_hours"`
		RevenueContrib int64   `gorm:"column:revenue_contribution"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT u.name,
			count(*)::bigint AS assigned_count,
			count(*) FILTER (WHERE so.completed_at IS NOT NULL AND so.status <> 'cancelled')::bigint AS completed_count,
			count(*) FILTER (WHERE so.completed_at IS NULL AND so.status NOT IN ('cancelled','picked_up'))::bigint AS active_count,
			COALESCE(avg(extract(epoch FROM (so.completed_at - so.started_at)) / 3600.0)
				FILTER (WHERE so.completed_at IS NOT NULL AND so.started_at IS NOT NULL), 0) AS avg_repair_hours,
			COALESCE(sum(i.total_amount) FILTER (WHERE i.status NOT IN ('draft','void')), 0)::bigint AS revenue_contribution
		FROM service_orders so
		JOIN users u ON u.id = so.assigned_technician_id
		LEFT JOIN invoices i ON i.service_order_id = so.id
		WHERE so.tenant_id = ? AND so.received_at >= ? AND so.received_at < ?`+outletSQL+`
		GROUP BY u.name ORDER BY completed_count DESC`,
		append([]any{sc.TenantID, sc.From, end}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]TechnicianRow, 0, len(rows))
	for _, r := range rows {
		items = append(items, TechnicianRow{
			Name: r.Name, AssignedCount: r.AssignedCount, CompletedCount: r.CompletedCount,
			ActiveCount: r.ActiveCount, AvgRepairHours: r.AvgRepairHours, RevenueContrib: r.RevenueContrib,
		})
	}
	return &TechniciansReport{From: sc.FromKey, To: sc.ToKey, Items: items}, nil
}

// Stock lists sparepart stock with valuation (tenant-wide, outlet neutral).
func (s *Service) Stock(ctx context.Context, filter Filter) (*StockReport, error) {
	sc, err := s.resolveScope(ctx, filter, 0)
	if err != nil {
		return nil, err
	}

	var rows []struct {
		SKU        string `gorm:"column:sku"`
		Name       string `gorm:"column:name"`
		Category   string `gorm:"column:category"`
		Stock      int64  `gorm:"column:stock"`
		MinStock   int64  `gorm:"column:min_stock"`
		CostPrice  int64  `gorm:"column:cost_price"`
		StockValue int64  `gorm:"column:stock_value"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT sp.sku, sp.name, COALESCE(c.name, 'Tanpa kategori') AS category,
			sp.stock, sp.min_stock, sp.cost_price,
			(sp.stock * sp.cost_price)::bigint AS stock_value
		FROM spareparts sp
		LEFT JOIN sparepart_categories c ON c.id = sp.category_id
		WHERE sp.tenant_id = ? AND sp.deleted_at IS NULL
		ORDER BY stock_value DESC, sp.name`,
		sc.TenantID).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]StockRow, 0, len(rows))
	var totalValue int64
	var lowCount int64
	for _, r := range rows {
		isLow := r.Stock <= r.MinStock
		if isLow {
			lowCount++
		}
		items = append(items, StockRow{
			SKU: r.SKU, Name: r.Name, Category: r.Category, Stock: r.Stock,
			MinStock: r.MinStock, CostPrice: r.CostPrice, StockValue: r.StockValue, IsLow: isLow,
		})
		totalValue += r.StockValue
	}
	return &StockReport{Items: items, TotalValue: totalValue, TotalItems: int64(len(items)), LowStockRows: lowCount}, nil
}

// InventoryValuation summarises stock value for the profit-aware viewer.
func (s *Service) InventoryValuation(ctx context.Context, filter Filter) (*StockReport, error) {
	return s.Stock(ctx, filter)
}

// storeNames resolves store and outlet display names for export headers.
func (s *Service) storeNames(ctx context.Context, filter Filter) (store string, outlet string, err error) {
	tenantID, err := uuid.Parse(filter.TenantID)
	if err != nil {
		return "", "", apperror.Validation(map[string]string{"tenant": "Tenant tidak valid."})
	}
	store, outlet = "FixPhone", ""
	var row struct {
		StoreName  string  `gorm:"column:store_name"`
		OutletName *string `gorm:"column:outlet_name"`
	}
	query := `
		SELECT bs.store_name, o.name AS outlet_name
		FROM business_settings bs
		LEFT JOIN outlets o ON o.id = bs.outlet_id
		WHERE bs.tenant_id = ?`
	args := []any{tenantID}
	if strings.TrimSpace(filter.OutletID) != "" {
		query += ` AND bs.outlet_id = ?`
		args = append(args, filter.OutletID)
	} else {
		query += ` AND bs.outlet_id IS NULL`
	}
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&row).Error; err != nil {
		return "", "", apperror.Internal(err)
	}
	if strings.TrimSpace(row.StoreName) != "" {
		store = row.StoreName
	}
	if row.OutletName != nil && strings.TrimSpace(*row.OutletName) != "" {
		outlet = *row.OutletName
	}
	return store, outlet, nil
}
