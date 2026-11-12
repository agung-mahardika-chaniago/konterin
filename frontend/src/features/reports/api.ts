import { api, apiDownload, saveBlob } from '@/lib/api-client';
import type { DownloadedFile } from '@/lib/api-client';
import type {
  BrandsReport,
  ExpensesReport,
  ProfitReport,
  ReportFilter,
  RevenueReport,
  ServiceVsPartsReport,
  ServicesReport,
  StockReport,
  TechniciansReport,
} from './types';

export type ReportTab =
  | 'revenue'
  | 'services'
  | 'brands'
  | 'service-vs-parts'
  | 'profit'
  | 'expenses'
  | 'technicians'
  | 'stock'
  | 'valuation';

function reportQuery(filter: ReportFilter, extra: Record<string, string> = {}): string {
  const params = new URLSearchParams();
  Object.entries({ ...filter, ...extra }).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value));
    }
  });
  const query = params.toString();
  return query ? `?${query}` : '';
}

export async function getRevenueReport(filter: ReportFilter): Promise<RevenueReport> {
  return api.get<RevenueReport>(`/reports/revenue${reportQuery(filter)}`);
}

export async function getServicesReport(filter: ReportFilter): Promise<ServicesReport> {
  return api.get<ServicesReport>(`/reports/services${reportQuery(filter)}`);
}

export async function getBrandsReport(filter: ReportFilter): Promise<BrandsReport> {
  return api.get<BrandsReport>(`/reports/brands${reportQuery(filter)}`);
}

export async function getServiceVsPartsReport(filter: ReportFilter): Promise<ServiceVsPartsReport> {
  return api.get<ServiceVsPartsReport>(`/reports/service-vs-parts${reportQuery(filter)}`);
}

export async function getProfitReport(filter: ReportFilter): Promise<ProfitReport> {
  return api.get<ProfitReport>(`/reports/profit${reportQuery(filter)}`);
}

export async function getExpensesReport(filter: ReportFilter): Promise<ExpensesReport> {
  return api.get<ExpensesReport>(`/reports/expenses${reportQuery(filter)}`);
}

export async function getTechniciansReport(filter: ReportFilter): Promise<TechniciansReport> {
  return api.get<TechniciansReport>(`/reports/technicians${reportQuery(filter)}`);
}

export async function getStockReport(filter: ReportFilter): Promise<StockReport> {
  return api.get<StockReport>(`/reports/stock${reportQuery(filter)}`);
}

export async function getInventoryValuation(filter: ReportFilter): Promise<StockReport> {
  return api.get<StockReport>(`/reports/inventory-valuation${reportQuery(filter)}`);
}

/** Report path per tab; used by both the JSON fetch and the xlsx export. */
export const REPORT_PATH: Record<ReportTab, string> = {
  revenue: '/reports/revenue',
  services: '/reports/services',
  brands: '/reports/brands',
  'service-vs-parts': '/reports/service-vs-parts',
  profit: '/reports/profit',
  expenses: '/reports/expenses',
  technicians: '/reports/technicians',
  stock: '/reports/stock',
  valuation: '/reports/inventory-valuation',
};

/** Downloads the xlsx export of one report and saves it locally. */
export async function exportReportXlsx(tab: ReportTab, filter: ReportFilter): Promise<DownloadedFile> {
  const file = await apiDownload(`${REPORT_PATH[tab]}${reportQuery(filter, { format: 'xlsx' })}`);
  saveBlob(file.blob, file.fileName);
  return file;
}
