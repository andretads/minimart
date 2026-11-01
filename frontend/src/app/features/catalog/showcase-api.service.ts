import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/api/api-client';
import {
  Product,
  ProductListResponse,
  ProductReview,
  ProductReviewListResponse,
} from '../../shared/models/catalog.models';

export interface ProductListQuery {
  pageSize?: number;
  cursor?: string | null;
  categoryId?: string;
}

@Injectable({ providedIn: 'root' })
export class ShowcaseApiService {
  private readonly api = inject(ApiClient);

  list(query: ProductListQuery = {}): Observable<ProductListResponse> {
    const params = new URLSearchParams();
    params.set('page_size', String(query.pageSize ?? 20));
    if (query.cursor) {
      params.set('cursor', query.cursor);
    }
    if (query.categoryId) {
      params.set('category_id', query.categoryId);
    }
    return this.api.get<ProductListResponse>(`/v1/showcase/products?${params.toString()}`, {
      skipAuth: true,
    });
  }

  get(id: string): Observable<Product> {
    return this.api.get<Product>(`/v1/showcase/products/${id}`, { skipAuth: true });
  }

  getReviews(id: string, page = 1, pageSize = 20): Observable<ProductReviewListResponse> {
    const params = new URLSearchParams({
      page: String(page),
      page_size: String(pageSize),
    });
    return this.api.get<ProductReviewListResponse>(
      `/v1/showcase/products/${id}/reviews?${params.toString()}`,
      { skipAuth: true },
    );
  }

  createReview(
    productId: string,
    review: { description: string; rating: number },
  ): Observable<ProductReview> {
    return this.api.post<ProductReview>(`/v1/products/${productId}/reviews`, {
      description: review.description,
      rating: review.rating,
    });
  }
}
