export interface WATemplates {
  status_update: string;
  ready_for_pickup: string;
  invoice: string;
  reminder: string;
  warranty: string;
  claim_received: string;
}

export interface Settings {
  store_name: string;
  legal_name: string | null;
  logo_path: string | null;
  address: string | null;
  phone: string | null;
  email: string | null;
  tax_id: string | null;
  timezone: string;
  ticket_prefix: string;
  ticket_format: string;
  invoice_prefix: string;
  invoice_format: string;
  invoice_terms: string | null;
  invoice_footer: string | null;
  default_service_warranty_days: number;
  default_part_warranty_days: number;
  receipt_paper_width: number;
  technician_sees_only_assigned: boolean;
  public_tracking_enabled: boolean;
  wa_provider: string;
  wa_templates: WATemplates;
  updated_at: string;
}

export type WAProvider = 'noop' | 'fonnte' | 'wablas';

/** Form state: nullable text fields become plain strings; empty clears the column. */
export interface SettingsForm {
  store_name: string;
  legal_name: string;
  address: string;
  phone: string;
  email: string;
  tax_id: string;
  timezone: string;
  ticket_prefix: string;
  ticket_format: string;
  invoice_prefix: string;
  invoice_format: string;
  invoice_terms: string;
  invoice_footer: string;
  default_service_warranty_days: number;
  default_part_warranty_days: number;
  receipt_paper_width: number;
  technician_sees_only_assigned: boolean;
  public_tracking_enabled: boolean;
  wa_provider: string;
  wa_templates: WATemplates;
}

export interface Backup {
  id: string;
  file_name: string;
  size_bytes: number;
  status: string;
  table_count: number;
  row_count: number;
  notes: string | null;
  created_at: string;
  created_by: string | null;
}
