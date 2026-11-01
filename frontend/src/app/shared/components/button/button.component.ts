import { Component, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';

export type ButtonVariant =
  'primary' | 'secondary' | 'outline' | 'danger' | 'ghost' | 'main' | 'main-outline';
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg';

@Component({
  selector: 'app-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      [type]="type()"
      [disabled]="disabled() || loading()"
      [class]="computedClasses()"
      (click)="handleClick($event)"
    >
      @if (loading()) {
        <span class="spinner" aria-hidden="true"></span>
      }
      <span [class]="loading() ? 'sr-only' : ''">{{ label() }}</span>
    </button>
  `,
  styles: [
    `
      button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        min-height: 2.35rem;
        padding-inline: 1rem;
        font-weight: 600;
        font-size: 0.875rem;
        letter-spacing: -0.01em;
        border-radius: var(--radius-full);
        border: 1px solid transparent;
        cursor: pointer;
        transition:
          transform var(--transition-fast),
          background-color var(--transition-fast),
          border-color var(--transition-fast),
          color var(--transition-fast),
          box-shadow var(--transition-fast),
          opacity var(--transition-fast);
        font-family: inherit;
        white-space: nowrap;
      }
      button:hover:not(:disabled) {
        transform: translateY(-1px);
      }
      button:active:not(:disabled) {
        transform: translateY(0);
      }
      button:disabled {
        opacity: 0.45;
        cursor: not-allowed;
      }
      button:focus-visible {
        outline: 2px solid var(--color-focus);
        outline-offset: 2px;
      }
      .sr-only {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
      }
      .spinner {
        width: 1em;
        height: 1em;
        border: 2px solid currentColor;
        border-right-color: transparent;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
      }
      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
      }
      .xs {
        min-height: 1.9rem;
        padding-inline: 0.7rem;
        font-size: 0.75rem;
      }
      .sm {
        min-height: 2.1rem;
        padding-inline: 0.85rem;
        font-size: 0.8125rem;
      }
      .md {
        min-height: 2.5rem;
        padding-inline: 1.15rem;
      }
      .lg {
        min-height: 3rem;
        padding-inline: 1.6rem;
        font-size: 0.95rem;
      }
      .primary,
      .main {
        background: var(--accent);
        color: var(--accent-foreground);
        border-color: var(--accent);
        box-shadow: 0 10px 24px -14px color-mix(in oklab, var(--accent) 85%, transparent);
      }
      .primary:hover:not(:disabled),
      .main:hover:not(:disabled) {
        background: var(--accent-hover);
        border-color: var(--accent-hover);
        box-shadow: 0 16px 30px -14px color-mix(in oklab, var(--accent) 80%, transparent);
      }
      .secondary {
        background: var(--muted);
        color: var(--foreground);
        border-color: var(--border);
      }
      .secondary:hover:not(:disabled) {
        background: color-mix(in oklab, var(--muted) 70%, var(--card));
      }
      .outline,
      .main-outline {
        background: transparent;
        color: var(--foreground);
        border-color: var(--border);
      }
      .outline:hover:not(:disabled),
      .main-outline:hover:not(:disabled) {
        background: var(--color-primary-bg);
        color: var(--accent-strong);
        border-color: color-mix(in oklab, var(--accent) 40%, var(--border));
      }
      .danger {
        background: var(--destructive);
        color: var(--destructive-foreground);
        border-color: var(--destructive);
      }
      .danger:hover:not(:disabled) {
        filter: brightness(0.94);
      }
      .ghost {
        background: transparent;
        color: var(--muted-foreground);
        border-color: transparent;
      }
      .ghost:hover:not(:disabled) {
        background: var(--muted);
        color: var(--foreground);
      }
      .full-width {
        width: 100%;
      }
    `,
  ],
})
export class ButtonComponent {
  label = input.required<string>();
  variant = input<ButtonVariant>('primary');
  size = input<ButtonSize>('md');
  type = input<'button' | 'submit' | 'reset'>('button');
  disabled = input(false);
  loading = input(false);
  fullWidth = input(false);
  clicked = output<MouseEvent>();

  computedClasses = computed(() =>
    [this.size(), this.variant(), this.fullWidth() ? 'full-width' : ''].filter(Boolean).join(' '),
  );

  handleClick(event: MouseEvent): void {
    if (!this.disabled() && !this.loading()) {
      this.clicked.emit(event);
    }
  }
}
