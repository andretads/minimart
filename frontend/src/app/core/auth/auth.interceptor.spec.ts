import { TestBed } from '@angular/core/testing';
import { HttpClient, HttpContext, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { AUTH_INTERCEPTOR, REFRESH_ATTEMPTED } from './auth.interceptor';
import { SKIP_AUTH, SKIP_REFRESH } from '../api/api-client';
import { RefreshService } from './refresh.service';
import { AuthStore } from './auth.store';

describe('AUTH_INTERCEPTOR', () => {
  let http: HttpTestingController;
  let httpc: HttpClient;
  let refresh: { refresh: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    refresh = { refresh: vi.fn(() => of(void 0)) };
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([AUTH_INTERCEPTOR])),
        provideHttpClientTesting(),
        { provide: RefreshService, useValue: refresh },
        { provide: AuthStore, useValue: { isAuthenticated: () => true, user: () => null } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    httpc = TestBed.inject(HttpClient);
  });

  afterEach(() => http.verify());

  it('SKIP_AUTH short-circuits without error handling', () => {
    let errored = false;
    httpc.get('/x', { context: new HttpContext().set(SKIP_AUTH, true) }).subscribe({
      error: () => (errored = true),
    });
    http.expectOne('/x').flush(null, { status: 401, statusText: 'Unauthorized' });
    expect(errored).toBe(true);
    expect(refresh.refresh).not.toHaveBeenCalled();
  });

  it('passes through non-401 errors', () => {
    let status = 0;
    httpc.get('/x').subscribe({ error: (e) => (status = (e as { status: number }).status) });
    http.expectOne('/x').flush(null, { status: 500, statusText: 'Server Error' });
    expect(status).toBe(500);
    expect(refresh.refresh).not.toHaveBeenCalled();
  });

  it('on 401 with SKIP_REFRESH does not refresh', () => {
    let status = 0;
    httpc
      .get('/x', { context: new HttpContext().set(SKIP_REFRESH, true) })
      .subscribe({ error: (e) => (status = (e as { status: number }).status) });
    http.expectOne('/x').flush(null, { status: 401, statusText: 'Unauthorized' });
    expect(status).toBe(401);
    expect(refresh.refresh).not.toHaveBeenCalled();
  });

  it('on 401 refreshes and replays with REFRESH_ATTEMPTED set', () => {
    let ok = false;
    httpc.get('/x').subscribe({ next: () => (ok = true) });
    http.expectOne('/x').flush(null, { status: 401, statusText: 'Unauthorized' });
    expect(refresh.refresh).toHaveBeenCalled();
    const replay = http.expectOne('/x');
    expect(replay.request.context.get(REFRESH_ATTEMPTED)).toBe(true);
    replay.flush({ success: true, data: null });
    expect(ok).toBe(true);
  });

  it('on 401 when refresh fails rethrows the original error', () => {
    refresh.refresh.mockReturnValue(throwError(() => new Error('refresh failed')));
    let errored = false;
    httpc.get('/x').subscribe({ error: () => (errored = true) });
    http.expectOne('/x').flush(null, { status: 401, statusText: 'Unauthorized' });
    expect(refresh.refresh).toHaveBeenCalled();
    expect(errored).toBe(true);
  });
});
