export type ReportPeriod = 'daily' | 'monthly' | 'yearly';

export interface ReportFilter {
  outlet_id?: string;
  from?: string;
  to?: string;
  period?: ReportPeriod;
}

export interface RevenuePoint {
  label: string;
  invoice_count: number;
  service_count: number;
  service_amount: number;
  part_amount: number;
  discount_amount: number;
  total_revenue: number;
  paid_amount: number;
  unpaid_amount: number;
}

export interface RevenueReport {
  period: ReportPeriod;
  from: string;
  to: string;
  items: RevenuePoint[];
  total: RevenuePoint;
}

export interface ServicesReport {
  from: string;
  to: string;
  total: number;
  cancelled: number;
  completed: number;
  in_progress: number;
  avg_hours: number;
  status_breakdown: { status: string; count: number }[];
  technician_rank: { name: string; count: number }[];
}

export interface BrandsReport {
  from: string;
  to: string;
  items: { brand: string; count: number; revenue: number }[];
}

export interface ServiceVsPartsReport {
  from: string;
  to: string;
  items: { item_type: string; qty: number; amount: number }[];
}

export interface ProfitReport {
  from: string;
  to: string;
  revenue: number;
  cost: number;
  gross_profit: number;
  expense_total: number;
  net_profit: number;
  invoice_count: number;
  avg_invoice: number;
}

export interface ExpensesReport {
  from: string;
  to: string;
  items: { category: string; count: number; amount: number }[];
  total: number;
}

export interface TechniciansReport {
  from: string;
  to: string;
  items: {
    name: string;
    assigned_count: number;
    completed_count: number;
    active_count: number;
    avg_repair_hours: number;
    revenue_contribution: number;
  }[];
}

export interface StockReport {
  items: {
    sku: string;
    name: string;
    category: string;
    stock: number;
    min_stock: number;
    cost_price: number;
    stock_value: number;
    is_low: boolean;
  }[];
  total_value: number;
  total_items: number;
  low_stock_rows: number;
}
