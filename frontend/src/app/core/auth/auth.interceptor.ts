import { inject } from '@angular/core';
import {
  HttpInterceptorFn,
  HttpRequest,
  HttpHandlerFn,
  HttpErrorResponse,
  HttpContextToken,
} from '@angular/common/http';
import { throwError, switchMap, catchError } from 'rxjs';
import { RefreshService } from './refresh.service';
import { SKIP_AUTH, SKIP_REFRESH } from '../api/api-client';

export const REFRESH_ATTEMPTED = new HttpContextToken<boolean>(() => false);

export const AUTH_INTERCEPTOR: HttpInterceptorFn = (
  req: HttpRequest<unknown>,
  next: HttpHandlerFn,
) => {
  const refreshService = inject(RefreshService);

  if (req.context.get(SKIP_AUTH)) {
    return next(req);
  }

  // Cookies are sent automatically; there is no Authorization header to add.
  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      if (error.status !== 401) {
        return throwError(() => error);
      }

      if (req.context.get(SKIP_REFRESH) || req.context.get(REFRESH_ATTEMPTED)) {
        return throwError(() => error);
      }

      return refreshService.refresh().pipe(
        switchMap(() => {
          // Retry the original request with the renewed cookies.
          const retryReq = req.clone({
            context: req.context.set(REFRESH_ATTEMPTED, true),
            withCredentials: true,
          });
          return next(retryReq);
        }),
        catchError(() => {
          // RefreshService clears the session only on 401/403. Keep the
          // original error for the caller either way.
          return throwError(() => error);
        }),
      );
    }),
  );
};
