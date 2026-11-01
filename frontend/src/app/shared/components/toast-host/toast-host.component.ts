import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ToastService } from '../../../core/toast/toast.service';

@Component({
  selector: 'app-toast-host',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="toast-container" aria-live="polite" aria-atomic="true">
      @for (toast of toasts(); track toast.id) {
        <div class="toast" [class]="toast.type" role="alert">
          <span class="toast-dot" aria-hidden="true"></span>
          <span class="toast-message">{{ toast.message }}</span>
          <button class="toast-close" (click)="dismiss(toast.id)" aria-label="Fechar">
            <span class="icon icon-sm icon-x" aria-hidden="true"></span>
          </button>
        </div>
      }
    </div>
  `,
  styles: [
    `
      .toast-container {
        position: fixed;
        bottom: 1.25rem;
        right: 1.25rem;
        display: flex;
        flex-direction: column;
        gap: 0.6rem;
        z-index: 1000;
        width: min(380px, calc(100vw - 2rem));
      }
      .toast {
        display: flex;
        align-items: center;
        gap: 0.7rem;
        padding: 0.8rem 0.9rem;
        border-radius: var(--radius-lg);
        box-shadow: var(--shadow-lg);
        animation: slideIn 0.3s cubic-bezier(0.2, 0.65, 0.3, 1);
        background: var(--popover);
        color: var(--popover-foreground);
        border: 1px solid var(--border);
      }
      .toast-dot {
        width: 0.55rem;
        height: 0.55rem;
        flex: 0 0 auto;
        border-radius: 50%;
        background: var(--muted-foreground);
      }
      .toast.success .toast-dot {
        background: var(--accent);
      }
      .toast.error .toast-dot {
        background: var(--destructive);
      }
      .toast.warning .toast-dot {
        background: var(--color-warning);
      }
      .toast.info .toast-dot {
        background: var(--color-info);
      }
      .toast-message {
        flex: 1;
        font-size: 0.85rem;
        font-weight: 500;
      }
      .toast-close {
        display: grid;
        place-items: center;
        width: 1.6rem;
        height: 1.6rem;
        border: 0;
        border-radius: var(--radius-sm);
        background: transparent;
        color: var(--muted-foreground);
        cursor: pointer;
      }
      .toast-close:hover {
        background: var(--muted);
        color: var(--foreground);
      }
      @keyframes slideIn {
        from {
          opacity: 0;
          transform: translateY(12px) scale(0.98);
        }
        to {
          opacity: 1;
          transform: translateY(0) scale(1);
        }
      }
    `,
  ],
})
export class ToastHostComponent {
  private readonly toastService = inject(ToastService);
  readonly toasts = this.toastService.toasts;

  dismiss(id: string): void {
    this.toastService.dismiss(id);
  }
}
