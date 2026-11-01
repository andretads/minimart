import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { CategoriesApiService } from './categories-api.service';

describe('CategoriesApiService', () => {
  let api: { get: ReturnType<typeof vi.fn> };
  let service: CategoriesApiService;

  beforeEach(() => {
    api = { get: vi.fn(() => of([])) };
    TestBed.configureTestingModule({
      providers: [CategoriesApiService, { provide: ApiClient, useValue: api }],
    });
    service = TestBed.inject(CategoriesApiService);
  });

  it('list calls GET /v1/showcase/categories with skipAuth', () => {
    service.list().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/showcase/categories', { skipAuth: true });
  });
});
