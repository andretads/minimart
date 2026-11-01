import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/api/api-client';
import { Category } from '../../shared/models/catalog.models';

@Injectable({ providedIn: 'root' })
export class CategoriesApiService {
  private readonly api = inject(ApiClient);

  list(): Observable<Category[]> {
    return this.api.get<Category[]>('/v1/showcase/categories', { skipAuth: true });
  }
}
