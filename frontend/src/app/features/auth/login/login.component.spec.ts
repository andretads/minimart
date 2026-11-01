import { TestBed } from '@angular/core/testing';
import { Observable, of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { LoginComponent } from './login.component';
import { AuthService } from '@core/auth/auth.service';
import { AuthStore } from '@core/auth/auth.store';

describe('LoginComponent', () => {
  let authService: { login: ReturnType<typeof vi.fn> };
  let authStore: { isAuthenticated: ReturnType<typeof vi.fn> };
  let router: { navigate: ReturnType<typeof vi.fn> };
  let route: { queryParams: Observable<Params> };

  beforeEach(async () => {
    authService = { login: vi.fn(() => of(void 0)) };
    authStore = { isAuthenticated: vi.fn(() => false) };
    router = { navigate: vi.fn(() => Promise.resolve(true)) };
    route = { queryParams: of({ returnUrl: '/cart' }) };

    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [
        { provide: AuthService, useValue: authService },
        { provide: AuthStore, useValue: authStore },
        { provide: Router, useValue: router },
        { provide: ActivatedRoute, useValue: route },
      ],
    }).compileComponents();
  });

  it('reads returnUrl from query params', () => {
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    expect(f.componentInstance.returnUrl()).toBe('/cart');
  });

  it('redirects immediately when already authenticated', () => {
    authStore.isAuthenticated.mockReturnValue(true);
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    expect(router.navigate).toHaveBeenCalledWith(['/cart']);
  });

  it('does not submit invalid form', () => {
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    f.componentInstance.onSubmit();
    expect(authService.login).not.toHaveBeenCalled();
  });

  it('logs in and navigates to returnUrl', () => {
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({ username: 'u', password: 'secret1' });
    f.componentInstance.onSubmit();
    expect(authService.login).toHaveBeenCalledWith({
      username: 'u',
      password: 'secret1',
      rememberMe: false,
    });
    expect(router.navigate).toHaveBeenCalledWith(['/cart']);
  });

  it('sets error message on failure', () => {
    authService.login.mockReturnValue(throwError(() => ({ message: 'bad', status: 401 })));
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({ username: 'u', password: 'secret1' });
    f.componentInstance.onSubmit();
    expect(f.componentInstance.error()).toBe('bad');
    expect(f.componentInstance.showCaptcha()).toBe(false);
  });

  it('shows captcha on 429', () => {
    authService.login.mockReturnValue(throwError(() => ({ message: 'slow', status: 429 })));
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({ username: 'u', password: 'secret1' });
    f.componentInstance.onSubmit();
    expect(f.componentInstance.showCaptcha()).toBe(true);
  });

  it('usernameError reports required when touched', () => {
    const f = TestBed.createComponent(LoginComponent);
    f.componentInstance.form.get('username')!.markAsTouched();
    expect(f.componentInstance.usernameError()).toBe('Username is required');
  });

  it('usernameError reports maxlength', () => {
    const f = TestBed.createComponent(LoginComponent);
    const c = f.componentInstance.form.get('username')!;
    c.setValue('a'.repeat(201));
    c.markAsTouched();
    expect(f.componentInstance.usernameError()).toBe('Username must be 200 characters or less');
  });

  it('passwordError reports required', () => {
    const f = TestBed.createComponent(LoginComponent);
    f.componentInstance.form.get('password')!.markAsTouched();
    expect(f.componentInstance.passwordError()).toBe('Password is required');
  });

  it('passwordError reports minlength', () => {
    const f = TestBed.createComponent(LoginComponent);
    const c = f.componentInstance.form.get('password')!;
    c.setValue('abc');
    c.markAsTouched();
    expect(f.componentInstance.passwordError()).toBe('Password must be at least 6 characters');
  });

  it('passwordError reports maxlength', () => {
    const f = TestBed.createComponent(LoginComponent);
    const c = f.componentInstance.form.get('password')!;
    c.setValue('a'.repeat(73));
    c.markAsTouched();
    expect(f.componentInstance.passwordError()).toBe('Password must be 72 characters or less');
  });

  it('onSubmit is a no-op while loading', () => {
    const f = TestBed.createComponent(LoginComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({ username: 'u', password: 'secret1' });
    f.componentInstance.loading.set(true);
    f.componentInstance.onSubmit();
    expect(authService.login).not.toHaveBeenCalled();
  });
});
