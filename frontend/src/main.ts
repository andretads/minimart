import { bootstrapApplication } from '@angular/platform-browser';
import * as Sentry from '@sentry/angular';
import posthog from 'posthog-js';
import { appConfig } from './app/app.config';
import { App } from './app/app';
import { environment } from './environments/environment';

if (environment.sentryDsn) {
  Sentry.init({
    dsn: environment.sentryDsn,
    environment: environment.production ? 'production' : 'development',
    tracesSampleRate: 0.1,
  });
}

if (environment.posthogKey) {
  posthog.init(environment.posthogKey, {
    api_host: environment.posthogHost,
    capture_pageview: true,
  });
}

bootstrapApplication(App, appConfig).catch((err) => {
  console.error(err);
  if (environment.sentryDsn) {
    Sentry.captureException(err);
  }
});
