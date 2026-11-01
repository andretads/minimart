import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/api/api-client';
import { Payment, CreatePaymentRequest } from '../../shared/models/payment.models';
import { AuthStore } from '../../core/auth/auth.store';

@Injectable({ providedIn: 'root' })
export class PaymentsApiService {
  private readonly api = inject(ApiClient);
  private readonly authStore = inject(AuthStore);

  private getUserId(): string {
    const userId = this.authStore.userId();
    if (!userId) throw new Error('User not authenticated');
    return userId;
  }

  create(request: CreatePaymentRequest, idempotencyKey: string): Observable<Payment> {
    return this.api.post<Payment>('/v1/payments', request, {
      headers: { 'Idempotency-Key': idempotencyKey },
    });
  }

  get(id: string): Observable<Payment> {
    return this.api.get<Payment>(`/v1/payments/${id}`);
  }
}
