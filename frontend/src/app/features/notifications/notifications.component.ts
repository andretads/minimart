import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subject } from 'rxjs';
import { NotificationsApiService } from '@features/notifications/notifications-api.service';
import {
  Notification,
  NotificationType,
  NOTIFICATION_TYPE_LABELS,
} from '@shared/models/notification.models';
import { ButtonComponent } from '@shared/components/button/button.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { AuthStore } from '@core/auth/auth.store';
import { ToastService } from '@core/toast/toast.service';
import { Router, RouterLink } from '@angular/router';

@Component({
  selector: 'app-notifications',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ButtonComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
    SkeletonComponent,
  ],
  template: `
    <app-page-shell title="Notificações">
      <div class="notifications-page">
        @if (loading()) {
          <div class="notifications-skeleton" aria-busy="true">
            @for (i of [1, 2, 3, 4, 5]; track i) {
              <app-skeleton variant="rectangular" height="5rem" />
            }
          </div>
        } @else if (error()) {
          <app-error-state
            title="Falha ao carregar notificações"
            [message]="error()!"
            retryLabel="Tentar novamente"
            (retry)="loadNotifications()"
          />
        } @else if (notifications().length === 0) {
          <app-empty-state
            title="Nenhuma notificação ainda"
            message="Você não tem notificações. Elas aparecerão aqui quando você fizer pedidos, pagamentos ou quando tivermos atualizações para você."
            icon="bell"
            actionLabel="Atualizar"
            (action)="loadNotifications()"
          />
        } @else {
          <div class="toolbar">
            <p class="toolbar-summary">
              <strong>{{ unreadCount() }}</strong> não lidas de {{ notifications().length }} total
            </p>
            <div class="toolbar-actions">
              <div class="filter-tabs" role="tablist">
                @for (filter of filters; track filter.value) {
                  <button
                    class="filter-tab"
                    [class.active]="activeFilter() === filter.value"
                    (click)="setFilter(filter.value)"
                    role="tab"
                    [attr.aria-selected]="activeFilter() === filter.value"
                  >
                    {{ filter.label }}<span class="filter-count">{{ filter.count() }}</span>
                  </button>
                }
              </div>
              @if (unreadCount() > 0) {
                <app-button
                  variant="ghost"
                  size="sm"
                  label="Marcar todas como lidas"
                  (clicked)="markAllRead()"
                />
              }
            </div>
          </div>

          <div class="notifications-list">
            @for (notification of filteredNotifications(); track notification.id) {
              <article
                class="notification-card"
                [class.unread]="!notificationRead(notification)"
                [class.read]="notificationRead(notification)"
              >
                <span class="notif-icon" [class]="'notif-icon ' + notification.type"
                  ><i [class]="notificationIcon(notification.type)"></i
                ></span>
                <div class="notification-body">
                  <header class="notification-header">
                    <h3>{{ notification.subject }}</h3>
                    @if (!notificationRead(notification)) {
                      <span class="unread-dot" aria-label="Não lida"></span>
                    }
                    <time [attr.dateTime]="notification.sentAt">{{
                      formatDateTime(notification.sentAt)
                    }}</time>
                  </header>
                  <p class="notification-message">{{ notification.body }}</p>
                  <footer class="notification-footer">
                    <span class="order-chip"
                      >Pedido <code>{{ notification.orderId.slice(0, 8) }}…</code></span
                    >
                    <div class="notification-actions">
                      @if (!notificationRead(notification)) {
                        <button
                          class="icon-btn"
                          (click)="markAsRead(notification)"
                          aria-label="Marcar como lida"
                          title="Marcar como lida"
                        >
                          <i class="icon icon-check"></i>
                        </button>
                      }
                      <a
                        [routerLink]="['/orders', notification.orderId]"
                        class="icon-btn"
                        aria-label="Ver pedido"
                        title="Ver pedido"
                        (click)="openOrder(notification)"
                      >
                        <i class="icon icon-eye"></i>
                      </a>
                      <button
                        class="icon-btn danger"
                        (click)="deleteNotification(notification.id)"
                        aria-label="Excluir notificação"
                        title="Excluir"
                      >
                        <i class="icon icon-trash"></i>
                      </button>
                    </div>
                  </footer>
                </div>
              </article>
            }

            @if (filteredNotifications().length === 0 && notifications().length > 0) {
              <div class="empty-filter">
                <i class="icon icon-filter icon-lg"></i>
                <p>
                  Nenhuma notificação
                  {{
                    activeFilter() === 'unread'
                      ? 'não lida'
                      : activeFilter() === 'read'
                        ? 'lida'
                        : ''
                  }}
                  encontrada.
                </p>
              </div>
            }

            <div class="list-footer">
              <app-button
                variant="outline"
                size="sm"
                label="Atualizar"
                (clicked)="loadNotifications()"
              />
              <p>As notificações são criadas de forma assíncrona e podem aparecer com atraso.</p>
            </div>
          </div>
        }
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .notifications-page {
        max-width: 54rem;
        margin: 0 auto;
      }
      .notifications-skeleton {
        display: grid;
        gap: 0.85rem;
      }

      .toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 1rem;
        margin-bottom: 1.25rem;
      }
      .toolbar-summary {
        margin: 0;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .toolbar-summary strong {
        color: var(--color-text);
        font-weight: 700;
      }
      .toolbar-actions {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.6rem;
      }
      .filter-tabs {
        display: flex;
        gap: 0.15rem;
        padding: 0.22rem;
        border: 1px solid var(--color-border);
        border-radius: 0.75rem;
        background: var(--color-surface);
      }
      .filter-tab {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        padding: 0.4rem 0.75rem;
        border: 0;
        border-radius: 0.55rem;
        background: none;
        color: var(--color-text-muted);
        font: inherit;
        font-size: 0.75rem;
        font-weight: 500;
        cursor: pointer;
        transition:
          background 0.2s,
          color 0.2s;
      }
      .filter-tab:hover {
        color: var(--color-text);
      }
      .filter-tab.active {
        background: color-mix(in oklab, var(--accent) 11%, transparent);
        color: var(--color-accent);
        font-weight: 600;
      }
      .filter-count {
        padding: 0.05rem 0.35rem;
        border-radius: 999px;
        background: var(--color-surface-muted);
        font-size: 0.625rem;
        font-weight: 600;
      }
      .filter-tab.active .filter-count {
        background: color-mix(in oklab, var(--accent) 16%, transparent);
      }

      .notifications-list {
        display: grid;
        gap: 0.7rem;
      }
      .notification-card {
        display: flex;
        gap: 0.9rem;
        padding: 1.1rem 1.25rem;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface);
        transition:
          border-color 0.2s,
          box-shadow 0.2s,
          transform 0.2s;
      }
      .notification-card:hover {
        border-color: color-mix(in oklab, var(--accent) 32%, transparent);
        box-shadow: var(--shadow-sm);
        transform: translateY(-1px);
      }
      .notification-card.unread {
        border-left: 3px solid var(--color-accent);
        background: color-mix(in oklab, var(--accent) 4%, var(--color-surface));
      }
      .notification-card.read {
        opacity: 0.82;
      }
      .notif-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 2.6rem;
        height: 2.6rem;
        border-radius: 0.8rem;
        background: color-mix(in oklab, var(--accent) 10%, transparent);
        color: var(--color-accent);
      }
      .notif-icon.order_confirmation {
        background: var(--color-info-bg);
        color: var(--color-info);
      }
      .notif-icon.order_cancelled {
        background: var(--color-surface-muted);
        color: var(--color-text-muted);
      }
      .notif-icon.payment_succeeded {
        background: var(--color-success-bg);
        color: var(--color-success);
      }
      .notif-icon.payment_failed {
        background: var(--color-danger-bg);
        color: var(--color-danger);
      }

      .notification-body {
        flex: 1;
        min-width: 0;
      }
      .notification-header {
        display: flex;
        align-items: center;
        gap: 0.45rem;
      }
      .notification-header h3 {
        margin: 0;
        font-size: 0.875rem;
        font-weight: 650;
        letter-spacing: -0.01em;
      }
      .notification-header time {
        margin-left: auto;
        font-size: 0.6875rem;
        color: var(--color-text-muted);
        white-space: nowrap;
      }
      .unread-dot {
        width: 0.4rem;
        height: 0.4rem;
        border-radius: 999px;
        background: var(--color-accent);
      }
      .notification-message {
        margin: 0.35rem 0 0;
        font-size: 0.8125rem;
        line-height: 1.6;
        color: var(--color-text-muted);
      }

      .notification-footer {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 0.75rem;
        margin-top: 0.8rem;
        padding-top: 0.8rem;
        border-top: 1px solid var(--color-border);
      }
      .order-chip {
        font-size: 0.6875rem;
        color: var(--color-text-muted);
      }
      .order-chip code {
        padding: 0.1rem 0.35rem;
        border-radius: 0.3rem;
        background: var(--color-surface-muted);
        font-family: var(--font-mono);
        font-size: 0.625rem;
      }
      .notification-actions {
        display: flex;
        gap: 0.3rem;
      }
      .icon-btn {
        display: grid;
        place-items: center;
        width: 2rem;
        height: 2rem;
        border: 1px solid var(--color-border);
        border-radius: 0.55rem;
        background: var(--color-surface);
        color: var(--color-text-muted);
        cursor: pointer;
        text-decoration: none;
        transition:
          border-color 0.2s,
          color 0.2s,
          background 0.2s;
      }
      .icon-btn:hover {
        border-color: color-mix(in oklab, var(--accent) 40%, transparent);
        color: var(--color-accent);
        background: color-mix(in oklab, var(--accent) 7%, transparent);
      }
      .icon-btn.danger:hover {
        border-color: color-mix(in oklab, var(--color-danger) 40%, transparent);
        color: var(--color-danger);
        background: var(--color-danger-bg);
      }

      .empty-filter {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
        padding: 2.5rem 1.5rem;
        border: 1px dashed var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface-muted);
        color: var(--color-text-muted);
      }
      .empty-filter p {
        margin: 0;
        font-size: 0.8125rem;
        text-transform: capitalize;
      }

      .list-footer {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.6rem;
        padding-top: 1.25rem;
        margin-top: 0.5rem;
        border-top: 1px solid var(--color-border);
        text-align: center;
      }
      .list-footer p {
        margin: 0;
        font-size: 0.6875rem;
        color: var(--color-text-muted);
      }

      @media (max-width: 640px) {
        .toolbar {
          align-items: stretch;
        }
        .toolbar-actions {
          justify-content: space-between;
        }
        .notification-card {
          padding: 0.9rem 1rem;
        }
        .notification-header {
          flex-wrap: wrap;
        }
        .notification-header time {
          width: 100%;
          margin-left: 0;
        }
        .notification-footer {
          align-items: center;
        }
      }
    `,
  ],
})
export class NotificationsComponent implements OnInit, OnDestroy {
  private readonly api = inject(NotificationsApiService);
  private readonly authStore = inject(AuthStore);
  private readonly toast = inject(ToastService);
  private readonly destroy$ = new Subject<void>();

  notifications = signal<Notification[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  activeFilter = signal<'all' | 'unread' | 'read'>('all');

  filters = [
    { value: 'all' as const, label: 'Todas', count: computed(() => this.notifications().length) },
    { value: 'unread' as const, label: 'Não lidas', count: computed(() => this.unreadCount()) },
    {
      value: 'read' as const,
      label: 'Lidas',
      count: computed(() => this.notifications().length - this.unreadCount()),
    },
  ];

  ngOnInit(): void {
    this.loadNotifications();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadNotifications(): void {
    this.loading.set(true);
    this.error.set(null);
    this.api.list().subscribe({
      next: (notifications) => {
        this.notifications.set(notifications.filter((n) => n.userId === this.authStore.userId()));
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.message);
        this.loading.set(false);
      },
    });
  }

  setFilter(filter: 'all' | 'unread' | 'read'): void {
    this.activeFilter.set(filter);
  }

  filteredNotifications = computed(() => {
    const filter = this.activeFilter();
    const all = this.notifications();
    if (filter === 'all') return all;
    if (filter === 'unread') return all.filter((n) => !this.notificationRead(n));
    return all.filter((n) => this.notificationRead(n));
  });

  unreadCount = computed(
    () => this.notifications().filter((n) => !this.notificationRead(n)).length,
  );

  typeLabel(type: NotificationType): string {
    return NOTIFICATION_TYPE_LABELS[type];
  }

  notificationRead(notification: Notification): boolean {
    return !!notification.readAt;
  }

  notificationIcon(type: NotificationType): string {
    switch (type) {
      case 'order_confirmation':
        return 'icon icon-lg icon-notif-order_confirmation';
      case 'order_cancelled':
        return 'icon icon-lg icon-notif-order_cancelled';
      case 'payment_succeeded':
        return 'icon icon-lg icon-notif-payment_succeeded';
      case 'payment_failed':
        return 'icon icon-lg icon-notif-payment_failed';
      default:
        return 'icon icon-lg icon-bell';
    }
  }

  notificationIconClass(type: NotificationType): string {
    return type;
  }

  markAsRead(notification: Notification, options: { silent?: boolean } = {}): void {
    if (this.notificationRead(notification)) return;
    this.api.markAsRead(notification.id).subscribe({
      next: () => {
        this.notifications.update((list) =>
          list.map((n) =>
            n.id === notification.id ? { ...n, readAt: new Date().toISOString() } : n,
          ),
        );
        if (!options.silent) {
          this.toast.success('Marcada como lida');
        }
      },
      error: (err) => {
        if (!options.silent) {
          this.toast.error(err?.message ?? 'Falha ao marcar como lida');
        }
      },
    });
  }

  openOrder(notification: Notification): void {
    if (!this.notificationRead(notification)) {
      this.markAsRead(notification, { silent: true });
    }
  }

  markAllRead(): void {
    this.api.markAllAsRead().subscribe({
      next: () => {
        const now = new Date().toISOString();
        this.notifications.update((list) => list.map((n) => ({ ...n, readAt: n.readAt ?? now })));
        this.toast.success('Todas as notificações marcadas como lidas');
      },
      error: (err) => this.toast.error(err?.message ?? 'Falha ao marcar todas como lidas'),
    });
  }

  deleteNotification(id: string): void {
    this.api.delete(id).subscribe({
      next: () => {
        this.notifications.update((list) => list.filter((item) => item.id !== id));
        this.toast.success('Notificação excluída');
      },
      error: (err) => this.toast.error(err?.message ?? 'Falha ao excluir notificação'),
    });
  }

  formatDateTime(iso: string): string {
    return new Date(iso).toLocaleString('pt-BR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  private router = inject(Router);
}
