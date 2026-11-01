import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ProductReview } from '@shared/models/catalog.models';
import { formatDatePtBr } from '@shared/utils/datetime';

@Component({
  selector: 'app-review-list',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="review-summary">
      <div class="overall-rating">
        @if (hasRating()) {
          <span class="big-rating">{{ averageRating() }}</span>
          <div class="stars" aria-hidden="true">
            @for (filled of stars(); track $index) {
              <span class="icon icon-sm icon-star" [class.muted-star]="!filled"></span>
            }
          </div>
          <span>Baseado em {{ totalReviews() }} avaliações</span>
        } @else {
          <span>Ainda sem avaliações</span>
        }
      </div>
    </div>
    <div class="reviews-list">
      @for (review of reviews(); track review.id) {
        <article class="review-item">
          <div class="review-head">
            <div>
              <strong>{{ reviewAuthor(review) }}</strong>
              <span>{{ reviewDate(review) }}</span>
            </div>
            <div class="stars" [attr.aria-label]="'Avaliação ' + review.rating + ' de 5'">
              @for (s of [1, 2, 3, 4, 5]; track s) {
                <span class="icon icon-sm icon-star" [class.muted-star]="s > review.rating"></span>
              }
            </div>
          </div>
          <p>{{ review.description }}</p>
        </article>
      } @empty {
        <p class="reviews-empty">Nenhuma avaliação aprovada ainda.</p>
      }
    </div>
  `,
  styles: [
    `
      .review-summary {
        display: flex;
        flex-wrap: wrap;
        gap: 2rem;
        padding: 1.25rem;
        margin-bottom: 1.5rem;
        border-radius: 1rem;
        background: var(--color-surface-muted);
      }

      .overall-rating {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.35rem;
        min-width: 9rem;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .big-rating {
        font-size: 2.75rem;
        font-weight: 750;
        line-height: 1;
        color: var(--color-text);
      }

      .overall-rating .stars {
        display: inline-flex;
        gap: 0.1rem;
        color: #f59e0b;
      }

      .reviews-list {
        display: flex;
        flex-direction: column;
      }

      .review-item {
        padding: 1.1rem 0;
        border-bottom: 1px solid var(--color-border);
      }

      .review-item:last-child {
        border-bottom: 0;
      }

      .review-head {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 1rem;
        margin-bottom: 0.35rem;
      }

      .review-head > div:first-child {
        display: flex;
        flex-direction: column;
      }

      .review-head strong {
        font-size: 0.9rem;
      }

      .review-head span {
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .review-head .stars {
        display: inline-flex;
        gap: 0.1rem;
        color: #f59e0b;
      }

      .review-item p {
        margin: 0;
        font-size: 0.875rem;
        line-height: 1.6;
        color: var(--color-text-muted);
      }

      .reviews-empty {
        padding: 1rem 0;
        text-align: center;
        color: var(--color-text-muted);
      }

      .muted-star {
        opacity: 0.2;
      }

      @media (max-width: 640px) {
        .review-summary {
          padding: 1rem;
          gap: 1rem;
        }
        .overall-rating {
          flex-direction: row;
          min-width: auto;
        }
      }
    `,
  ],
})
export class ReviewListComponent {
  readonly reviews = input.required<ProductReview[]>();
  readonly totalReviews = input(0);

  readonly hasRating = computed(() => this.totalReviews() > 0);

  readonly averageRating = computed(() => {
    const reviews = this.reviews();
    if (reviews.length === 0) return '0,0';
    const sum = reviews.reduce((acc, r) => acc + r.rating, 0);
    return (sum / reviews.length).toFixed(1).replace('.', ',');
  });

  readonly stars = computed(() => {
    const reviews = this.reviews();
    if (reviews.length === 0) return [false, false, false, false, false];
    const sum = reviews.reduce((acc, r) => acc + r.rating, 0);
    const avg = Math.round(sum / reviews.length);
    return Array.from({ length: 5 }, (_, index) => index < avg);
  });

  reviewAuthor(review: ProductReview): string {
    return review.customerId ? `Cliente ${review.customerId.slice(0, 6).toUpperCase()}` : 'Cliente';
  }

  reviewDate(review: ProductReview): string {
    return formatDatePtBr(review.createdAt);
  }
}
