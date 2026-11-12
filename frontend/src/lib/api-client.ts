import { getAccessToken, notifyUnauthorized, refreshSession } from '@/lib/auth';
import { getActiveOutletId } from '@/lib/outlet';
import type { ApiEnvelope, ApiErrorBody } from '@/types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details?: ApiErrorBody['details'];
  readonly requestId?: string;

  constructor(status: number, error: ApiErrorBody, requestId?: string) {
    super(error.message || 'Terjadi kesalahan.');
    this.name = 'ApiError';
    this.status = status;
    this.code = error.code;
    this.details = error.details;
    this.requestId = requestId;
  }

  fieldErrors(): Record<string, string> {
    return this.details?.fields ?? {};
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

async function requestRaw(path: string, options: RequestOptions = {}): Promise<Response> {
  const headers = new Headers(options.headers);
  const token = getAccessToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  const outletId = getActiveOutletId();
  if (outletId) {
    headers.set('X-Outlet-Id', outletId);
  }
  let body = options.body;
  if (body !== undefined && !(body instanceof FormData) && typeof body !== 'string') {
    headers.set('Content-Type', 'application/json');
    body = JSON.stringify(body);
  }

  return fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
    body: body as BodyInit | undefined,
    credentials: 'include',
  });
}

async function decodeEnvelope<T>(response: Response): Promise<ApiEnvelope<T>> {
  if (response.status === 204) {
    return { success: true, data: undefined as T };
  }
  const payload = (await response.json().catch(() => null)) as ApiEnvelope<T> | null;
  if (!response.ok || !payload?.success) {
    const fallback: ApiErrorBody = {
      code: response.status === 0 ? 'NETWORK_ERROR' : 'UNKNOWN_ERROR',
      message: 'Tidak dapat memproses permintaan. Periksa koneksi Anda.',
    };
    throw new ApiError(response.status, payload?.error ?? fallback, payload?.request_id);
  }
  return payload;
}

async function decode<T>(response: Response): Promise<T> {
  const env = await decodeEnvelope<T>(response);
  return env.data;
}

/**
 * Fetch wrapper for the FixPhone API.
 *
 * Transparently refreshes an expired access token exactly once. Any other 401
 * clears the client session through the registered unauthorized handler.
 */
export async function apiRequest<T>(path: string, options: RequestOptions = {}, retry = true): Promise<T> {
  let response: Response;
  try {
    response = await requestRaw(path, options);
  } catch {
    throw new ApiError(0, { code: 'NETWORK_ERROR', message: 'Tidak dapat terhubung ke server. Periksa koneksi Anda.' });
  }

  if (response.status === 401 && retry && !path.startsWith('/auth/refresh') && !path.startsWith('/auth/login')) {
    const payload = (await response.clone().json().catch(() => null)) as ApiEnvelope<unknown> | null;
    if (payload?.error?.code === 'TOKEN_EXPIRED') {
      const session = await refreshSession();
      if (session) {
        return apiRequest<T>(path, options, false);
      }
    }
    notifyUnauthorized();
  }

  return decode<T>(response);
}

export async function apiRequestEnvelope<T>(path: string, options: RequestOptions = {}, retry = true): Promise<ApiEnvelope<T>> {
  let response: Response;
  try {
    response = await requestRaw(path, options);
  } catch {
    throw new ApiError(0, { code: 'NETWORK_ERROR', message: 'Tidak dapat terhubung ke server. Periksa koneksi Anda.' });
  }

  if (response.status === 401 && retry && !path.startsWith('/auth/refresh') && !path.startsWith('/auth/login')) {
    const payload = (await response.clone().json().catch(() => null)) as ApiEnvelope<unknown> | null;
    if (payload?.error?.code === 'TOKEN_EXPIRED') {
      const session = await refreshSession();
      if (session) {
        return apiRequestEnvelope<T>(path, options, false);
      }
    }
    notifyUnauthorized();
  }

  return decodeEnvelope<T>(response);
}

export const api = {
  get: <T>(path: string) => apiRequest<T>(path, { method: 'GET' }),
  getEnvelope: <T>(path: string) => apiRequestEnvelope<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown) => apiRequest<T>(path, { method: 'POST', body: body ?? {} }),
  put: <T>(path: string, body?: unknown) => apiRequest<T>(path, { method: 'PUT', body: body ?? {} }),
  patch: <T>(path: string, body?: unknown) => apiRequest<T>(path, { method: 'PATCH', body: body ?? {} }),
  delete: <T>(path: string) => apiRequest<T>(path, { method: 'DELETE' }),
};

export interface DownloadedFile {
  blob: Blob;
  fileName: string;
}

/**
 * Downloads a binary attachment (xlsx export, gzip backup) with the same
 * authentication and outlet headers as the JSON API.
 */
export async function apiDownload(path: string, retry = true): Promise<DownloadedFile> {
  let response: Response;
  try {
    response = await requestRaw(path, { method: 'GET' });
  } catch {
    throw new ApiError(0, { code: 'NETWORK_ERROR', message: 'Tidak dapat terhubung ke server. Periksa koneksi Anda.' });
  }

  if (response.status === 401 && retry) {
    const payload = (await response.clone().json().catch(() => null)) as ApiEnvelope<unknown> | null;
    if (payload?.error?.code === 'TOKEN_EXPIRED') {
      const session = await refreshSession();
      if (session) {
        return apiDownload(path, false);
      }
    }
    notifyUnauthorized();
    throw new ApiError(response.status, payload?.error ?? { code: 'UNAUTHORIZED', message: 'Sesi Anda berakhir.' });
  }

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as ApiErrorBody | null;
    throw new ApiError(
      response.status,
      payload ?? { code: 'DOWNLOAD_FAILED', message: 'Berkas tidak dapat diunduh. Coba lagi.' },
    );
  }

  const blob = await response.blob();
  return { blob, fileName: fileNameFromDisposition(response.headers.get('Content-Disposition')) ?? 'unduhan' };
}

function fileNameFromDisposition(disposition: string | null): string | null {
  if (!disposition) return null;
  const utf8 = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8) {
    try {
      return decodeURIComponent(utf8[1].trim());
    } catch {
      return utf8[1].trim();
    }
  }
  const plain = disposition.match(/filename="?([^";]+)"?/i);
  return plain ? plain[1].trim() : null;
}

/** Offers a downloaded blob to the browser as a file save. */
export function saveBlob(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
