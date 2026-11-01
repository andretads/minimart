import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-page-shell',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="page-shell" [class.bare]="!header()">
      @if (header()) {
        <header class="page-header">
          <div class="page-heading">
            <span class="page-eyebrow">MiniMart</span>
            <h1 class="page-title">{{ title() }}</h1>
          </div>
          <ng-content select="[slot=header-actions]"></ng-content>
        </header>
      }
      <main class="page-content">
        <ng-content></ng-content>
      </main>
      @if (footer()) {
        <footer class="page-footer">
          <ng-content select="[slot=footer]"></ng-content>
        </footer>
      }
    </div>
  `,
  styles: [
    `
      .page-shell {
        display: flex;
        flex-direction: column;
        width: 100%;
        max-width: var(--container);
        min-height: 60vh;
        margin-inline: auto;
        padding: clamp(1.75rem, 4vw, 3rem) clamp(1rem, 4vw, 2rem) clamp(3.5rem, 7vw, 6rem);
      }
      .page-shell.bare {
        max-width: none;
        min-height: 0;
        padding: 0;
      }
      .page-header {
        display: flex;
        align-items: flex-end;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 1rem;
        margin-bottom: clamp(1.75rem, 3vw, 2.5rem);
      }
      .page-heading {
        min-width: 0;
      }
      .page-eyebrow {
        display: block;
        margin-bottom: 0.45rem;
        color: var(--accent-strong);
        font-size: 0.7rem;
        font-weight: 700;
        letter-spacing: 0.16em;
        text-transform: uppercase;
      }
      .page-title {
        margin: 0;
        font-size: clamp(1.75rem, 3.6vw, 2.75rem);
        font-weight: 750;
        letter-spacing: -0.04em;
        color: var(--foreground);
      }
      .page-content {
        flex: 1;
        min-width: 0;
      }
      .page-footer {
        margin-top: 3rem;
        padding-top: 2rem;
        border-top: 1px solid var(--border);
      }
      @media (max-width: 640px) {
        .page-title {
          font-size: 1.65rem;
        }
      }
    `,
  ],
})
export class PageShellComponent {
  title = input<string>('');
  header = input(true);
  footer = input(false);
}
