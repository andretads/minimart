import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ButtonComponent } from '../button/button.component';

@Component({
  selector: 'app-error-state',
  standalone: true,
  imports: [CommonModule, ButtonComponent],
  template: `
    <div class="error-state" role="alert">
      <div class="error-icon" aria-hidden="true"><span class="icon icon-xl icon-alert"></span></div>
      <h3 class="error-title">{{ title() }}</h3>
      <p class="error-message">{{ message() }}</p>
      @if (retryLabel()) {
        <app-button
          [variant]="retryVariant()"
          label="{{ retryLabel() }}"
          (clicked)="retry.emit()"
        />
      }
    </div>
  `,
  styles: [
    `
      .error-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 0.9rem;
        padding: clamp(2.5rem, 6vw, 4.5rem) 1.5rem;
        text-align: center;
      }
      .error-icon {
        display: grid;
        place-items: center;
        width: 4.5rem;
        height: 4.5rem;
        margin-bottom: 0.35rem;
        border: 1px solid color-mix(in oklab, var(--destructive) 30%, var(--border));
        border-radius: var(--radius-2xl);
        background: var(--color-danger-bg);
        color: var(--destructive);
      }
      .error-icon .icon {
        width: 1.85rem;
        height: 1.85rem;
      }
      .error-title {
        margin: 0;
        font-size: 1.2rem;
        font-weight: 700;
        letter-spacing: -0.02em;
        color: var(--foreground);
      }
      .error-message {
        margin: 0;
        max-width: 38ch;
        color: var(--muted-foreground);
      }
    `,
  ],
})
export class ErrorStateComponent {
  title = input.required<string>();
  message = input.required<string>();
  retryLabel = input<string>('');
  retryVariant = input<'primary' | 'outline'>('primary');
  retry = output<void>();
}
