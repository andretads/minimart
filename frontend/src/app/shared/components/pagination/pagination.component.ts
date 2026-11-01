import { Component, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ButtonComponent } from '../button/button.component';

@Component({
  selector: 'app-pagination',
  standalone: true,
  imports: [CommonModule, ButtonComponent],
  template: `
    @if (totalPages() > 1) {
      <nav class="pagination" aria-label="Paginação">
        <app-button
          variant="outline"
          size="sm"
          [disabled]="currentPage() === 1"
          (clicked)="pageChange.emit(currentPage() - 1)"
          label="Anterior"
        />

        <div class="page-numbers" role="group" aria-label="Números das páginas">
          @for (page of visiblePages(); track page) {
            @if (page === '...') {
              <span class="ellipsis" aria-hidden="true">…</span>
            } @else {
              <app-button
                size="sm"
                [variant]="page === currentPage() ? 'main' : 'outline'"
                (clicked)="pageChange.emit(page)"
                [attr.aria-current]="page === currentPage() ? 'page' : null"
                [attr.aria-label]="'Página ' + page"
                label="{{ page }}"
              />
            }
          }
        </div>

        <app-button
          variant="outline"
          size="sm"
          [disabled]="currentPage() === totalPages()"
          (clicked)="pageChange.emit(currentPage() + 1)"
          label="Próxima"
        />
      </nav>
    }
  `,
  styles: [
    `
      .pagination {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.45rem;
        flex-wrap: wrap;
        margin-top: clamp(2rem, 4vw, 3rem);
      }
      .page-numbers {
        display: flex;
        gap: 0.35rem;
      }
      .ellipsis {
        display: flex;
        align-items: center;
        padding: 0 0.35rem;
        color: var(--muted-foreground);
      }
    `,
  ],
})
export class PaginationComponent {
  currentPage = input.required<number>();
  totalPages = input.required<number>();
  pageChange = output<number>();

  visiblePages = computed(() => {
    const current = this.currentPage();
    const total = this.totalPages();
    const pages: (number | '...')[] = [];

    if (total <= 7) {
      for (let i = 1; i <= total; i++) pages.push(i);
    } else {
      pages.push(1);
      if (current > 3) pages.push('...');
      const start = Math.max(2, current - 1);
      const end = Math.min(total - 1, current + 1);
      for (let i = start; i <= end; i++) pages.push(i);
      if (current < total - 2) pages.push('...');
      pages.push(total);
    }
    return pages;
  });
}
