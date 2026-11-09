import { api } from '@/lib/api-client';
import type { PublicTracking } from './types';

/** Looks up a ticket using the ticket number and the last four phone digits. */
export async function trackByTicket(ticket: string, phoneLast4: string): Promise<PublicTracking> {
  const params = new URLSearchParams({
    ticket: ticket.trim(),
    phone_last4: phoneLast4.trim(),
  });
  return api.get<PublicTracking>(`/public/track?${params.toString()}`);
}

/** Looks up a ticket directly through the QR/deep-link token. */
export async function trackByToken(token: string): Promise<PublicTracking> {
  return api.get<PublicTracking>(`/public/track/${encodeURIComponent(token.trim())}`);
}
