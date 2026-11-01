import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpContext,
  HttpContextToken,
  HttpErrorResponse,
  HttpParams,
  HttpHeaders,
} from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { API_BASE_URL } from './api-base-url.token';
import { ApiEnvelope, createApiError } from './api-error';

export const SKIP_AUTH = new HttpContextToken<boolean>(() => false);
export const SKIP_REFRESH = new HttpContextToken<boolean>(() => false);

export interface ApiClientOptions {
  skipAuth?: boolean;
  skipRefresh?: boolean;
  headers?: Record<string, string>;
}

function toCamelCase(str: string): string {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
}

function toSnakeCase(str: string): string {
  return str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
}

function snakeToCamel(obj: unknown): unknown {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    return obj.map(snakeToCamel);
  }
  const record = obj as Record<string, unknown>;
  const result: Record<string, unknown> = {};
  for (const key in record) {
    if (Object.prototype.hasOwnProperty.call(record, key)) {
      const camelKey = toCamelCase(key);
      result[camelKey] = snakeToCamel(record[key]);
    }
  }
  return result;
}

function camelToSnake(obj: unknown): unknown {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  if (Array.isArray(obj)) {
    return obj.map(camelToSnake);
  }
  const record = obj as Record<string, unknown>;
  const result: Record<string, unknown> = {};
  for (const key in record) {
    if (Object.prototype.hasOwnProperty.call(record, key)) {
      const snakeKey = toSnakeCase(key);
      result[snakeKey] = camelToSnake(record[key]);
    }
  }
  return result;
}

@Injectable({ providedIn: 'root' })
export class ApiClient {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = inject(API_BASE_URL).replace(/\/+$/, '');

  private buildUrl(path: string): string {
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    return `${this.baseUrl}${normalizedPath}`;
  }

  private buildOptions(options: ApiClientOptions = {}): {
    context: HttpContext;
    headers: HttpHeaders;
    params?: HttpParams;
    withCredentials: boolean;
  } {
    const context = new HttpContext()
      .set(SKIP_AUTH, options.skipAuth ?? false)
      .set(SKIP_REFRESH, options.skipRefresh ?? false);
    const headers = new HttpHeaders(options.headers ?? {});
    return {
      context,
      headers,
      withCredentials: true,
    };
  }

  get<T>(path: string, options: ApiClientOptions = {}): Observable<T> {
    return this.request<T>('GET', path, undefined, options);
  }

  post<T>(path: string, body: unknown, options: ApiClientOptions = {}): Observable<T> {
    return this.request<T>('POST', path, body, options);
  }

  patch<T>(path: string, body: unknown, options: ApiClientOptions = {}): Observable<T> {
    return this.request<T>('PATCH', path, body, options);
  }

  put<T>(path: string, body: unknown, options: ApiClientOptions = {}): Observable<T> {
    return this.request<T>('PUT', path, body, options);
  }

  putFormData<T>(path: string, formData: FormData, options: ApiClientOptions = {}): Observable<T> {
    return this.request<T>('PUT', path, formData, options);
  }

  delete<T>(path: string, options: ApiClientOptions = {}): Observable<T> {
    return this.request<T>('DELETE', path, undefined, options);
  }

  private request<T>(
    method: string,
    path: string,
    body: unknown | undefined,
    options: ApiClientOptions,
  ): Observable<T> {
    const { context, headers, withCredentials } = this.buildOptions(options);
    const url = this.buildUrl(path);

    // Convert request body from camelCase to snake_case for mutating requests
    // Skip conversion for FormData (multipart uploads)
    const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
    const processedBody =
      (method === 'POST' || method === 'PATCH' || method === 'PUT') &&
      body !== undefined &&
      !isFormData
        ? camelToSnake(body)
        : body;

    const request$ = this.http.request<ApiEnvelope<T>>(method, url, {
      body: processedBody,
      context,
      headers,
      observe: 'response',
      withCredentials,
    });

    return request$.pipe(
      map((response) => {
        if (response.status === 204) {
          return undefined as T;
        }
        const body = response.body ?? undefined;
        if (!body?.success) {
          throw createApiError(response.status, body);
        }
        if (body.meta !== undefined && body.meta !== null) {
          const meta = body.meta as Record<string, unknown>;
          const normalizedMeta = {
            page: meta['page'] ?? 1,
            pageSize: meta['page_size'] ?? meta['pageSize'] ?? 20,
            total: meta['total'] ?? 0,
            totalPages: meta['total_pages'] ?? meta['totalPages'] ?? 0,
            nextCursor: (meta['next_cursor'] ?? meta['nextCursor'] ?? null) as string | null,
            hasMore: (meta['has_more'] ?? meta['hasMore'] ?? false) as boolean,
          };
          return { data: snakeToCamel(body.data ?? []), meta: normalizedMeta } as T;
        }
        return snakeToCamel(body.data) as T;
      }),
      catchError((error: HttpErrorResponse) => {
        if (error.status === 0) {
          return throwError(() => createApiError(0, undefined, error));
        }
        return throwError(() =>
          createApiError(error.status, error.error as ApiEnvelope<unknown> | undefined, error),
        );
      }),
    );
  }
}
