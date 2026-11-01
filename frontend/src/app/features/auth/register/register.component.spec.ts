import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router';
import { FormControl, FormGroup } from '@angular/forms';
import { RegisterComponent } from './register.component';
import { AuthService } from '@core/auth/auth.service';
import { AuthStore } from '@core/auth/auth.store';

describe('RegisterComponent', () => {
  let authService: { signup: ReturnType<typeof vi.fn> };
  let authStore: { isAuthenticated: ReturnType<typeof vi.fn> };
  let router: { navigate: ReturnType<typeof vi.fn> };
  let route: { snapshot: { queryParamMap: ReturnType<typeof convertToParamMap> } };

  beforeEach(async () => {
    authService = { signup: vi.fn(() => of(void 0)) };
    authStore = { isAuthenticated: vi.fn(() => false) };
    router = { navigate: vi.fn(() => Promise.resolve(true)) };
    route = { snapshot: { queryParamMap: convertToParamMap({ returnUrl: '/orders' }) } };

    await TestBed.configureTestingModule({
      imports: [RegisterComponent],
      providers: [
        { provide: AuthService, useValue: authService },
        { provide: AuthStore, useValue: authStore },
        { provide: Router, useValue: router },
        { provide: ActivatedRoute, useValue: route },
      ],
    }).compileComponents();
  });

  it('reads returnUrl from snapshot', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.detectChanges();
    expect(f.componentInstance.returnUrl()).toBe('/orders');
  });

  it('redirects when already authenticated', () => {
    authStore.isAuthenticated.mockReturnValue(true);
    const f = TestBed.createComponent(RegisterComponent);
    f.detectChanges();
    expect(router.navigate).toHaveBeenCalledWith(['/orders']);
  });

  it('passwordMatchValidator returns mismatch', () => {
    const form = new FormGroup({
      password: new FormControl('abc123'),
      confirmPassword: new FormControl('xyz'),
    });
    expect(RegisterComponent.passwordMatchValidator(form)).toEqual({ passwordMismatch: true });
  });

  it('passwordMatchValidator returns null when matching or empty', () => {
    const match = new FormGroup({
      password: new FormControl('abc123'),
      confirmPassword: new FormControl('abc123'),
    });
    expect(RegisterComponent.passwordMatchValidator(match)).toBeNull();
    const empty = new FormGroup({
      password: new FormControl(''),
      confirmPassword: new FormControl(''),
    });
    expect(RegisterComponent.passwordMatchValidator(empty)).toBeNull();
  });

  it('does not submit invalid form', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.detectChanges();
    f.componentInstance.onSubmit();
    expect(authService.signup).not.toHaveBeenCalled();
  });

  it('signs up and shows success', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({
      username: 'u',
      email: 'a@b.com',
      password: 'secret1',
      confirmPassword: 'secret1',
    });
    f.componentInstance.onSubmit();
    expect(authService.signup).toHaveBeenCalledWith({
      username: 'u',
      email: 'a@b.com',
      password: 'secret1',
    });
    expect(f.componentInstance.success()).toBe(true);
  });

  it('sets error on failure', () => {
    authService.signup.mockReturnValue(throwError(() => ({ message: 'taken' })));
    const f = TestBed.createComponent(RegisterComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({
      username: 'u',
      email: 'a@b.com',
      password: 'secret1',
      confirmPassword: 'secret1',
    });
    f.componentInstance.onSubmit();
    expect(f.componentInstance.error()).toBe('taken');
    expect(f.componentInstance.success()).toBe(false);
  });

  it('confirmPasswordError reports mismatch when touched', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.componentInstance.form.patchValue({ password: 'secret1', confirmPassword: 'nope' });
    f.componentInstance.form.get('confirmPassword')!.markAsTouched();
    expect(f.componentInstance.confirmPasswordError()).toBe('Passwords do not match');
  });

  it('confirmPasswordError reports required', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.componentInstance.form.get('confirmPassword')!.markAsTouched();
    expect(f.componentInstance.confirmPasswordError()).toBe('Please confirm your password');
  });

  it('field error computeds report messages', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.componentInstance.form.get('username')!.markAsTouched();
    f.componentInstance.form.get('email')!.setValue('not-an-email');
    f.componentInstance.form.get('email')!.markAsTouched();
    f.componentInstance.form.get('password')!.markAsTouched();
    expect(f.componentInstance.usernameError()).toBe('Username is required');
    expect(f.componentInstance.emailError()).toBe('Enter a valid email address');
    expect(f.componentInstance.passwordError()).toBe('Password is required');
  });

  it('onSubmit is a no-op while loading', () => {
    const f = TestBed.createComponent(RegisterComponent);
    f.detectChanges();
    f.componentInstance.form.setValue({
      username: 'u',
      email: 'a@b.com',
      password: 'secret1',
      confirmPassword: 'secret1',
    });
    f.componentInstance.loading.set(true);
    f.componentInstance.onSubmit();
    expect(authService.signup).not.toHaveBeenCalled();
  });
});
