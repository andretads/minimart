import { TestBed } from '@angular/core/testing';
import { AuthStore } from './auth.store';
import { User } from '../../shared/models/auth.models';

describe('AuthStore', () => {
  let store: AuthStore;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    store = TestBed.inject(AuthStore);
  });

  it('starts unauthenticated', () => {
    expect(store.user()).toBeNull();
    expect(store.loading()).toBe(false);
    expect(store.error()).toBeNull();
    expect(store.isRefreshing()).toBe(false);
    expect(store.accessTokenExpiresAt()).toBeNull();
    expect(store.isAuthenticated()).toBe(false);
    expect(store.userId()).toBeNull();
  });

  it('setUser updates user and isAuthenticated/userId', () => {
    const u: User = { id: 'u1', username: 'john', email: 'j@e.com', createdAt: 't' };
    store.setUser(u);
    expect(store.user()).toEqual(u);
    expect(store.isAuthenticated()).toBe(true);
    expect(store.userId()).toBe('u1');
  });

  it('setLoading/setError/setRefreshing/setAccessTokenExpiresAt', () => {
    store.setLoading(true);
    store.setError('boom');
    store.setRefreshing(true);
    store.setAccessTokenExpiresAt(123);
    expect(store.loading()).toBe(true);
    expect(store.error()).toBe('boom');
    expect(store.isRefreshing()).toBe(true);
    expect(store.accessTokenExpiresAt()).toBe(123);
  });

  it('setSessionExpiry: relative expiresIn wins over absolute expiresAt', () => {
    const now = Date.now();
    vi.useFakeTimers();
    vi.setSystemTime(now);
    store.setSessionExpiry(10, '2020-01-01T00:00:00Z');
    expect(store.accessTokenExpiresAt()).toBe(now + 10_000);
    vi.useRealTimers();
  });

  it('setSessionExpiry: falls back to absolute when expiresIn <= 0', () => {
    const target = new Date('2030-01-01T00:00:00Z').getTime();
    store.setSessionExpiry(0, '2030-01-01T00:00:00Z');
    expect(store.accessTokenExpiresAt()).toBe(target);
    store.setSessionExpiry(-5, '2030-01-01T00:00:00Z');
    expect(store.accessTokenExpiresAt()).toBe(target);
    store.setSessionExpiry(undefined, '2030-01-01T00:00:00Z');
    expect(store.accessTokenExpiresAt()).toBe(target);
    store.setSessionExpiry(null, '2030-01-01T00:00:00Z');
    expect(store.accessTokenExpiresAt()).toBe(target);
  });

  it('setSessionExpiry: invalid absolute date -> null', () => {
    store.setSessionExpiry(undefined, 'not-a-date');
    expect(store.accessTokenExpiresAt()).toBeNull();
    store.setSessionExpiry(undefined, null);
    expect(store.accessTokenExpiresAt()).toBeNull();
    store.setSessionExpiry(undefined, undefined);
    expect(store.accessTokenExpiresAt()).toBeNull();
  });

  it('setSessionExpiry: expiresIn non-number ignored', () => {
    store.setSessionExpiry('10' as unknown as number, null);
    expect(store.accessTokenExpiresAt()).toBeNull();
  });

  it('clear resets all state', () => {
    const u: User = { id: 'u1', username: 'john', email: 'j@e.com', createdAt: 't' };
    store.setUser(u);
    store.setLoading(true);
    store.setError('e');
    store.setRefreshing(true);
    store.setAccessTokenExpiresAt(9);
    store.clear();
    expect(store.user()).toBeNull();
    expect(store.loading()).toBe(false);
    expect(store.error()).toBeNull();
    expect(store.isRefreshing()).toBe(false);
    expect(store.accessTokenExpiresAt()).toBeNull();
    expect(store.isAuthenticated()).toBe(false);
    expect(store.userId()).toBeNull();
  });
});
