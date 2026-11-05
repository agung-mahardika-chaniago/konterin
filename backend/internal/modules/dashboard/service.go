package dashboard

import (
	"context"
	"strings"
	"time"

	"fixphone/backend/internal/pkg/apperror"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultTimezone = "Asia/Jakarta"
	chartMaxDays    = 90
	techMaxDays     = 180
)

// Filter is the resolved tenant/outlet boundary for a dashboard request.
type Filter struct {
	TenantID uuid.UUID
	OutletID *uuid.UUID
}

// Service builds dashboard aggregates from live data.
type Service struct {
	db *gorm.DB
}

// NewService builds a dashboard service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Summary returns today's operational counters and revenue.
func (s *Service) Summary(ctx context.Context, filter Filter, date string) (*Summary, error) {
	loc, err := s.location(ctx, filter)
	if err != nil {
		return nil, err
	}
	day, err := parseDay(date, loc)
	if err != nil {
		return nil, err
	}
	start := day
	end := day.AddDate(0, 0, 1)
	monthStart := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, loc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	out := &Summary{}
	outletSQL, outletArgs := outletClause(filter, "service_orders")

	queries := []struct {
		target *int64
		query  string
		args   []any
	}{
		{
			&out.ServicesToday,
			`SELECT count(*) FROM service_orders
			 WHERE tenant_id = ? AND status <> 'cancelled'
			   AND received_at >= ? AND received_at < ?` + outletSQL,
			append([]any{filter.TenantID, start, end}, outletArgs...),
		},
		{
			&out.WaitingCount,
			`SELECT count(*) FROM service_orders
			 WHERE tenant_id = ? AND status IN ('received','waiting_diagnosis','waiting_approval','waiting_part')` + outletSQL,
			append([]any{filter.TenantID}, outletArgs...),
		},
		{
			&out.InProgressCount,
			`SELECT count(*) FROM service_orders
			 WHERE tenant_id = ? AND status IN ('in_progress','testing')` + outletSQL,
			append([]any{filter.TenantID}, outletArgs...),
		},
		{
			&out.CompletedCount,
			`SELECT count(*) FROM service_orders
			 WHERE tenant_id = ? AND status IN ('completed','ready_for_pickup','picked_up')
			   AND completed_at >= ? AND completed_at < ?` + outletSQL,
			append([]any{filter.TenantID, start, end}, outletArgs...),
		},
		{
			&out.NotPickedUpCount,
			`SELECT count(*) FROM service_orders
			 WHERE tenant_id = ? AND status IN ('completed','ready_for_pickup')
			   AND picked_up_at IS NULL` + outletSQL,
			append([]any{filter.TenantID}, outletArgs...),
		},
		{
			&out.TotalCustomers,
			`SELECT count(*) FROM customers
			 WHERE tenant_id = ? AND deleted_at IS NULL`,
			[]any{filter.TenantID},
		},
	}

	for _, item := range queries {
		if err := s.db.WithContext(ctx).Raw(item.query, item.args...).Scan(item.target).Error; err != nil {
			return nil, apperror.Internal(err)
		}
	}

	paymentOutletSQL, paymentOutletArgs := outletClause(filter, "payments")
	paymentArgs := append([]any{filter.TenantID, start, end}, paymentOutletArgs...)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(sum(amount), 0)::bigint FROM payments
		WHERE tenant_id = ? AND status = 'posted'
		  AND paid_at >= ? AND paid_at < ?`+paymentOutletSQL, paymentArgs...).Scan(&out.RevenueToday).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	monthArgs := append([]any{filter.TenantID, monthStart, monthEnd}, paymentOutletArgs...)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(sum(amount), 0)::bigint FROM payments
		WHERE tenant_id = ? AND status = 'posted'
		  AND paid_at >= ? AND paid_at < ?`+paymentOutletSQL, monthArgs...).Scan(&out.RevenueMonth).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	alerts, err := s.alertCounts(ctx, filter)
	if err != nil {
		return nil, err
	}
	out.Alerts = *alerts
	return out, nil
}

func (s *Service) alertCounts(ctx context.Context, filter Filter) (*SummaryAlerts, error) {
	out := &SummaryAlerts{}
	outletSQL, outletArgs := outletClause(filter, "service_orders")

	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM service_orders
		WHERE tenant_id = ? AND estimated_done_at IS NOT NULL AND estimated_done_at < now()
		  AND status NOT IN ('picked_up','cancelled')`+outletSQL,
		append([]any{filter.TenantID}, outletArgs...)...).Scan(&out.Overdue).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM service_orders
		WHERE tenant_id = ? AND status IN ('completed','ready_for_pickup')
		  AND picked_up_at IS NULL AND completed_at IS NOT NULL
		  AND completed_at < now() - interval '3 days'`+outletSQL,
		append([]any{filter.TenantID}, outletArgs...)...).Scan(&out.NotPickedUp3d).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM spareparts
		WHERE tenant_id = ? AND deleted_at IS NULL AND is_active = true AND stock <= min_stock`,
		filter.TenantID).Scan(&out.LowStock).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	return out, nil
}

// Charts returns an inclusive daily revenue and service series.
func (s *Service) Charts(ctx context.Context, filter Filter, from, to string) (*Charts, error) {
	loc, err := s.location(ctx, filter)
	if err != nil {
		return nil, err
	}
	fromDay, toDay, err := parseRange(from, to, loc, 6, chartMaxDays)
	if err != nil {
		return nil, err
	}
	start := fromDay
	end := toDay.AddDate(0, 0, 1)

	days := make([]ChartPoint, 0, 8)
	index := make(map[string]int, 8)
	for day := fromDay; !day.After(toDay); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		index[key] = len(days)
		days = append(days, ChartPoint{Date: key, Label: day.Format("02 Jan")})
	}

	outletSQL, outletArgs := outletClause(filter, "payments")
	type totalRow struct {
		Day   time.Time
		Total int64
	}
	rows := make([]totalRow, 0, len(days))
	if err := s.db.WithContext(ctx).Raw(`
		SELECT (paid_at AT TIME ZONE ?)::date AS day, COALESCE(sum(amount), 0)::bigint AS total
		FROM payments
		WHERE tenant_id = ? AND status = 'posted' AND paid_at >= ? AND paid_at < ?`+outletSQL+`
		GROUP BY 1`, append([]any{loc.String(), filter.TenantID, start, end}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	for _, row := range rows {
		if i, ok := index[row.Day.Format("2006-01-02")]; ok {
			days[i].Revenue = row.Total
		}
	}

	outletSQL, outletArgs = outletClause(filter, "service_orders")
	rows = rows[:0]
	if err := s.db.WithContext(ctx).Raw(`
		SELECT (received_at AT TIME ZONE ?)::date AS day, count(*)::bigint AS total
		FROM service_orders
		WHERE tenant_id = ? AND status <> 'cancelled'
		  AND received_at >= ? AND received_at < ?`+outletSQL+`
		GROUP BY 1`, append([]any{loc.String(), filter.TenantID, start, end}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	for _, row := range rows {
		if i, ok := index[row.Day.Format("2006-01-02")]; ok {
			days[i].ServicesReceived = row.Total
		}
	}

	rows = rows[:0]
	if err := s.db.WithContext(ctx).Raw(`
		SELECT (completed_at AT TIME ZONE ?)::date AS day, count(*)::bigint AS total
		FROM service_orders
		WHERE tenant_id = ? AND status <> 'cancelled' AND completed_at IS NOT NULL
		  AND completed_at >= ? AND completed_at < ?`+outletSQL+`
		GROUP BY 1`, append([]any{loc.String(), filter.TenantID, start, end}, outletArgs...)...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	for _, row := range rows {
		if i, ok := index[row.Day.Format("2006-01-02")]; ok {
			days[i].ServicesCompleted = row.Total
		}
	}

	return &Charts{
		From: fromDay.Format("2006-01-02"),
		To:   toDay.Format("2006-01-02"),
		Days: days,
	}, nil
}

// Alerts lists overdue orders, uncollected devices, and low stock.
func (s *Service) Alerts(ctx context.Context, filter Filter) (*Alerts, error) {
	out := &Alerts{
		Overdue:     []AlertItem{},
		NotPickedUp: []AlertItem{},
		LowStock:    []StockAlert{},
	}
	outletSQL, outletArgs := outletClause(filter, "so")

	type alertRow struct {
		ID              uuid.UUID
		TicketNo        string
		CustomerName    string
		Device          string
		Status          string
		EstimatedDoneAt *time.Time
		CompletedAt     *time.Time
		Days            int64
	}

	overdue := make([]alertRow, 0, 10)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT so.id, so.ticket_no, c.name AS customer_name,
		       so.device_brand || ' ' || so.device_model AS device,
		       so.status, so.estimated_done_at, so.completed_at,
		       GREATEST(0, floor(extract(epoch FROM (now() - so.estimated_done_at)) / 86400))::bigint AS days
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		WHERE so.tenant_id = ? AND so.estimated_done_at IS NOT NULL
		  AND so.estimated_done_at < now()
		  AND so.status NOT IN ('picked_up','cancelled')`+outletSQL+`
		ORDER BY so.estimated_done_at ASC
		LIMIT 20`, append([]any{filter.TenantID}, outletArgs...)...).Scan(&overdue).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	for _, row := range overdue {
		out.Overdue = append(out.Overdue, AlertItem{
			ID: row.ID.String(), TicketNo: row.TicketNo, CustomerName: row.CustomerName,
			Device: row.Device, Status: row.Status, EstimatedDoneAt: row.EstimatedDoneAt,
			CompletedAt: row.CompletedAt, Days: row.Days,
		})
	}

	outletSQL, outletArgs = outletClause(filter, "so")
	waiting := make([]alertRow, 0, 10)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT so.id, so.ticket_no, c.name AS customer_name,
		       so.device_brand || ' ' || so.device_model AS device,
		       so.status, so.estimated_done_at, so.completed_at,
		       GREATEST(0, floor(extract(epoch FROM (now() - so.completed_at)) / 86400))::bigint AS days
		FROM service_orders so
		JOIN customers c ON c.id = so.customer_id
		WHERE so.tenant_id = ? AND so.status IN ('completed','ready_for_pickup')
		  AND so.picked_up_at IS NULL AND so.completed_at IS NOT NULL`+outletSQL+`
		ORDER BY so.completed_at ASC
		LIMIT 20`, append([]any{filter.TenantID}, outletArgs...)...).Scan(&waiting).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	for _, row := range waiting {
		out.NotPickedUp = append(out.NotPickedUp, AlertItem{
			ID: row.ID.String(), TicketNo: row.TicketNo, CustomerName: row.CustomerName,
			Device: row.Device, Status: row.Status, EstimatedDoneAt: row.EstimatedDoneAt,
			CompletedAt: row.CompletedAt, Days: row.Days,
		})
	}

	type stockRow struct {
		ID       uuid.UUID
		SKU      string
		Name     string
		Stock    int64
		MinStock int64
	}
	stock := make([]stockRow, 0, 10)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, sku, name, stock, min_stock
		FROM spareparts
		WHERE tenant_id = ? AND deleted_at IS NULL AND is_active = true AND stock <= min_stock
		ORDER BY (min_stock - stock) DESC, name
		LIMIT 20`, filter.TenantID).Scan(&stock).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	for _, row := range stock {
		out.LowStock = append(out.LowStock, StockAlert{
			ID: row.ID.String(), SKU: row.SKU, Name: row.Name,
			Stock: row.Stock, MinStock: row.MinStock,
		})
	}
	return out, nil
}

// Technicians returns per-technician activity for the requested range.
func (s *Service) Technicians(ctx context.Context, filter Filter, from, to string) ([]Technician, error) {
	loc, err := s.location(ctx, filter)
	if err != nil {
		return nil, err
	}
	fromDay, toDay, err := parseRange(from, to, loc, 29, techMaxDays)
	if err != nil {
		return nil, err
	}
	start := fromDay
	end := toDay.AddDate(0, 0, 1)

	outletSQL, outletArgs := outletClause(filter, "so")
	args := []any{start, end, start, end}
	args = append(args, outletArgs...)
	args = append(args, filter.TenantID)

	type techRow struct {
		ID             uuid.UUID
		Name           string
		CompletedCount int64
		ActiveCount    int64
		AvgMinutes     int64
	}
	rows := make([]techRow, 0, 8)
	query := `
		SELECT u.id, u.name,
		       count(*) FILTER (
		           WHERE so.status IN ('completed','ready_for_pickup','picked_up')
		             AND so.completed_at >= ? AND so.completed_at < ?
		       )::bigint AS completed_count,
		       count(*) FILTER (
		           WHERE so.status IN ('received','waiting_diagnosis','waiting_approval','waiting_part','in_progress','testing')
		       )::bigint AS active_count,
		       COALESCE(round(avg(
		           extract(epoch FROM (so.completed_at - coalesce(so.started_at, so.received_at))) / 60
		       ) FILTER (WHERE so.completed_at >= ? AND so.completed_at < ?)), 0)::bigint AS avg_minutes
		FROM users u
		JOIN roles r ON r.id = u.role_id AND r.slug = 'technician'
		LEFT JOIN service_orders so
		       ON so.assigned_technician_id = u.id AND so.tenant_id = u.tenant_id` + outletSQL + `
		WHERE u.tenant_id = ? AND u.deleted_at IS NULL
		GROUP BY u.id, u.name
		HAVING count(so.id) > 0
		ORDER BY completed_count DESC, u.name`
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	out := make([]Technician, 0, len(rows))
	for _, row := range rows {
		out = append(out, Technician{
			ID: row.ID.String(), Name: row.Name, CompletedCount: row.CompletedCount,
			ActiveCount: row.ActiveCount, AvgMinutes: row.AvgMinutes,
		})
	}
	return out, nil
}

func (s *Service) location(ctx context.Context, filter Filter) (*time.Location, error) {
	var timezone string
	var err error
	if filter.OutletID != nil {
		err = s.db.WithContext(ctx).Raw(`
			SELECT timezone FROM outlets WHERE id = ? AND tenant_id = ?`,
			*filter.OutletID, filter.TenantID).Scan(&timezone).Error
	} else {
		err = s.db.WithContext(ctx).Raw(`
			SELECT timezone FROM outlets WHERE tenant_id = ?
			ORDER BY is_active DESC, name LIMIT 1`, filter.TenantID).Scan(&timezone).Error
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
		// A tenant with an invalid timezone should still see a usable dashboard.
		loc = time.FixedZone("WIB", 7*3600)
	}
	return loc, nil
}

func parseDay(value string, loc *time.Location) (time.Time, error) {
	now := time.Now().In(loc)
	if strings.TrimSpace(value) == "" {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	}
	day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), loc)
	if err != nil {
		return time.Time{}, apperror.Validation(map[string]string{"date": "Format tanggal harus YYYY-MM-DD."})
	}
	return day, nil
}

func parseRange(from, to string, loc *time.Location, defaultDays, maxDays int) (time.Time, time.Time, error) {
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	toDay, err := parseDay(to, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if strings.TrimSpace(to) == "" {
		toDay = today
	}
	fromDay, err := parseDay(from, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if strings.TrimSpace(from) == "" {
		fromDay = toDay.AddDate(0, 0, -defaultDays)
	}
	if fromDay.After(toDay) {
		return time.Time{}, time.Time{}, apperror.Validation(map[string]string{"from": "Tanggal awal tidak boleh setelah tanggal akhir."})
	}
	if int(toDay.Sub(fromDay).Hours()/24)+1 > maxDays {
		return time.Time{}, time.Time{}, apperror.Validation(map[string]string{"from": "Rentang tanggal terlalu panjang."})
	}
	return fromDay, toDay, nil
}

func outletClause(filter Filter, alias string) (string, []any) {
	if filter.OutletID == nil {
		return "", nil
	}
	return " AND " + alias + ".outlet_id = ?", []any{*filter.OutletID}
}
