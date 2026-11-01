import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-skeleton',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div
      class="skeleton"
      [class]="variant()"
      [style.width]="width()"
      [style.height]="height()"
      [style.border-radius]="radius()"
      aria-hidden="true"
    ></div>
  `,
  styles: [
    `
      .skeleton {
        background: linear-gradient(
          90deg,
          var(--color-skeleton-base) 25%,
          var(--color-skeleton-highlight) 50%,
          var(--color-skeleton-base) 75%
        );
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
      }
      .skeleton.text {
        border-radius: var(--radius-sm);
      }
      .skeleton.circular {
        border-radius: 50%;
      }
      .skeleton.rectangular {
        border-radius: var(--radius-lg);
      }
      @keyframes shimmer {
        to {
          background-position: -200% 0;
        }
      }
    `,
  ],
})
export class SkeletonComponent {
  variant = input<'text' | 'circular' | 'rectangular'>('rectangular');
  width = input<string>('100%');
  height = input<string>('1rem');
  radius = input<string>('0.375rem');
}
