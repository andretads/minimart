import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideRouter, withComponentInputBinding } from '@angular/router';
import { provideHttpClient, withInterceptors, withFetch } from '@angular/common/http';
import { provideAnimations } from '@angular/platform-browser/animations';
import { firstValueFrom } from 'rxjs';
import { AuthService } from './core/auth/auth.service';

import { routes } from './app.routes';
import { AUTH_INTERCEPTOR } from './core/auth/auth.interceptor';

function initializeAuth(): Promise<void> {
  return firstValueFrom(inject(AuthService).initAuth());
}

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideRouter(routes, withComponentInputBinding()),
    provideHttpClient(withFetch(), withInterceptors([AUTH_INTERCEPTOR])),
    provideAnimations(),
    provideAppInitializer(initializeAuth),
  ],
};
