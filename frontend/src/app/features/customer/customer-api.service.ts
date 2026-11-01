import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/api/api-client';
import {
  Customer,
  UpdateCustomerRequest,
  GetCustomerByUserIdResponse,
} from '../../shared/models/customer.models';
import {
  Address,
  CreateAddressRequest,
  UpdateAddressRequest,
} from '../../shared/models/address.models';
import { AuthStore } from '../../core/auth/auth.store';

@Injectable({ providedIn: 'root' })
export class CustomerApiService {
  private readonly api = inject(ApiClient);
  private readonly authStore = inject(AuthStore);

  private getUserId(): string {
    const userId = this.authStore.userId();
    if (!userId) throw new Error('User not authenticated');
    return userId;
  }

  getByUserId(): Observable<GetCustomerByUserIdResponse> {
    return this.api.get<GetCustomerByUserIdResponse>(`/v1/customers?user_id=${this.getUserId()}`);
  }

  update(id: string, data: UpdateCustomerRequest): Observable<Customer> {
    return this.api.patch<Customer>(`/v1/customers/${id}`, data);
  }

  getAddresses(): Observable<Address[]> {
    return this.api.get<Address[]>(`/v1/addresses`);
  }

  createAddress(data: CreateAddressRequest): Observable<Address> {
    return this.api.post<Address>(`/v1/addresses`, data);
  }

  updateAddress(id: string, data: UpdateAddressRequest): Observable<Address> {
    return this.api.patch<Address>(`/v1/addresses/${id}`, data);
  }

  deleteAddress(id: string): Observable<void> {
    return this.api.delete<void>(`/v1/addresses/${id}`);
  }

  setDefaultAddress(id: string): Observable<Address> {
    return this.api.post<Address>(`/v1/addresses/${id}/default`, {});
  }
}
