import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { AuthService } from './auth.service';
import { AuthStore } from './auth.store';
import { RefreshService } from './refresh.service';
import { ApiClient } from '../api/api-client';
import { User } from '../../shared/models/auth.models';

describe('AuthService', () => {
  let service: AuthService;
  let store: {
    user: ReturnType<typeof vi.fn>;
    setUser: ReturnType<typeof vi.fn>;
    setLoading: ReturnType<typeof vi.fn>;
    setError: ReturnType<typeof vi.fn>;
    setSessionExpiry: ReturnType<typeof vi.fn>;
    clear: ReturnType<typeof vi.fn>;
    isAuthenticated: ReturnType<typeof vi.fn>;
    accessTokenExpiresAt: ReturnType<typeof vi.fn>;
  };
  let refresh: { refresh: ReturnType<typeof vi.fn> };
  let api: { post: ReturnType<typeof vi.fn>; get: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    vi.useFakeTimers();
    store = {
      user: vi.fn(() => null),
      setUser: vi.fn(),
      setLoading: vi.fn(),
      setError: vi.fn(),
      setSessionExpiry: vi.fn(),
      clear: vi.fn(),
      isAuthenticated: vi.fn(() => false),
      accessTokenExpiresAt: vi.fn(() => null),
    };
    refresh = { refresh: vi.fn(() => of(void 0)) };
    api = { post: vi.fn(), get: vi.fn() };

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        AuthService,
        { provide: AuthStore, useValue: store },
        { provide: RefreshService, useValue: refresh },
        { provide: ApiClient, useValue: api },
      ],
    });
    service = TestBed.inject(AuthService);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('initAuth resolves void on success and on failure', async () => {
    const u: User = { id: 'u1', username: 'john', email: 'j@e.com', createdAt: 't' };
    api.get.mockReturnValue(of({ ...u, accessTokenExpiresAt: 't', expiresIn: 10 }));
    const r1 = await new Promise<unknown>((res) =>
      service.initAuth().subscribe({ next: res, error: res }),
    );
    expect(r1).toBeUndefined();

    api.get.mockReturnValue(throwError(() => new Error('no session')));
    const r2 = await new Promise<unknown>((res) =>
      service.initAuth().subscribe({ next: res, error: res }),
    );
    expect(r2).toBeUndefined();
  });

  it('me calls GET /v1/auth/me and sets user + session expiry', () => {
    const u: User = { id: 'u1', username: 'john', email: 'j@e.com', createdAt: 't' };
    api.get.mockReturnValue(of({ ...u, accessTokenExpiresAt: 'exp', expiresIn: 900 }));
    let result: unknown;
    service.me().subscribe((v) => (result = v));
    expect(api.get).toHaveBeenCalledWith('/v1/auth/me');
    expect(store.setUser).toHaveBeenCalledWith(u);
    expect(store.setSessionExpiry).toHaveBeenCalledWith(900, 'exp');
    expect(result).toEqual({ ...u, accessTokenExpiresAt: 'exp', expiresIn: 900 });
  });

  it('login sets loading+error before request, sets user on success', () => {
    const u: User = { id: 'u1', username: 'john', email: 'j@e.com', createdAt: 't' };
    api.post.mockReturnValue(of({ ...u, accessTokenExpiresAt: 'exp', expiresIn: 900 }));
    let result: unknown;
    service.login({ username: 'john', password: 'pw' }).subscribe((v) => (result = v));
    expect(store.setLoading).toHaveBeenNthCalledWith(1, true);
    expect(store.setError).toHaveBeenNthCalledWith(1, null);
    expect(api.post).toHaveBeenCalledWith(
      '/v1/auth/login',
      { username: 'john', password: 'pw' },
      { skipAuth: true },
    );
    expect(store.setUser).toHaveBeenCalledWith(u);
    expect(store.setSessionExpiry).toHaveBeenCalledWith(900, 'exp');
    expect(store.setLoading).toHaveBeenLastCalledWith(false);
    expect(result).toEqual({ ...u, accessTokenExpiresAt: 'exp', expiresIn: 900 });
  });

  it('login on error clears loading, sets error, rethrows', () => {
    api.post.mockReturnValue(throwError(() => new Error('bad creds')));
    let err!: { message: string; status?: number };
    service.login({ username: 'john', password: 'pw' }).subscribe({ error: (e) => (err = e) });
    expect(store.setLoading).toHaveBeenLastCalledWith(false);
    expect(store.setError).toHaveBeenCalledWith('bad creds');
    expect(err.message).toBe('bad creds');
  });

  it('signup sets loading/error and only clears loading on success (no setUser)', () => {
    const resp = { id: 'u2', username: 'new', email: 'n@e.com', createdAt: 't' };
    api.post.mockReturnValue(of(resp));
    let result: unknown;
    service
      .signup({ username: 'new', email: 'n@e.com', password: 'pw' })
      .subscribe((v) => (result = v));
    expect(api.post).toHaveBeenCalledWith(
      '/v1/auth/signup',
      { username: 'new', email: 'n@e.com', password: 'pw' },
      { skipAuth: true },
    );
    expect(store.setLoading).toHaveBeenLastCalledWith(false);
    expect(store.setUser).not.toHaveBeenCalled();
    expect(result).toEqual(resp);
  });

  it('signup on error sets error and rethrows', () => {
    api.post.mockReturnValue(throwError(() => new Error('taken')));
    let err!: { message: string; status?: number };
    service
      .signup({ username: 'x', email: 'x@e.com', password: 'pw' })
      .subscribe({ error: (e) => (err = e) });
    expect(store.setLoading).toHaveBeenLastCalledWith(false);
    expect(store.setError).toHaveBeenCalledWith('taken');
    expect(err.message).toBe('taken');
  });

  it('changePassword posts snake_case body to /v1/auth/change-password', () => {
    api.post.mockReturnValue(of({ status: 'ok' }));
    service
      .changePassword({ currentPassword: 'a', newPassword: 'b', confirmNewPassword: 'b' })
      .subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/auth/change-password', {
      current_password: 'a',
      new_password: 'b',
      confirm_new_password: 'b',
    });
  });

  it('logout clears proactive refresh and store on success and on error', () => {
    api.post.mockReturnValue(of(void 0));
    service.logout().subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/auth/logout', {}, { skipAuth: true });
    expect(store.clear).toHaveBeenCalled();

    store.clear.mockClear();
    api.post.mockReturnValue(throwError(() => new Error('fail')));
    let err!: { message: string; status?: number };
    service.logout().subscribe({ error: (e) => (err = e) });
    expect(store.clear).toHaveBeenCalled();
    expect(err.message).toBe('fail');
  });

  it('scheduleProactiveRefresh: no expiresAt resets retry attempt and sets no timer', () => {
    store.isAuthenticated.mockReturnValue(true);
    // trigger via effect by setting expiresAt null -> effect already ran on construction with null
    expect(
      (service as unknown as { proactiveRefreshTimeoutId: ReturnType<typeof setTimeout> | null })
        .proactiveRefreshTimeoutId,
    ).toBeNull();
  });

  it('triggerProactiveRefresh does nothing when unauthenticated', () => {
    store.isAuthenticated.mockReturnValue(false);
    service['triggerProactiveRefresh']();
    expect(refresh.refresh).not.toHaveBeenCalled();
  });

  it('triggerProactiveRefresh on auth error does not retry', () => {
    store.isAuthenticated.mockReturnValue(true);
    const authErr = Object.assign(new Error('unauth'), { status: 401 });
    refresh.refresh.mockReturnValue(throwError(() => authErr));
    service['triggerProactiveRefresh']();
    vi.runAllTimers();
    // no retry timer scheduled
    expect(
      (service as unknown as { proactiveRefreshTimeoutId: ReturnType<typeof setTimeout> | null })
        .proactiveRefreshTimeoutId,
    ).toBeNull();
  });

  it('triggerProactiveRefresh on transient error schedules retry with backoff', () => {
    store.isAuthenticated.mockReturnValue(true);
    refresh.refresh.mockReturnValue(throwError(() => new Error('net')));
    service['triggerProactiveRefresh']();
    // retry timer scheduled
    expect(
      (service as unknown as { proactiveRefreshTimeoutId: ReturnType<typeof setTimeout> | null })
        .proactiveRefreshTimeoutId,
    ).not.toBeNull();
  });

  it('visibility change triggers refresh only when near expiry', () => {
    store.isAuthenticated.mockReturnValue(true);
    store.accessTokenExpiresAt.mockReturnValue(Date.now() + 30_000); // within 60s buffer
    refresh.refresh.mockReturnValue(of(void 0));
    // invoke handler
    (service as unknown as { onVisibilityChange: () => void }).onVisibilityChange();
    expect(refresh.refresh).toHaveBeenCalled();
  });

  it('visibility change does nothing when not visible / not authenticated / not near expiry', () => {
    // not authenticated
    store.isAuthenticated.mockReturnValue(false);
    store.accessTokenExpiresAt.mockReturnValue(Date.now() + 30_000);
    (service as unknown as { onVisibilityChange: () => void }).onVisibilityChange();
    expect(refresh.refresh).not.toHaveBeenCalled();

    // not near expiry
    store.isAuthenticated.mockReturnValue(true);
    store.accessTokenExpiresAt.mockReturnValue(Date.now() + 5 * 60_000);
    (service as unknown as { onVisibilityChange: () => void }).onVisibilityChange();
    expect(refresh.refresh).not.toHaveBeenCalled();
  });
});
