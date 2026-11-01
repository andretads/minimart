import { Injectable, signal, computed } from '@angular/core';
import { User } from '../../shared/models/auth.models';

@Injectable({ providedIn: 'root' })
export class AuthStore {
  private readonly _user = signal<User | null>(null);
  private readonly _loading = signal(false);
  private readonly _error = signal<string | null>(null);
  private readonly _isRefreshing = signal(false);
  private readonly _accessTokenExpiresAt = signal<number | null>(null);

  readonly user = this._user.asReadonly();
  readonly loading = this._loading.asReadonly();
  readonly error = this._error.asReadonly();
  readonly isRefreshing = this._isRefreshing.asReadonly();
  readonly accessTokenExpiresAt = this._accessTokenExpiresAt.asReadonly();
  readonly isAuthenticated = computed(() => !!this._user());
  readonly userId = computed(() => this._user()?.id ?? null);

  setUser(user: User): void {
    this._user.set(user);
  }

  setLoading(loading: boolean): void {
    this._loading.set(loading);
  }

  setError(error: string | null): void {
    this._error.set(error);
  }

  setRefreshing(refreshing: boolean): void {
    this._isRefreshing.set(refreshing);
  }

  setAccessTokenExpiresAt(expiresAt: number | null): void {
    this._accessTokenExpiresAt.set(expiresAt);
  }

  /**
   * Stores the access-token expiry, preferring the server-provided relative
   * lifetime so client clock skew cannot distort proactive-refresh scheduling.
   */
  setSessionExpiry(
    expiresIn: number | null | undefined,
    expiresAt: string | null | undefined,
  ): void {
    const fromRelative =
      typeof expiresIn === 'number' && expiresIn > 0 ? Date.now() + expiresIn * 1000 : null;
    const parsed = expiresAt ? new Date(expiresAt).getTime() : Number.NaN;
    const fromAbsolute = Number.isFinite(parsed) ? parsed : null;
    this._accessTokenExpiresAt.set(fromRelative ?? fromAbsolute);
  }

  clear(): void {
    this._user.set(null);
    this._error.set(null);
    this._loading.set(false);
    this._isRefreshing.set(false);
    this._accessTokenExpiresAt.set(null);
  }
}
