import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ButtonComponent } from '../button/button.component';

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  imports: [CommonModule, ButtonComponent],
  template: `
    @if (open()) {
      <div
        class="overlay"
        (click)="close.emit()"
        (keydown.escape)="close.emit()"
        tabindex="-1"
        aria-hidden="true"
      ></div>
      <!-- eslint-disable-next-line @angular-eslint/template/click-events-have-key-events -->
      <div
        class="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="dialog-title"
        (click)="$event.stopPropagation()"
      >
        <header class="dialog-header">
          <span class="dialog-icon" aria-hidden="true"><span class="icon icon-alert"></span></span>
          <h2 id="dialog-title">{{ title() }}</h2>
        </header>
        <div class="dialog-body">
          <p>{{ message() }}</p>
        </div>
        <footer class="dialog-footer">
          <app-button variant="ghost" label="Cancelar" (clicked)="close.emit()" />
          <app-button
            [variant]="confirmVariant()"
            [loading]="confirmLoading()"
            label="{{ confirmLabel() }}"
            (clicked)="confirm.emit()"
          />
        </footer>
      </div>
    }
  `,
  styles: [
    `
      .overlay {
        position: fixed;
        inset: 0;
        background: rgb(0 0 0 / 0.5);
        -webkit-backdrop-filter: blur(4px);
        backdrop-filter: blur(4px);
        z-index: 999;
        animation: fadeIn 0.2s ease;
      }
      .dialog {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: min(430px, calc(100vw - 2rem));
        background: var(--card);
        color: var(--card-foreground);
        border: 1px solid var(--border);
        border-radius: var(--radius-2xl);
        box-shadow: var(--shadow-xl);
        z-index: 1000;
        overflow: hidden;
        animation: slideUp 0.24s cubic-bezier(0.2, 0.65, 0.3, 1);
      }
      .dialog-header {
        display: flex;
        align-items: center;
        gap: 0.7rem;
        padding: 1.15rem 1.35rem 0.9rem;
      }
      .dialog-icon {
        display: grid;
        place-items: center;
        width: 2.25rem;
        height: 2.25rem;
        border-radius: var(--radius-lg);
        background: var(--color-danger-bg);
        color: var(--destructive);
      }
      .dialog-header h2 {
        margin: 0;
        font-size: 1.05rem;
        font-weight: 700;
        letter-spacing: -0.02em;
      }
      .dialog-body {
        padding: 0 1.35rem 1.35rem;
      }
      .dialog-body p {
        margin: 0;
        color: var(--muted-foreground);
        font-size: 0.9rem;
      }
      .dialog-footer {
        display: flex;
        justify-content: flex-end;
        gap: 0.6rem;
        padding: 0.9rem 1.35rem;
        border-top: 1px solid var(--border);
        background: color-mix(in oklab, var(--muted) 45%, transparent);
      }
      @keyframes fadeIn {
        from {
          opacity: 0;
        }
        to {
          opacity: 1;
        }
      }
      @keyframes slideUp {
        from {
          opacity: 0;
          transform: translate(-50%, -44%) scale(0.98);
        }
        to {
          opacity: 1;
          transform: translate(-50%, -50%) scale(1);
        }
      }
    `,
  ],
})
export class ConfirmDialogComponent {
  open = input.required<boolean>();
  title = input.required<string>();
  message = input.required<string>();
  confirmLabel = input<string>('Confirmar');
  confirmVariant = input<'primary' | 'danger'>('danger');
  confirmLoading = input(false);
  // eslint-disable-next-line @angular-eslint/no-output-native
  close = output<void>();
  confirm = output<void>();
}
