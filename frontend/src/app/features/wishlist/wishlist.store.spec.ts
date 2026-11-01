import { TestBed } from '@angular/core/testing';
import { of, Subject, throwError } from 'rxjs';
import { vi } from 'vitest';
import { WishlistStore } from './wishlist.store';
import { WishlistService } from './wishlist.service';
import { AuthStore } from '@core/auth/auth.store';
import { Wishlist } from '@shared/models/wishlist.models';

describe('WishlistStore', () => {
  let store: WishlistStore;
  let service: {
    getWishlist: ReturnType<typeof vi.fn>;
    addItem: ReturnType<typeof vi.fn>;
    removeItem: ReturnType<typeof vi.fn>;
    clearWishlist: ReturnType<typeof vi.fn>;
  };
  let auth: { userId: ReturnType<typeof vi.fn> };

  const wishlist: Wishlist = {
    id: 'w1',
    customerId: 'c1',
    items: [{ productId: 'p1', addedAt: 't' }],
    createdAt: 't',
    updatedAt: 't',
  };

  beforeEach(() => {
    service = {
      getWishlist: vi.fn(() => of(wishlist)),
      addItem: vi.fn(() => of(wishlist)),
      removeItem: vi.fn(() => of({ ...wishlist, items: [] })),
      clearWishlist: vi.fn(() => of(void 0)),
    };
    auth = { userId: vi.fn(() => 'u1') };
    TestBed.configureTestingModule({
      providers: [
        WishlistStore,
        { provide: WishlistService, useValue: service },
        { provide: AuthStore, useValue: auth },
      ],
    });
    store = TestBed.inject(WishlistStore);
  });

  it('isInWishlist reflects productIds', () => {
    TestBed.flushEffects();
    expect(store.isInWishlist('p1')).toBe(true);
    expect(store.isInWishlist('p2')).toBe(false);
  });

  it('load sets wishlist and loadedForUserId', () => {
    store.load('u1');
    expect(service.getWishlist).toHaveBeenCalled();
    expect(store.wishlist()).toEqual(wishlist);
    expect(store.loading()).toBe(false);
  });

  it('load is no-op when userId unresolved', () => {
    auth.userId.mockReturnValue(null);
    store.load();
    expect(service.getWishlist).not.toHaveBeenCalled();
  });

  it('add calls service and updates wishlist', () => {
    TestBed.flushEffects();
    store.add('p2');
    expect(service.addItem).toHaveBeenCalledWith('p2');
  });

  it('add is no-op when already pending', () => {
    TestBed.flushEffects();
    const subject = new Subject<Wishlist>();
    service.addItem.mockReturnValue(subject.asObservable());
    store.add('p2');
    const calls = service.addItem.mock.calls.length;
    store.add('p2');
    expect(service.addItem.mock.calls.length).toBe(calls);
    subject.next(wishlist);
    subject.complete();
  });

  it('remove calls service', () => {
    TestBed.flushEffects();
    store.remove('p1');
    expect(service.removeItem).toHaveBeenCalledWith('p1');
  });

  it('clear calls service and empties items', () => {
    TestBed.flushEffects();
    store.clear();
    expect(service.clearWishlist).toHaveBeenCalled();
  });

  it('toggle adds when not in wishlist', () => {
    TestBed.flushEffects();
    store.toggle('p9');
    expect(service.addItem).toHaveBeenCalledWith('p9');
  });

  it('toggle removes when in wishlist', () => {
    TestBed.flushEffects();
    store.toggle('p1');
    expect(service.removeItem).toHaveBeenCalledWith('p1');
  });

  it('add reloads on 409 conflict', () => {
    TestBed.flushEffects();
    const conflict = Object.assign(new Error('conflict'), { status: 409 });
    service.addItem.mockReturnValue(throwError(() => conflict));
    store.add('p9');
    expect(service.getWishlist).toHaveBeenCalled();
  });

  it('remove reloads on 404', () => {
    TestBed.flushEffects();
    const notFound = Object.assign(new Error('missing'), { status: 404 });
    service.removeItem.mockReturnValue(throwError(() => notFound));
    store.remove('p1');
    expect(service.getWishlist).toHaveBeenCalled();
  });

  it('load sets error on failure', () => {
    service.getWishlist.mockReturnValue(throwError(() => new Error('boom')));
    store.load('u1');
    expect(store.error()).toBe('boom');
    expect(store.loading()).toBe(false);
  });

  it('clear sets error on failure', () => {
    TestBed.flushEffects();
    service.clearWishlist.mockReturnValue(throwError(() => new Error('nope')));
    store.clear();
    expect(store.error()).toBe('nope');
  });

  it('productIds is a Set', () => {
    TestBed.flushEffects();
    expect(store.productIds()).toBeInstanceOf(Set);
  });
});
