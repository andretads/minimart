import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { ApiClient } from '@core/api/api-client';
import { Notification, NotificationPreferences } from '@shared/models/notification.models';

@Injectable({ providedIn: 'root' })
export class NotificationsApiService {
  private readonly api = inject(ApiClient);

  list(): Observable<Notification[]> {
    return this.api.get<{ data: Notification[] }>('/v1/notifications').pipe(map((r) => r.data));
  }

  get(id: string): Observable<Notification> {
    return this.api.get<Notification>(`/v1/notifications/${id}`);
  }

  getPreferences(): Observable<NotificationPreferences> {
    return this.api.get<NotificationPreferences>('/v1/notifications/preferences');
  }

  updatePreferences(preferences: NotificationPreferences): Observable<NotificationPreferences> {
    return this.api.put<NotificationPreferences>('/v1/notifications/preferences', preferences);
  }

  updatePreference(key: string, enabled: boolean): Observable<NotificationPreferences> {
    return new Observable((subscriber) => {
      this.getPreferences().subscribe({
        next: (prefs) => {
          const updated = { ...prefs, [key]: enabled };
          this.updatePreferences(updated).subscribe({
            next: (result) => subscriber.next(result),
            error: (err) => subscriber.error(err),
          });
        },
        error: (err) => subscriber.error(err),
      });
    });
  }

  markAsRead(id: string): Observable<void> {
    return this.api.post<void>(`/v1/notifications/${id}/read`, {});
  }

  markAsUnread(id: string): Observable<void> {
    return this.api.post<void>(`/v1/notifications/${id}/unread`, {});
  }

  markAllAsRead(): Observable<void> {
    return this.api.post<void>('/v1/notifications/read-all', {});
  }

  delete(id: string): Observable<void> {
    return this.api.delete<void>(`/v1/notifications/${id}`);
  }
}
