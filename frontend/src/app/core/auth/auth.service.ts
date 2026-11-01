import { DestroyRef, effect, inject, Injectable } from '@angular/core';
import { Observable, tap, catchError, throwError, of, map, finalize } from 'rxjs';
import { ApiClient } from '../api/api-client';
import { isAuthenticationError } from '../api/api-error';
import { AuthStore } from './auth.store';
import { RefreshService } from './refresh.service';
import {
  ChangePasswordRequest,
  ChangePasswordResponse,
  LoginRequest,
  SignupRequest,
  SignupResponse,
  MeResponse,
  SessionResponse,
  User,
} from '../../shared/models/auth.models';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly api = inject(ApiClient);
  private readonly store = inject(AuthStore);
  private readonly refreshService = inject(RefreshService);
  private readonly destroyRef = inject(DestroyRef);

  private readonly PROACTIVE_REFRESH_BUFFER_MS = 60_000;
  private readonly MIN_REFRESH_DELAY_MS = 5_000;
  private readonly MAX_REFRESH_RETRY_MS = 5 * 60_000;

  private proactiveRefreshTimeoutId: ReturnType<typeof setTimeout> | null = null;
  private refreshRetryAttempt = 0;

  private readonly onVisibilityChange = (): void => {
    if (typeof document === 'undefined' || document.visibilityState !== 'visible') {
      return;
    }
    const expiresAt = this.store.accessTokenExpiresAt();
    if (!expiresAt || !this.store.isAuthenticated()) {
      return;
    }
    if (expiresAt - Date.now() <= this.PROACTIVE_REFRESH_BUFFER_MS) {
      this.triggerProactiveRefresh();
    }
  };

  constructor() {
    effect(() => {
      this.scheduleProactiveRefresh(this.store.accessTokenExpiresAt());
    });

    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', this.onVisibilityChange);
    }

    this.destroyRef.onDestroy(() => {
      this.clearProactiveRefresh();
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', this.onVisibilityChange);
      }
    });
  }

  initAuth(): Observable<void> {
    return this.me().pipe(
      map(() => void 0),
      catchError(() => {
        // No valid session - just continue without setting user
        return of(void 0);
      }),
    );
  }

  login(credentials: LoginRequest): Observable<User> {
    this.store.setLoading(true);
    this.store.setError(null);
    return this.api.post<SessionResponse>('/v1/auth/login', credentials, { skipAuth: true }).pipe(
      tap((session) => {
        const { accessTokenExpiresAt, expiresIn, ...user } = session;
        this.store.setUser(user);
        this.store.setSessionExpiry(expiresIn, accessTokenExpiresAt);
        this.store.setLoading(false);
      }),
      catchError((error) => {
        this.store.setLoading(false);
        this.store.setError(error.message);
        return throwError(() => error);
      }),
    );
  }

  signup(data: SignupRequest): Observable<SignupResponse> {
    this.store.setLoading(true);
    this.store.setError(null);
    return this.api.post<SignupResponse>('/v1/auth/signup', data, { skipAuth: true }).pipe(
      tap(() => this.store.setLoading(false)),
      catchError((error) => {
        this.store.setLoading(false);
        this.store.setError(error.message);
        return throwError(() => error);
      }),
    );
  }

  changePassword(data: ChangePasswordRequest): Observable<ChangePasswordResponse> {
    return this.api.post<ChangePasswordResponse>('/v1/auth/change-password', {
      current_password: data.currentPassword,
      new_password: data.newPassword,
      confirm_new_password: data.confirmNewPassword,
    });
  }

  me(): Observable<User> {
    return this.api.get<MeResponse>('/v1/auth/me').pipe(
      tap((response) => {
        const { accessTokenExpiresAt, expiresIn, ...user } = response;
        this.store.setUser(user);
        this.store.setSessionExpiry(expiresIn, accessTokenExpiresAt);
      }),
    );
  }

  logout(): Observable<void> {
    return this.api.post<void>('/v1/auth/logout', {}, { skipAuth: true }).pipe(
      finalize(() => {
        this.clearProactiveRefresh();
        this.store.clear();
      }),
    );
  }

  private scheduleProactiveRefresh(expiresAt: number | null): void {
    this.clearProactiveRefresh();

    if (!expiresAt) {
      this.refreshRetryAttempt = 0;
      return;
    }

    const delay = Math.max(
      expiresAt - Date.now() - this.PROACTIVE_REFRESH_BUFFER_MS,
      this.MIN_REFRESH_DELAY_MS,
    );

    this.proactiveRefreshTimeoutId = setTimeout(() => {
      this.proactiveRefreshTimeoutId = null;
      this.triggerProactiveRefresh();
    }, delay);
  }

  private triggerProactiveRefresh(): void {
    if (!this.store.isAuthenticated()) {
      return;
    }

    this.clearProactiveRefresh();
    this.refreshService.refresh().subscribe({
      next: () => {
        this.refreshRetryAttempt = 0;
      },
      error: (error) => {
        if (isAuthenticationError(error)) {
          return;
        }
        this.scheduleRefreshRetry();
      },
    });
  }

  private scheduleRefreshRetry(): void {
    this.clearProactiveRefresh();

    if (!this.store.isAuthenticated()) {
      return;
    }

    const delay = Math.min(30_000 * 2 ** this.refreshRetryAttempt, this.MAX_REFRESH_RETRY_MS);
    this.refreshRetryAttempt++;

    this.proactiveRefreshTimeoutId = setTimeout(() => {
      this.proactiveRefreshTimeoutId = null;
      this.triggerProactiveRefresh();
    }, delay);
  }

  private clearProactiveRefresh(): void {
    if (this.proactiveRefreshTimeoutId !== null) {
      clearTimeout(this.proactiveRefreshTimeoutId);
      this.proactiveRefreshTimeoutId = null;
    }
  }
}
