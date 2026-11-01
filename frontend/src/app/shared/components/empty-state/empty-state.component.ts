import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ButtonComponent } from '../button/button.component';
import { SkeletonComponent } from '../skeleton/skeleton.component';

@Component({
  selector: 'app-empty-state',
  standalone: true,
  imports: [CommonModule, ButtonComponent, SkeletonComponent],
  template: `
    <div class="empty-state" [class.loading]="loading()">
      @if (loading()) {
        <app-skeleton variant="circular" width="4rem" height="4rem" />
        <app-skeleton variant="text" width="60%" height="1.5rem" />
        <app-skeleton variant="text" width="40%" height="1rem" />
      } @else {
        <div class="empty-icon" aria-hidden="true">
          <span [class]="'icon icon-xl icon-' + (icon() || 'package')"></span>
        </div>
        <h3 class="empty-title">{{ title() }}</h3>
        @if (message()) {
          <p class="empty-message">{{ message() }}</p>
        }
        @if (actionLabel()) {
          <app-button
            [variant]="actionVariant()"
            label="{{ actionLabel() }}"
            (clicked)="action.emit()"
          />
        }
      }
    </div>
  `,
  styles: [
    `
      .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 0.9rem;
        padding: clamp(2.5rem, 6vw, 4.5rem) 1.5rem;
        text-align: center;
      }
      .empty-state.loading {
        min-height: 220px;
      }
      .empty-icon {
        display: grid;
        place-items: center;
        width: 4.5rem;
        height: 4.5rem;
        margin-bottom: 0.35rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-2xl);
        background: var(--color-primary-bg);
        color: var(--accent-strong);
      }
      .empty-icon .icon {
        width: 1.85rem;
        height: 1.85rem;
      }
      .empty-title {
        margin: 0;
        font-size: 1.2rem;
        font-weight: 700;
        letter-spacing: -0.02em;
        color: var(--foreground);
      }
      .empty-message {
        margin: 0;
        max-width: 34ch;
        color: var(--muted-foreground);
      }
    `,
  ],
})
export class EmptyStateComponent {
  title = input.required<string>();
  message = input<string>('');
  icon = input('📦');
  loading = input(false);
  actionLabel = input<string>('');
  actionVariant = input<'primary' | 'secondary' | 'outline'>('primary');
  action = output<void>();
}
