// Package report builds Stage 10 business reports.
package report

import (
	"encoding/json"
	"time"
)

// Filter is the resolved tenant/outlet boundary plus period for a report.
type Filter struct {
	TenantID string
	OutletID string
	From     string // YYYY-MM-DD inclusive, outlet timezone
	To       string // YYYY-MM-DD inclusive, outlet timezone
}

// Period identifies the grouping requested by the client.
type Period string

const (
	PeriodDaily   Period = "daily"
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
)

// RevenuePoint is one aggregated bucket of the revenue report.
type RevenuePoint struct {
	Label          string `json:"label"`
	InvoiceCount   int64  `json:"invoice_count"`
	ServiceCount   int64  `json:"service_count"`
	ServiceAmount  int64  `json:"service_amount"`
	PartAmount     int64  `json:"part_amount"`
	DiscountAmount int64  `json:"discount_amount"`
	TotalRevenue   int64  `json:"total_revenue"`
	PaidAmount     int64  `json:"paid_amount"`
	UnpaidAmount   int64  `json:"unpaid_amount"`
}

// RevenueReport is the revenue aggregation over a period range.
type RevenueReport struct {
	Period Period        `json:"period"`
	From   string        `json:"from"`
	To     string        `json:"to"`
	Items  []RevenuePoint `json:"items"`
	Total  RevenuePoint   `json:"total"`
}

// ServiceStatusRow counts service orders per status.
type ServiceStatusRow struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// ServiceTechnicianRow counts completed orders per technician.
type ServiceTechnicianRow struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// ServicesReport summarises service orders in the period.
type ServicesReport struct {
	From           string                 `json:"from"`
	To             string                 `json:"to"`
	Total          int64                  `json:"total"`
	Cancelled      int64                  `json:"cancelled"`
	Completed      int64                  `json:"completed"`
	InProgress     int64                  `json:"in_progress"`
	AvgHours       float64                `json:"avg_hours"`
	StatusBreakdown []ServiceStatusRow     `json:"status_breakdown"`
	TechnicianRank []ServiceTechnicianRow `json:"technician_rank"`
}

// BrandRow aggregates service orders by device brand.
type BrandRow struct {
	Brand  string `json:"brand"`
	Count  int64  `json:"count"`
	Revenue int64 `json:"revenue"`
}

// BrandsReport lists device brand distribution.
type BrandsReport struct {
	From  string    `json:"from"`
	To    string    `json:"to"`
	Items []BrandRow `json:"items"`
}

// ItemTypeRow aggregates revenue by item type (service vs sparepart).
type ItemTypeRow struct {
	ItemType string `json:"item_type"`
	Qty      int64  `json:"qty"`
	Amount   int64  `json:"amount"`
}

// ServiceVsPartsReport compares service and sparepart revenue.
type ServiceVsPartsReport struct {
	From  string        `json:"from"`
	To    string        `json:"to"`
	Items []ItemTypeRow `json:"items"`
}

// ProfitReport is the gross/net profit over the period.
type ProfitReport struct {
	From             string `json:"from"`
	To               string `json:"to"`
	Revenue          int64  `json:"revenue"`
	Cost             int64  `json:"cost"`
	GrossProfit      int64  `json:"gross_profit"`
	ExpenseTotal     int64  `json:"expense_total"`
	NetProfit        int64  `json:"net_profit"`
	InvoiceCount     int64  `json:"invoice_count"`
	AvgInvoice       int64  `json:"avg_invoice"`
}

// ExpenseCategoryRow aggregates expenses by category.
type ExpenseCategoryRow struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
	Amount   int64  `json:"amount"`
}

// ExpensesReport aggregates expenses in the period.
type ExpensesReport struct {
	From  string                `json:"from"`
	To    string                `json:"to"`
	Items []ExpenseCategoryRow  `json:"items"`
	Total int64                 `json:"total"`
}

// TechnicianRow measures technician productivity.
type TechnicianRow struct {
	Name            string  `json:"name"`
	AssignedCount   int64   `json:"assigned_count"`
	CompletedCount   int64   `json:"completed_count"`
	ActiveCount     int64   `json:"active_count"`
	AvgRepairHours  float64 `json:"avg_repair_hours"`
	RevenueContrib  int64   `json:"revenue_contribution"`
}

// TechniciansReport lists technician productivity.
type TechniciansReport struct {
	From  string          `json:"from"`
	To    string          `json:"to"`
	Items []TechnicianRow `json:"items"`
}

// StockRow is one sparepart stock line.
type StockRow struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Stock      int64  `json:"stock"`
	MinStock   int64  `json:"min_stock"`
	CostPrice  int64  `json:"cost_price"`
	StockValue int64  `json:"stock_value"`
	IsLow      bool   `json:"is_low"`
}

// StockReport lists sparepart stock with valuation.
type StockReport struct {
	Items        []StockRow `json:"items"`
	TotalValue   int64     `json:"total_value"`
	TotalItems   int64     `json:"total_items"`
	LowStockRows int64     `json:"low_stock_rows"`
}

// ExportMeta carries the metadata used on every export file.
type ExportMeta struct {
	Report      string `json:"report"`
	StoreName   string `json:"store_name"`
	OutletName  string `json:"outlet_name,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	GeneratedBy string `json:"generated_by"`
	GeneratedAt string `json:"generated_at"`
}

// AuditList is the paginated audit log response.
type AuditList struct {
	Items []AuditRow `json:"items"`
}

// AuditRow is one audit log entry.
type AuditRow struct {
	ID         int64           `json:"id"`
	ActorName  string          `json:"actor_name"`
	Action     string          `json:"action"`
	Module     string          `json:"module"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id,omitempty"`
	Before     json.RawMessage `json:"before_data,omitempty"`
	After      json.RawMessage `json:"after_data,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
