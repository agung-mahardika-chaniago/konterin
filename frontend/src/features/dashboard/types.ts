export interface DashboardSummary {
  services_today: number;
  waiting_count: number;
  in_progress_count: number;
  completed_count: number;
  not_picked_up_count: number;
  revenue_today: number;
  revenue_month: number;
  total_customers: number;
  alerts: {
    overdue: number;
    not_picked_up_3d: number;
    low_stock: number;
  };
}

export interface DashboardChartDay {
  date: string;
  label: string;
  revenue: number;
  services_received: number;
  services_completed: number;
}

export interface DashboardCharts {
  from: string;
  to: string;
  days: DashboardChartDay[];
}

export interface DashboardAlertItem {
  id: string;
  ticket_no: string;
  customer_name: string;
  device: string;
  status: string;
  estimated_done_at?: string | null;
  completed_at?: string | null;
  days: number;
}

export interface DashboardStockAlert {
  id: string;
  sku: string;
  name: string;
  stock: number;
  min_stock: number;
}

export interface DashboardAlerts {
  overdue: DashboardAlertItem[];
  not_picked_up: DashboardAlertItem[];
  low_stock: DashboardStockAlert[];
}

export interface DashboardTechnician {
  id: string;
  name: string;
  completed_count: number;
  active_count: number;
  avg_minutes: number;
}

export type DashboardRange = 7 | 30;

export interface DashboardData {
  summary: DashboardSummary;
  charts: DashboardCharts;
  alerts: DashboardAlerts;
  technicians: DashboardTechnician[];
}
