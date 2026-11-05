export interface ApiErrorBody {
  code: string;
  message: string;
  details?: {
    fields?: Record<string, string>;
    [key: string]: unknown;
  };
}

export interface ApiEnvelope<T> {
  success: boolean;
  data: T;
  meta?: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
  error?: ApiErrorBody;
  request_id?: string;
}
