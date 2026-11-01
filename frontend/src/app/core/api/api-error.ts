export interface ApiErrorEnvelope {
  code: string;
  message: string;
  fields?: Record<string, string[]>;
}

export interface Meta {
  page?: number;
  page_size?: number;
  total?: number;
  total_pages?: number;
  next_cursor?: string | null;
  has_more?: boolean;
}

export interface ApiEnvelope<T> {
  time: string;
  success: boolean;
  data?: T;
  error?: ApiErrorEnvelope;
  meta?: Meta;
}

export interface ApiError extends Error {
  status: number;
  code?: string;
  message: string;
  fields?: Record<string, string[]>;
  originalError?: unknown;
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof Error && 'status' in error;
}

export function isAuthenticationError(error: unknown): boolean {
  return isApiError(error) && (error.status === 401 || error.status === 403);
}

export function createApiError(
  status: number,
  envelope?: ApiEnvelope<unknown>,
  originalError?: unknown,
): ApiError {
  const message = envelope?.error?.message ?? `HTTP ${status}`;
  const code = envelope?.error?.code;
  const fields = envelope?.error?.fields;
  const err = new Error(message) as ApiError;
  err.status = status;
  err.code = code;
  err.fields = fields;
  err.originalError = originalError;
  return err;
}
