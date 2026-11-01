import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { ShowcaseApiService } from './showcase-api.service';

describe('ShowcaseApiService', () => {
  let api: { get: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn> };
  let service: ShowcaseApiService;

  beforeEach(() => {
    api = { get: vi.fn(() => of({})), post: vi.fn(() => of({})) };
    TestBed.configureTestingModule({
      providers: [ShowcaseApiService, { provide: ApiClient, useValue: api }],
    });
    service = TestBed.inject(ShowcaseApiService);
  });

  it('list builds query with page_size, cursor, category_id', () => {
    service.list({ pageSize: 10, cursor: 'cur1', categoryId: 'cat-1' }).subscribe();
    expect(api.get).toHaveBeenCalledWith(
      '/v1/showcase/products?page_size=10&cursor=cur1&category_id=cat-1',
      { skipAuth: true },
    );
  });

  it('list omits cursor and category_id when not provided', () => {
    service.list({ pageSize: 20 }).subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/showcase/products?page_size=20', {
      skipAuth: true,
    });
  });

  it('list defaults page_size to 20', () => {
    service.list().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/showcase/products?page_size=20', {
      skipAuth: true,
    });
  });

  it('get calls the product endpoint', () => {
    service.get('p1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/showcase/products/p1', { skipAuth: true });
  });

  it('getReviews calls reviews endpoint', () => {
    service.getReviews('p1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/showcase/products/p1/reviews?page=1&page_size=20', {
      skipAuth: true,
    });
  });

  it('createReview posts to /v1/products/:id/reviews without skipAuth', () => {
    service.createReview('p1', { description: 'd', rating: 5 }).subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/products/p1/reviews', {
      description: 'd',
      rating: 5,
    });
  });
});
