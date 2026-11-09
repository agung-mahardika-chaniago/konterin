export interface AppNotification {
  id: string;
  type: string;
  title: string;
  body?: string | null;
  link?: string | null;
  entity_type?: string | null;
  entity_id?: string | null;
  is_read: boolean;
  read_at?: string | null;
  is_broadcast: boolean;
  created_at: string;
}

export interface NotificationFilter {
  unread?: boolean;
  page?: number;
  per_page?: number;
}

export type WhatsAppStatus = 'queued' | 'processing' | 'sent' | 'failed' | 'cancelled';

export interface WhatsAppMessage {
  id: string;
  service_order_id?: string | null;
  ticket_no?: string | null;
  customer_name?: string | null;
  customer_id?: string | null;
  phone: string;
  template_key: string;
  body?: string | null;
  status: WhatsAppStatus;
  attempts: number;
  last_error?: string | null;
  provider: string;
  provider_message_id?: string | null;
  scheduled_at: string;
  sent_at?: string | null;
  created_at: string;
}

export interface OutboxFilter {
  status?: string;
  q?: string;
  page?: number;
  per_page?: number;
}
