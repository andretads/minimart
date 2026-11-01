import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { guestGuard } from './guest.guard';
import { AuthStore } from './auth.store';

describe('guestGuard', () => {
  it('allows when unauthenticated', async () => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: AuthStore, useValue: { isAuthenticated: () => false } },
      ],
    });
    const result = await TestBed.runInInjectionContext(() => guestGuard({} as never, {} as never));
    expect(result).toBe(true);
  });

  it('redirects to / when authenticated', async () => {
    const navigate = vi.fn();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: Router, useValue: { navigate } },
        { provide: AuthStore, useValue: { isAuthenticated: () => true } },
      ],
    });
    const result = await TestBed.runInInjectionContext(() => guestGuard({} as never, {} as never));
    expect(result).toBe(false);
    expect(navigate).toHaveBeenCalledWith(['/']);
  });
});
