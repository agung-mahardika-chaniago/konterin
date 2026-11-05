import type { AuthSession } from '@/types/auth';

/**
 * Access tokens live only in module memory. They are never written to
 * localStorage or sessionStorage, so an XSS payload cannot read a persisted
 * session. The refresh token is an HttpOnly cookie managed by the browser.
 */
let accessToken: string | null = null;
let refreshInFlight: Promise<AuthSession | null> | null = null;
let unauthorizedHandler: (() => void) | null = null;

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';
const SESSION_HINT_KEY = 'fixphone.has_session';

/**
 * A non-sensitive flag that tells the app whether a refresh cookie is likely
 * present. It is not a credential and lets first-time visitors skip a
 * guaranteed 401 probe.
 */
export function hasSessionHint(): boolean {
  try {
    return window.localStorage.getItem(SESSION_HINT_KEY) === '1';
  } catch {
    return false;
  }
}

export function markSessionHint(active: boolean): void {
  try {
    if (active) window.localStorage.setItem(SESSION_HINT_KEY, '1');
    else window.localStorage.removeItem(SESSION_HINT_KEY);
  } catch {
    // Storage can be unavailable; session recovery still works without it.
  }
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function clearAccessToken(): void {
  accessToken = null;
}

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  unauthorizedHandler = handler;
}

export function notifyUnauthorized(): void {
  unauthorizedHandler?.();
}

/**
 * Rotates the refresh cookie. Concurrent callers share one request so a page
 * with many parallel 401s cannot trigger reuse detection against itself.
 */
export async function refreshSession(): Promise<AuthSession | null> {
  if (refreshInFlight) {
    return refreshInFlight;
  }
  if (!hasSessionHint()) {
    accessToken = null;
    return null;
  }

  refreshInFlight = (async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      });
      if (!response.ok) {
        accessToken = null;
        markSessionHint(false);
        return null;
      }
      const payload = await response.json();
      if (!payload?.success || !payload?.data?.access_token) {
        accessToken = null;
        markSessionHint(false);
        return null;
      }
      const session = payload.data as AuthSession;
      accessToken = session.access_token;
      markSessionHint(true);
      return session;
    } catch {
      accessToken = null;
      return null;
    } finally {
      refreshInFlight = null;
    }
  })();

  return refreshInFlight;
}
