import { TestBed } from '@angular/core/testing';
import { ReviewListComponent } from './review-list.component';
import { ProductReview } from '@shared/models/catalog.models';

describe('ReviewListComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ReviewListComponent] }).compileComponents();
  });

  const reviews: ProductReview[] = [
    {
      id: 'r1',
      productId: 'p1',
      customerId: 'abc123xyz',
      description: 'd',
      rating: 4,
      status: 'approved',
      createdAt: '2026-01-02T10:00:00Z',
    },
    {
      id: 'r2',
      productId: 'p1',
      customerId: '',
      description: 'd',
      rating: 5,
      status: 'approved',
      createdAt: '2026-01-03T10:00:00Z',
    },
  ];

  it('hasRating false when totalReviews 0', () => {
    const f = TestBed.createComponent(ReviewListComponent);
    f.componentRef.setInput('reviews', reviews);
    f.componentRef.setInput('totalReviews', 0);
    f.detectChanges();
    expect(f.componentInstance.hasRating()).toBe(false);
  });

  it('averageRating formats with comma', () => {
    const f = TestBed.createComponent(ReviewListComponent);
    f.componentRef.setInput('reviews', reviews);
    f.componentRef.setInput('totalReviews', 2);
    f.detectChanges();
    expect(f.componentInstance.averageRating()).toBe('4,5');
  });

  it('averageRating 0,0 when empty', () => {
    const f = TestBed.createComponent(ReviewListComponent);
    f.componentRef.setInput('reviews', []);
    f.detectChanges();
    expect(f.componentInstance.averageRating()).toBe('0,0');
  });

  it('stars rounds average', () => {
    const f = TestBed.createComponent(ReviewListComponent);
    f.componentRef.setInput('reviews', reviews);
    f.detectChanges();
    expect(f.componentInstance.stars()).toEqual([true, true, true, true, true]);
  });

  it('stars all false when empty', () => {
    const f = TestBed.createComponent(ReviewListComponent);
    f.componentRef.setInput('reviews', []);
    f.detectChanges();
    expect(f.componentInstance.stars()).toEqual([false, false, false, false, false]);
  });

  it('reviewAuthor formats customerId', () => {
    const f = TestBed.createComponent(ReviewListComponent);
    f.componentRef.setInput('reviews', reviews);
    f.detectChanges();
    expect(f.componentInstance.reviewAuthor(reviews[0])).toBe('Cliente ABC123');
    expect(f.componentInstance.reviewAuthor(reviews[1])).toBe('Cliente');
  });
});
