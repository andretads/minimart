import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { AuthStore } from '@core/auth/auth.store';
import { ToastService } from '@core/toast/toast.service';
import { Notification } from '@shared/models/notification.models';
import { NotificationsApiService } from './notifications-api.service';
import { NotificationsComponent } from './notifications.component';

const notification: Notification = {
  id: 'n1',
  type: 'order_confirmation',
  userId: 'u1',
  orderId: 'o1',
  subject: 'Pedido confirmado',
  body: 'Seu pedido foi confirmado',
  sentAt: '2026-01-01T00:00:00Z',
  createdAt: '2026-01-01T00:00:00Z',
};

describe('NotificationsComponent', () => {
  let api: {
    list: ReturnType<typeof vi.fn>;
    markAsRead: ReturnType<typeof vi.fn>;
    markAllAsRead: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
  };
  let toast: { error: ReturnType<typeof vi.fn>; success: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    api = {
      list: vi.fn(() => of([notification])),
      markAsRead: vi.fn(() => of(void 0)),
      markAllAsRead: vi.fn(() => of(void 0)),
      delete: vi.fn(() => of(void 0)),
    };
    toast = { error: vi.fn(), success: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [NotificationsComponent],
      providers: [
        { provide: NotificationsApiService, useValue: api },
        { provide: AuthStore, useValue: { userId: vi.fn(() => 'u1') } },
        { provide: ToastService, useValue: toast },
        provideRouter([]),
      ],
    }).compileComponents();
  });

  it('marks an unread notification as read when opening its order, silently', () => {
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();

    fixture.componentInstance.openOrder(notification);

    expect(api.markAsRead).toHaveBeenCalledWith('n1');
    expect(toast.success).not.toHaveBeenCalled();
  });

  it('does not call the API when the notification is already read', () => {
    const readNotification = { ...notification, readAt: '2026-01-02T00:00:00Z' };
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();

    fixture.componentInstance.openOrder(readNotification);

    expect(api.markAsRead).not.toHaveBeenCalled();
  });

  it('shows a toast when marking as read explicitly', () => {
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();

    fixture.componentInstance.markAsRead(notification);

    expect(api.markAsRead).toHaveBeenCalledWith('n1');
    expect(toast.success).toHaveBeenCalledWith('Marcada como lida');
  });

  it('filters notifications by read state', () => {
    const read = { ...notification, id: 'n2', readAt: '2026-01-02T00:00:00Z' };
    api.list.mockReturnValue(of([notification, read]));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();

    fixture.componentInstance.setFilter('unread');
    expect(fixture.componentInstance.filteredNotifications().map((n) => n.id)).toEqual(['n1']);
    fixture.componentInstance.setFilter('read');
    expect(fixture.componentInstance.filteredNotifications().map((n) => n.id)).toEqual(['n2']);
    fixture.componentInstance.setFilter('all');
    expect(fixture.componentInstance.filteredNotifications().length).toBe(2);
  });

  it('filters out notifications for other users', () => {
    api.list.mockReturnValue(of([notification, { ...notification, id: 'n9', userId: 'other' }]));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.notifications().map((n) => n.id)).toEqual(['n1']);
  });

  it('computes unreadCount and filter counts', () => {
    const read = { ...notification, id: 'n2', readAt: '2026-01-02T00:00:00Z' };
    api.list.mockReturnValue(of([notification, read]));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.unreadCount()).toBe(1);
    expect(fixture.componentInstance.filters[0].count()).toBe(2);
    expect(fixture.componentInstance.filters[2].count()).toBe(1);
  });

  it('sets error on load failure', () => {
    api.list.mockReturnValue(throwError(() => ({ message: 'boom' })));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.error()).toBe('boom');
    expect(fixture.componentInstance.loading()).toBe(false);
  });

  it('silent markAsRead does not toast on failure', () => {
    api.markAsRead.mockReturnValue(throwError(() => ({ message: 'x' })));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    fixture.componentInstance.markAsRead(notification, { silent: true });
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('markAsRead toasts error on failure', () => {
    api.markAsRead.mockReturnValue(throwError(() => ({ message: 'x' })));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    fixture.componentInstance.markAsRead(notification);
    expect(toast.error).toHaveBeenCalledWith('x');
  });

  it('markAllRead marks all and toasts', () => {
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    fixture.componentInstance.markAllRead();
    expect(api.markAllAsRead).toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalledWith('Todas as notificações marcadas como lidas');
  });

  it('markAllRead toasts error on failure', () => {
    api.markAllAsRead.mockReturnValue(throwError(() => ({ message: 'fail' })));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    fixture.componentInstance.markAllRead();
    expect(toast.error).toHaveBeenCalledWith('fail');
  });

  it('deleteNotification removes item and toasts', () => {
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    fixture.componentInstance.deleteNotification('n1');
    expect(api.delete).toHaveBeenCalledWith('n1');
    expect(fixture.componentInstance.notifications()).toEqual([]);
    expect(toast.success).toHaveBeenCalledWith('Notificação excluída');
  });

  it('deleteNotification toasts error on failure', () => {
    api.delete.mockReturnValue(throwError(() => ({ message: 'del-fail' })));
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    fixture.componentInstance.deleteNotification('n1');
    expect(toast.error).toHaveBeenCalledWith('del-fail');
  });

  it('typeLabel and icon helpers', () => {
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.typeLabel('order_confirmation')).toBeTruthy();
    expect(fixture.componentInstance.notificationIcon('order_confirmation')).toBe(
      'icon icon-lg icon-notif-order_confirmation',
    );
    expect(fixture.componentInstance.notificationIcon('payment_failed')).toBe(
      'icon icon-lg icon-notif-payment_failed',
    );
    expect(fixture.componentInstance.notificationIconClass('payment_failed')).toBe(
      'payment_failed',
    );
  });

  it('formatDateTime produces output', () => {
    const fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.formatDateTime('2026-06-15T12:00:00Z')).toContain('2026');
  });
});
