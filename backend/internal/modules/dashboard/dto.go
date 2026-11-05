package dashboard

import "time"

// Summary is the headline dashboard payload. Money values are integer rupiah.
type Summary struct {
	ServicesToday    int64         `json:"services_today"`
	WaitingCount     int64         `json:"waiting_count"`
	InProgressCount  int64         `json:"in_progress_count"`
	CompletedCount   int64         `json:"completed_count"`
	NotPickedUpCount int64         `json:"not_picked_up_count"`
	RevenueToday     int64         `json:"revenue_today"`
	RevenueMonth     int64         `json:"revenue_month"`
	TotalCustomers   int64         `json:"total_customers"`
	Alerts           SummaryAlerts `json:"alerts"`
}

// SummaryAlerts contains the counts shown as badges on the dashboard.
type SummaryAlerts struct {
	Overdue       int64 `json:"overdue"`
	NotPickedUp3d int64 `json:"not_picked_up_3d"`
	LowStock      int64 `json:"low_stock"`
}

// ChartPoint is one day in the revenue/service chart.
type ChartPoint struct {
	Date              string `json:"date"`
	Label             string `json:"label"`
	Revenue           int64  `json:"revenue"`
	ServicesReceived  int64  `json:"services_received"`
	ServicesCompleted int64  `json:"services_completed"`
}

// Charts contains an inclusive daily series.
type Charts struct {
	From string       `json:"from"`
	To   string       `json:"to"`
	Days []ChartPoint `json:"days"`
}

// AlertItem is an overdue or uncollected service order.
type AlertItem struct {
	ID              string     `json:"id"`
	TicketNo        string     `json:"ticket_no"`
	CustomerName    string     `json:"customer_name"`
	Device          string     `json:"device"`
	Status          string     `json:"status"`
	EstimatedDoneAt *time.Time `json:"estimated_done_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	Days            int64      `json:"days"`
}

// StockAlert is a sparepart at or below its minimum stock.
type StockAlert struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Stock    int64  `json:"stock"`
	MinStock int64  `json:"min_stock"`
}

// Alerts is the action list returned by GET /dashboard/alerts.
type Alerts struct {
	Overdue     []AlertItem  `json:"overdue"`
	NotPickedUp []AlertItem  `json:"not_picked_up"`
	LowStock    []StockAlert `json:"low_stock"`
}

// Technician is one technician's activity in a date range.
type Technician struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CompletedCount int64  `json:"completed_count"`
	ActiveCount    int64  `json:"active_count"`
	AvgMinutes     int64  `json:"avg_minutes"`
}
