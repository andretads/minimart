import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { authGuard } from './auth.guard';
import { AuthStore } from './auth.store';

describe('authGuard', () => {
  it('allows when authenticated', async () => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: AuthStore, useValue: { isAuthenticated: () => true } },
      ],
    });
    const result = await TestBed.runInInjectionContext(() =>
      authGuard({} as never, { url: '/cart' } as never),
    );
    expect(result).toBe(true);
  });

  it('redirects to /login with returnUrl when unauthenticated', async () => {
    const navigate = vi.fn();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: Router, useValue: { navigate } },
        { provide: AuthStore, useValue: { isAuthenticated: () => false } },
      ],
    });
    const result = await TestBed.runInInjectionContext(() =>
      authGuard({} as never, { url: '/cart?x=1' } as never),
    );
    expect(result).toBe(false);
    expect(navigate).toHaveBeenCalledWith(['/login'], { queryParams: { returnUrl: '/cart?x=1' } });
  });
});
