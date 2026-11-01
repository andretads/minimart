import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-logo',
  standalone: true,
  imports: [CommonModule, RouterLink],
  template: `
    <a [routerLink]="['/']" class="logo" aria-label="Página inicial da MiniMart">
      <span class="logo-mark" aria-hidden="true">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4Z" />
          <path d="M3 6h18" />
          <path d="M16 10a4 4 0 0 1-8 0" />
        </svg>
      </span>
      <span class="logo-text">{{ text() }}</span>
    </a>
  `,
  styles: [
    `
      .logo {
        display: inline-flex;
        align-items: center;
        gap: 0.6rem;
        text-decoration: none;
        color: var(--foreground);
      }
      .logo-mark {
        display: grid;
        place-items: center;
        width: 2.25rem;
        height: 2.25rem;
        border-radius: 0.7rem;
        background: linear-gradient(
          145deg,
          var(--accent),
          color-mix(in oklab, var(--accent) 55%, #0f766e)
        );
        color: var(--accent-foreground);
        box-shadow: 0 8px 18px -10px color-mix(in oklab, var(--accent) 70%, transparent);
        transition:
          transform var(--transition),
          box-shadow var(--transition);
      }
      .logo-mark svg {
        width: 1.15rem;
        height: 1.15rem;
      }
      .logo:hover .logo-mark {
        transform: translateY(-1px) rotate(-4deg);
        box-shadow: 0 12px 22px -10px color-mix(in oklab, var(--accent) 80%, transparent);
      }
      .logo-text {
        font-size: 1.3rem;
        font-weight: 800;
        letter-spacing: -0.045em;
      }
    `,
  ],
})
export class LogoComponent {
  text = input('MiniMart');
}
