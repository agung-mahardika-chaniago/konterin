import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type { AppNotification, NotificationFilter, OutboxFilter, WhatsAppMessage } from './types';

function queryString(filter: object): string {
  const params = new URLSearchParams();
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value));
    }
  });
  const query = params.toString();
  return query ? `?${query}` : '';
}

export async function listNotifications(filter: NotificationFilter = {}): Promise<ApiEnvelope<AppNotification[]>> {
  const params = new URLSearchParams();
  if (filter.unread) params.set('unread', 'true');
  if (filter.page) params.set('page', String(filter.page));
  if (filter.per_page) params.set('per_page', String(filter.per_page));
  const query = params.toString();
  return api.getEnvelope<AppNotification[]>(`/notifications${query ? `?${query}` : ''}`);
}

export async function getUnreadCount(): Promise<number> {
  const result = await api.get<{ count: number }>('/notifications/unread-count');
  return result.count;
}

export async function markNotificationRead(id: string): Promise<void> {
  await api.patch<{ id: string; is_read: boolean }>(`/notifications/${id}/read`);
}

export async function markAllNotificationsRead(): Promise<void> {
  await api.post<{ unread: number }>('/notifications/read-all');
}

export async function listWhatsAppMessages(filter: OutboxFilter = {}): Promise<ApiEnvelope<WhatsAppMessage[]>> {
  return api.getEnvelope<WhatsAppMessage[]>(`/whatsapp-messages${queryString(filter)}`);
}

export async function retryWhatsAppMessage(id: string): Promise<WhatsAppMessage> {
  return api.post<WhatsAppMessage>(`/whatsapp-messages/${id}/retry`);
}
