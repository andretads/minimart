import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '@core/api/api-client';
import {
  Cart,
  AddItemRequest,
  UpdateItemRequest,
  CheckoutResponse,
} from '@shared/models/cart.models';

@Injectable({ providedIn: 'root' })
export class CartService {
  private readonly api = inject(ApiClient);

  getCart(): Observable<Cart> {
    return this.api.get<Cart>('/v1/cart');
  }

  addItem(productId: string, quantity: number): Observable<Cart> {
    return this.api.post<Cart>('/v1/cart/items', { productId, quantity } as AddItemRequest);
  }

  updateItem(productId: string, quantity: number): Observable<Cart> {
    return this.api.patch<Cart>(`/v1/cart/items/${productId}`, { quantity } as UpdateItemRequest);
  }

  removeItem(productId: string): Observable<Cart> {
    return this.api.delete<Cart>(`/v1/cart/items/${productId}`);
  }

  clearCart(): Observable<void> {
    return this.api.delete<void>('/v1/cart');
  }

  checkout(
    idempotencyKey: string,
    shippingQuoteId: string,
    addressId: string,
  ): Observable<CheckoutResponse> {
    return this.api.post<CheckoutResponse>(
      '/v1/cart/checkout',
      { shippingQuoteId, addressId },
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
  }
}
