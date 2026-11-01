import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '@core/api/api-client';
import { Wishlist } from '@shared/models/wishlist.models';

@Injectable({ providedIn: 'root' })
export class WishlistService {
  private readonly api = inject(ApiClient);

  getWishlist(): Observable<Wishlist> {
    return this.api.get<Wishlist>('/v1/wishlist');
  }

  addItem(productId: string): Observable<Wishlist> {
    return this.api.post<Wishlist>('/v1/wishlist/items', { productId });
  }

  removeItem(productId: string): Observable<Wishlist> {
    return this.api.delete<Wishlist>(`/v1/wishlist/items/${productId}`);
  }

  clearWishlist(): Observable<void> {
    return this.api.delete<void>('/v1/wishlist');
  }

  toggleItem(productId: string): Observable<Wishlist> {
    return this.api.post<Wishlist>('/v1/wishlist/items', { productId });
  }
}
