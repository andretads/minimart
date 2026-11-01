import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-spinner',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div
      class="spinner"
      [style.width]="size()"
      [style.height]="size()"
      role="status"
      aria-label="Carregando"
    >
      <span class="sr-only">Carregando...</span>
    </div>
  `,
  styles: [
    `
      .spinner {
        border: 2px solid color-mix(in oklab, var(--accent) 22%, transparent);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
      }
      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
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
    `,
  ],
})
export class SpinnerComponent {
  size = input<string>('1.5rem');
}
