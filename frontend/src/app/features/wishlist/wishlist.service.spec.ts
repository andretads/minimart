import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { WishlistService } from './wishlist.service';

describe('WishlistService', () => {
  let api: {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    delete: ReturnType<typeof vi.fn>;
  };
  let service: WishlistService;

  beforeEach(() => {
    api = { get: vi.fn(() => of({})), post: vi.fn(() => of({})), delete: vi.fn(() => of(void 0)) };
    TestBed.configureTestingModule({
      providers: [WishlistService, { provide: ApiClient, useValue: api }],
    });
    service = TestBed.inject(WishlistService);
  });

  it('getWishlist GET /v1/wishlist', () => {
    service.getWishlist().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/wishlist');
  });

  it('addItem POST /v1/wishlist/items with { productId }', () => {
    service.addItem('p1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/wishlist/items', { productId: 'p1' });
  });

  it('removeItem DELETE /v1/wishlist/items/:id', () => {
    service.removeItem('p1').subscribe();
    expect(api.delete).toHaveBeenCalledWith('/v1/wishlist/items/p1');
  });

  it('clearWishlist DELETE /v1/wishlist', () => {
    service.clearWishlist().subscribe();
    expect(api.delete).toHaveBeenCalledWith('/v1/wishlist');
  });

  it('toggleItem is identical to addItem (POST same body)', () => {
    service.toggleItem('p1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/wishlist/items', { productId: 'p1' });
  });
});
