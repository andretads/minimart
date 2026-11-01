import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { AuthStore } from '@core/auth/auth.store';
import { AvatarApiService } from './avatar-api.service';

describe('AvatarApiService', () => {
  let api: { get: ReturnType<typeof vi.fn>; putFormData: ReturnType<typeof vi.fn> };
  let authStore: { userId: ReturnType<typeof vi.fn> };
  let service: AvatarApiService;

  beforeEach(() => {
    api = { get: vi.fn(() => of({})), putFormData: vi.fn(() => of({})) };
    authStore = { userId: vi.fn(() => 'u1') };
    TestBed.configureTestingModule({
      providers: [
        AvatarApiService,
        { provide: ApiClient, useValue: api },
        { provide: AuthStore, useValue: authStore },
      ],
    });
    service = TestBed.inject(AvatarApiService);
  });

  it('get uses accountId when provided', () => {
    service.get('acc-2').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/avatars/acc-2');
  });

  it('get falls back to authStore.userId', () => {
    service.get().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/avatars/u1');
  });

  it('get throws when no accountId available', () => {
    authStore.userId.mockReturnValue(null);
    expect(() => service.get()).toThrow('User not authenticated');
  });

  it('upload uses putFormData with FormData field image', () => {
    const file = new File(['x'], 'a.png', { type: 'image/png' });
    service.upload(file, 'acc-2').subscribe();
    expect(api.putFormData).toHaveBeenCalled();
    const [path, fd] = api.putFormData.mock.calls[0];
    expect(path).toBe('/v1/avatars/acc-2');
    const sent = fd.get('image') as File;
    expect(sent.name).toBe('a.png');
    expect(sent.type).toBe('image/png');
  });

  it('upload throws when no accountId available', () => {
    authStore.userId.mockReturnValue(null);
    const file = new File(['x'], 'a.png', { type: 'image/png' });
    expect(() => service.upload(file)).toThrow('User not authenticated');
  });

  it('encodes accountId in path', () => {
    service.get('a/b').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/avatars/a%2Fb');
  });
});
