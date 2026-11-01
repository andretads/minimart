import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { NotificationsApiService } from './notifications-api.service';
import { NotificationPreferences } from '@shared/models/notification.models';

describe('NotificationsApiService', () => {
  let api: {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    put: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
  };
  let service: NotificationsApiService;

  beforeEach(() => {
    api = {
      get: vi.fn(() => of({})),
      post: vi.fn(() => of({})),
      put: vi.fn(() => of({})),
      delete: vi.fn(() => of(void 0)),
    };
    TestBed.configureTestingModule({
      providers: [NotificationsApiService, { provide: ApiClient, useValue: api }],
    });
    service = TestBed.inject(NotificationsApiService);
  });

  it('list GET /v1/notifications', () => {
    service.list().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/notifications');
  });

  it('get GET /v1/notifications/:id', () => {
    service.get('n1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/notifications/n1');
  });

  it('getPreferences', () => {
    service.getPreferences().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/notifications/preferences');
  });

  it('updatePreferences PUT', () => {
    const p = {
      emailEnabled: true,
      smsEnabled: false,
      pushEnabled: true,
      marketingEnabled: false,
    } as NotificationPreferences;
    service.updatePreferences(p).subscribe();
    expect(api.put).toHaveBeenCalledWith('/v1/notifications/preferences', p);
  });

  it('markAsRead POST /read', () => {
    service.markAsRead('n1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/notifications/n1/read', {});
  });

  it('markAsUnread POST /unread', () => {
    service.markAsUnread('n1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/notifications/n1/unread', {});
  });

  it('markAllAsRead POST /read-all', () => {
    service.markAllAsRead().subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/notifications/read-all', {});
  });

  it('delete DELETE', () => {
    service.delete('n1').subscribe();
    expect(api.delete).toHaveBeenCalledWith('/v1/notifications/n1');
  });

  it('updatePreference composes get then put', () => {
    const prefs = { orderUpdates: true, promotions: true, securityAlerts: true, newsletter: false };
    api.get.mockReturnValue(of(prefs));
    api.put.mockReturnValue(of({ ...prefs, promotions: false }));
    service.updatePreference('promotions', false).subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/notifications/preferences');
    expect(api.put).toHaveBeenCalledWith('/v1/notifications/preferences', {
      ...prefs,
      promotions: false,
    });
  });
});
