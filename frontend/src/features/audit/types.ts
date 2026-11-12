export interface AuditLogRow {
  id: number;
  actor_name: string;
  action: string;
  module: string;
  entity_type: string;
  entity_id?: string;
  before_data?: unknown;
  after_data?: unknown;
  created_at: string;
}

export interface AuditFilter {
  page?: number;
  per_page?: number;
  user_id?: string;
  module?: string;
  action?: string;
  entity_type?: string;
  from?: string;
  to?: string;
}

export interface AuditListData {
  items: AuditLogRow[];
}
