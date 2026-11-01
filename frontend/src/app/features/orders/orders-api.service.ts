import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/api/api-client';
import { AuthStore } from '../../core/auth/auth.store';
import { Order, OrderListResponse } from '../../shared/models/order.models';

@Injectable({ providedIn: 'root' })
export class OrdersApiService {
  private readonly api = inject(ApiClient);
  private readonly authStore = inject(AuthStore);

  list(page = 1, pageSize = 20): Observable<OrderListResponse> {
    const userId = this.getUserId();
    const params = new URLSearchParams({
      user_id: userId,
      page: String(page),
      page_size: String(pageSize),
    });
    return this.api.get<OrderListResponse>(`/v1/orders?${params.toString()}`);
  }

  get(id: string): Observable<Order> {
    return this.api.get<Order>(`/v1/orders/${id}`);
  }

  cancel(id: string): Observable<Order> {
    return this.api.post<Order>(`/v1/orders/${id}/cancel`, {});
  }

  private getUserId(): string {
    const userId = this.authStore.userId();
    if (!userId) throw new Error('User not authenticated');
    return userId;
  }
}
