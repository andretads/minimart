export interface ApiEnvelope<T = unknown> {
  time: string;
  success: boolean;
  data?: T;
  error?: { code: string; message: string; fields?: Record<string, string> };
  meta?: Record<string, unknown>;
}

export function ok<T>(data: T, meta?: Record<string, unknown>): ApiEnvelope<T> {
  return {
    time: new Date().toISOString(),
    success: true,
    data,
    meta,
  };
}

export function fail(
  code: string,
  message: string,
  fields?: Record<string, string>,
  meta?: Record<string, unknown>,
): ApiEnvelope<never> {
  return {
    time: new Date().toISOString(),
    success: false,
    error: { code, message, fields },
    meta,
  };
}
