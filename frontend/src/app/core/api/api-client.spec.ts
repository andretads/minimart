import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { SKIP_AUTH, SKIP_REFRESH, ApiClient, ApiClientOptions } from './api-client';

describe('ApiClient', () => {
  let client: ApiClient;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    client = TestBed.inject(ApiClient);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('strips trailing slashes from base URL and normalizes path', () => {
    client.get('/foo').subscribe();
    const req = http.expectOne((r) => r.url.endsWith('/foo'));
    req.flush({ success: true, data: { ok: true } });
    expect(req.request.url).not.toMatch(/\/{2}foo/);
  });

  it('adds a leading slash when path has none', () => {
    client.get('foo').subscribe();
    const req = http.expectOne((r) => r.url.endsWith('/foo'));
    req.flush({ success: true, data: null });
  });

  it('sends withCredentials true and context tokens default false', () => {
    client.get('/x').subscribe();
    const req = http.expectOne('/x');
    expect(req.request.withCredentials).toBe(true);
    expect(req.request.context.get(SKIP_AUTH)).toBe(false);
    expect(req.request.context.get(SKIP_REFRESH)).toBe(false);
    req.flush({ success: true, data: null });
  });

  it('honors skipAuth/skipRefresh/headers options', () => {
    const opts: ApiClientOptions = {
      skipAuth: true,
      skipRefresh: true,
      headers: { 'X-Test': '1' },
    };
    client.get('/x', opts).subscribe();
    const req = http.expectOne('/x');
    expect(req.request.context.get(SKIP_AUTH)).toBe(true);
    expect(req.request.context.get(SKIP_REFRESH)).toBe(true);
    expect(req.request.headers.get('X-Test')).toBe('1');
    req.flush({ success: true, data: null });
  });

  it('converts request body camelCase -> snake_case on POST/PATCH/PUT', () => {
    client.post('/x', { productId: 'p1', nestedObj: { lineTotalCents: 5 } }).subscribe();
    const req = http.expectOne('/x');
    expect(req.request.body).toEqual({ product_id: 'p1', nested_obj: { line_total_cents: 5 } });
    req.flush({ success: true, data: null });
  });

  it('does not convert body on GET/DELETE', () => {
    // GET has no body; DELETE likewise. Verify POST with FormData is not converted.
    const fd = new FormData();
    fd.append('image', new Blob(['x']), 'a.png');
    client.putFormData('/x', fd).subscribe();
    const req = http.expectOne('/x');
    expect(req.request.body).toBe(fd);
    req.flush({ success: true, data: null });
  });

  it('unwraps single object (no meta) and converts snake -> camel', () => {
    let result!: {
      data?: unknown;
      meta?: { page: number; pageSize: number; total: number; totalPages: number };
    };
    client.get('/x').subscribe((v) => (result = v as typeof result));
    http.expectOne('/x').flush({ success: true, data: { user_id: 'u1', created_at: 't' } });
    expect(result).toEqual({ userId: 'u1', createdAt: 't' });
  });

  it('unwraps array (no meta)', () => {
    let result!: {
      data?: unknown;
      meta?: { page: number; pageSize: number; total: number; totalPages: number };
    };
    client.get('/x').subscribe((v) => (result = v as typeof result));
    http.expectOne('/x').flush({ success: true, data: [{ a_b: 1 }, { a_b: 2 }] });
    expect(result).toEqual([{ aB: 1 }, { aB: 2 }]);
  });

  it('unwraps meta responses with normalized meta defaults', () => {
    let result!: {
      data?: unknown;
      meta?: { page: number; pageSize: number; total: number; totalPages: number };
    };
    client.get('/x').subscribe((v) => (result = v as typeof result));
    http.expectOne('/x').flush({
      success: true,
      data: [{ id: '1' }],
      meta: { total: 42 },
    });
    expect(result).toEqual({
      data: [{ id: '1' }],
      meta: { page: 1, pageSize: 20, total: 42, totalPages: 0, nextCursor: null, hasMore: false },
    });
  });

  it('honors pageSize/pageSize aliases and total_pages/totalPages aliases', () => {
    let result!: {
      data?: unknown;
      meta?: { page: number; pageSize: number; total: number; totalPages: number };
    };
    client.get('/x').subscribe((v) => (result = v as typeof result));
    http.expectOne('/x').flush({
      success: true,
      data: [],
      meta: { page: 2, page_size: 10, total_pages: 5 },
    });
    expect(result.meta).toEqual({
      page: 2,
      pageSize: 10,
      total: 0,
      totalPages: 5,
      nextCursor: null,
      hasMore: false,
    });
  });

  it('returns undefined on HTTP 204', () => {
    let result: unknown = 'unset';
    client.delete('/x').subscribe((v) => (result = v as typeof result));
    http.expectOne('/x').flush(null, { status: 204, statusText: 'No Content' });
    expect(result).toBeUndefined();
  });

  it('throws ApiError when success is false', () => {
    let err!: { status: number; code?: string; message: string };
    client.get('/x').subscribe({ error: (e) => (err = e) });
    http
      .expectOne('/x')
      .flush(
        { success: false, error: { code: 'E', message: 'nope' } },
        { status: 400, statusText: 'Bad Request' },
      );
    expect(err.status).toBe(400);
    expect(err.code).toBe('E');
    expect(err.message).toBe('nope');
  });

  it('maps HttpErrorResponse (status 0) to createApiError(0)', () => {
    let err!: { status: number; code?: string; message: string };
    client.get('/x').subscribe({ error: (e) => (err = e) });
    http.expectOne('/x').error(new ProgressEvent('err'));
    expect(err.status).toBe(0);
  });

  it('maps HttpErrorResponse (status N) using error.error envelope', () => {
    let err!: { status: number; code?: string; message: string };
    client.get('/x').subscribe({ error: (e) => (err = e) });
    http
      .expectOne('/x')
      .flush(
        { success: false, error: { code: 'E2', message: 'boom' } },
        { status: 500, statusText: 'Server Error' },
      );
    expect(err.status).toBe(500);
    expect(err.code).toBe('E2');
    expect(err.message).toBe('boom');
  });

  it('handles null response body as failure', () => {
    let err!: { status: number; code?: string; message: string };
    client.get('/x').subscribe({ error: (e) => (err = e) });
    http.expectOne('/x').flush(null, { status: 200, statusText: 'OK' });
    expect(err.status).toBe(200);
  });

  it('converts nested arrays and primitives untouched', () => {
    let result!: {
      data?: unknown;
      meta?: { page: number; pageSize: number; total: number; totalPages: number };
    };
    client.get('/x').subscribe((v) => (result = v as typeof result));
    http.expectOne('/x').flush({
      success: true,
      data: { items: [{ product_id: 'p' }], count: 3, flag: true, nothing: null },
    });
    expect(result).toEqual({ items: [{ productId: 'p' }], count: 3, flag: true, nothing: null });
  });
});
