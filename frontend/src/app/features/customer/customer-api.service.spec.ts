import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { AuthStore } from '@core/auth/auth.store';
import { CustomerApiService } from './customer-api.service';

describe('CustomerApiService', () => {
  let api: {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    patch: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
  };
  let authStore: { userId: ReturnType<typeof vi.fn> };
  let service: CustomerApiService;

  beforeEach(() => {
    api = {
      get: vi.fn(() => of({})),
      post: vi.fn(() => of({})),
      patch: vi.fn(() => of({})),
      delete: vi.fn(() => of(void 0)),
    };
    authStore = { userId: vi.fn(() => 'u1') };
    TestBed.configureTestingModule({
      providers: [
        CustomerApiService,
        { provide: ApiClient, useValue: api },
        { provide: AuthStore, useValue: authStore },
      ],
    });
    service = TestBed.inject(CustomerApiService);
  });

  it('getByUserId queries by user_id', () => {
    service.getByUserId().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/customers?user_id=u1');
  });

  it('getByUserId throws when unauthenticated', () => {
    authStore.userId.mockReturnValue(null);
    expect(() => service.getByUserId()).toThrow('User not authenticated');
  });

  it('update patches /v1/customers/:id', () => {
    service.update('c1', { displayName: 'Ana', version: 2 }).subscribe();
    expect(api.patch).toHaveBeenCalledWith('/v1/customers/c1', { displayName: 'Ana', version: 2 });
  });

  it('getAddresses', () => {
    service.getAddresses().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/addresses');
  });

  it('createAddress posts', () => {
    service
      .createAddress({
        label: 'L',
        fullName: 'F',
        street: 'S',
        city: 'C',
        state: 'ST',
        zip: 'Z',
        country: 'BR',
      })
      .subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/addresses', expect.any(Object));
  });

  it('updateAddress patches', () => {
    service
      .updateAddress('a1', {
        label: 'L',
        fullName: 'F',
        street: 'S',
        city: 'C',
        state: 'ST',
        zip: 'Z',
        country: 'BR',
        version: 1,
      })
      .subscribe();
    expect(api.patch).toHaveBeenCalledWith('/v1/addresses/a1', expect.any(Object));
  });

  it('deleteAddress deletes', () => {
    service.deleteAddress('a1').subscribe();
    expect(api.delete).toHaveBeenCalledWith('/v1/addresses/a1');
  });

  it('setDefaultAddress posts to /default', () => {
    service.setDefaultAddress('a1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/addresses/a1/default', {});
  });
});
