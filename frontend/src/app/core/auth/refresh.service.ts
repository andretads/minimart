import { inject, Injectable } from '@angular/core';
import { Observable, catchError, finalize, map, shareReplay, tap, throwError } from 'rxjs';
import { ApiClient } from '../api/api-client';
import { isAuthenticationError } from '../api/api-error';
import { AuthStore } from './auth.store';
import { RefreshResponse } from '../../shared/models/auth.models';

@Injectable({ providedIn: 'root' })
export class RefreshService {
  private readonly api = inject(ApiClient);
  private readonly store = inject(AuthStore);
  private inFlight$: Observable<RefreshResponse> | null = null;

  /**
   * Refreshes the session. Concurrent callers share one request, and the
   * returned observable completes when the request settles so callers relying
   * on completion (for example the app initializer) never hang.
   */
  refresh(): Observable<void> {
    return this.refreshSession().pipe(map(() => void 0));
  }

  private refreshSession(): Observable<RefreshResponse> {
    if (this.inFlight$) {
      return this.inFlight$;
    }

    this.store.setRefreshing(true);

    const request$ = this.api
      .post<RefreshResponse>('/v1/auth/refresh', {}, { skipAuth: true, skipRefresh: true })
      .pipe(
        tap((response) => {
          this.store.setSessionExpiry(response.expiresIn, response.accessTokenExpiresAt);
        }),
        catchError((error) => {
          if (isAuthenticationError(error)) {
            this.store.clear();
          }
          return throwError(() => error);
        }),
        finalize(() => {
          this.inFlight$ = null;
          this.store.setRefreshing(false);
        }),
        shareReplay({ bufferSize: 1, refCount: true }),
      );

    this.inFlight$ = request$;
    return request$;
  }
}
