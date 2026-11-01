import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';

export type BadgeSize = 'sm' | 'md' | 'lg';

@Component({
  selector: 'app-status-badge',
  standalone: true,
  imports: [CommonModule],
  template: `
    <span
      class="badge"
      [class]="resolvedVariant()"
      [attr.data-status]="status()"
      [class.badge-sm]="size() === 'sm'"
      [class.badge-lg]="size() === 'lg'"
    >
      {{ resolvedLabel() }}
    </span>
  `,
  styles: [
    `
      .badge {
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
        padding: 0.28rem 0.68rem;
        font-size: 0.73rem;
        font-weight: 650;
        border-radius: var(--radius-full);
        border: 1px solid transparent;
        text-transform: capitalize;
        white-space: nowrap;
      }
      .badge::before {
        content: '';
        width: 0.42rem;
        height: 0.42rem;
        border-radius: 50%;
        background: currentColor;
        box-shadow: 0 0 0 3px color-mix(in oklab, currentColor 15%, transparent);
      }
      .badge-sm {
        padding: 0.15rem 0.5rem;
        font-size: 0.66rem;
      }
      .badge-lg {
        padding: 0.42rem 0.95rem;
        font-size: 0.8rem;
      }
      /* Order statuses */
      .badge.placed {
        background: var(--color-info-bg);
        color: var(--color-info);
      }
      .badge.paid {
        background: var(--color-success-bg);
        color: var(--color-success);
      }
      .badge.payment_failed {
        background: var(--color-danger-bg);
        color: var(--color-danger);
      }
      .badge.payment_unknown {
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      .badge.cancelled {
        background: var(--color-muted-bg);
        color: var(--color-text-muted);
      }
      /* Cart statuses */
      .badge.active {
        background: var(--color-primary-bg);
        color: var(--color-primary);
      }
      .badge.checkout_pending {
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      .badge.consumed {
        background: var(--color-muted-bg);
        color: var(--color-text-muted);
      }
      /* Payment statuses */
      .badge.pending {
        background: var(--color-info-bg);
        color: var(--color-info);
      }
      .badge.succeeded {
        background: var(--color-success-bg);
        color: var(--color-success);
      }
      .badge.failed {
        background: var(--color-danger-bg);
        color: var(--color-danger);
      }
      .badge.unknown {
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      /* Generic statuses */
      .badge.success {
        background: var(--color-success-bg);
        color: var(--color-success);
      }
      .badge.danger {
        background: var(--color-danger-bg);
        color: var(--color-danger);
      }
      .badge.warning {
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      .badge.info {
        background: var(--color-info-bg);
        color: var(--color-info);
      }
    `,
  ],
})
export class StatusBadgeComponent {
  status = input.required<string>();
  label = input<string>('');
  variant = input<string>('');
  size = input<BadgeSize>('md');

  resolvedLabel = computed(() => this.label() || this.status().replace(/_/g, ' '));
  resolvedVariant = computed(() => this.variant() || this.status());
}
