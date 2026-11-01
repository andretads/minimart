import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../../core/api/api-client';
import { Avatar } from '../../shared/models/avatar.models';
import { AuthStore } from '../../core/auth/auth.store';

@Injectable({ providedIn: 'root' })
export class AvatarApiService {
  private readonly api = inject(ApiClient);
  private readonly authStore = inject(AuthStore);

  private getAccountId(): string {
    const accountId = this.authStore.userId();
    if (!accountId) throw new Error('User not authenticated');
    return accountId;
  }

  get(accountId?: string): Observable<Avatar> {
    const id = accountId ?? this.getAccountId();
    return this.api.get<Avatar>(`/v1/avatars/${encodeURIComponent(id)}`);
  }

  upload(file: File, accountId?: string): Observable<Avatar> {
    const id = accountId ?? this.getAccountId();
    const formData = new FormData();
    formData.append('image', file, file.name);
    return this.api.putFormData<Avatar>(`/v1/avatars/${encodeURIComponent(id)}`, formData);
  }
}
