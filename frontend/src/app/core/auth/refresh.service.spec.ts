import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { RefreshService } from './refresh.service';
import { AuthStore } from './auth.store';

describe('RefreshService', () => {
  let service: RefreshService;
  let store: AuthStore;
  let http: HttpTestingController;

  const sessionEnvelope = {
    time: '2026-01-01T00:00:00Z',
    success: true,
    data: {
      status: 'refreshed',
      access_token_expires_at: '2026-01-01T00:15:00Z',
      expires_in: 900,
    },
    error: null,
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(RefreshService);
    store = TestBed.inject(AuthStore);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('completes after a successful refresh and stores the new expiry', () => {
    let completed = false;
    service.refresh().subscribe({ complete: () => (completed = true) });

    http.expectOne('/v1/auth/refresh').flush(sessionEnvelope);

    expect(completed).toBe(true);
    expect(store.accessTokenExpiresAt()).toBeGreaterThan(Date.now());
  });

  it('shares one request between concurrent callers', () => {
    let completed = 0;
    service.refresh().subscribe({ complete: () => completed++ });
    service.refresh().subscribe({ complete: () => completed++ });

    http.expectOne('/v1/auth/refresh').flush(sessionEnvelope);

    expect(completed).toBe(2);
  });

  it('keeps the session on a transient refresh failure', () => {
    store.setUser({ id: 'user-1', username: 'john', email: 'john@example.com', createdAt: '' });
    let errored = false;
    service.refresh().subscribe({ error: () => (errored = true) });

    http
      .expectOne('/v1/auth/refresh')
      .flush(
        { success: false, error: { code: 503, message: 'unavailable' } },
        { status: 503, statusText: 'Service Unavailable' },
      );

    expect(errored).toBe(true);
    expect(store.user()).not.toBeNull();
  });

  it('clears the session when the refresh token is rejected', () => {
    store.setUser({ id: 'user-1', username: 'john', email: 'john@example.com', createdAt: '' });
    let errored = false;
    service.refresh().subscribe({ error: () => (errored = true) });

    http
      .expectOne('/v1/auth/refresh')
      .flush(
        { success: false, error: { code: 401, message: 'invalid refresh token' } },
        { status: 401, statusText: 'Unauthorized' },
      );

    expect(errored).toBe(true);
    expect(store.user()).toBeNull();
    expect(store.accessTokenExpiresAt()).toBeNull();
  });
});
