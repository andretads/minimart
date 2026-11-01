import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormBuilder,
  ReactiveFormsModule,
  Validators,
  AbstractControl,
  ValidationErrors,
} from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { AuthStore } from '../../../core/auth/auth.store';
import { SignupRequest } from '../../../shared/models/auth.models';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { FormFieldComponent } from '../../../shared/components/form-field/form-field.component';
import { PageShellComponent } from '../../../shared/components/page-shell/page-shell.component';
import { LogoComponent } from '../../../shared/components/logo/logo.component';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonComponent,
    FormFieldComponent,
    PageShellComponent,
    LogoComponent,
  ],
  template: `
    <app-page-shell [header]="false" [footer]="false">
      <div class="auth-page">
        <div class="auth-shell">
          <aside class="auth-aside">
            <span class="aside-orb orb-one" aria-hidden="true"></span>
            <span class="aside-orb orb-two" aria-hidden="true"></span>
            <div class="aside-brand">
              <span class="aside-mark"><span class="icon icon-bag" aria-hidden="true"></span></span>
              <span>MiniMart</span>
            </div>
            <div class="aside-copy">
              <span class="aside-eyebrow">Junte-se à MiniMart</span>
              <h2>Seus essenciais do dia a dia em uma conta</h2>
              <p>
                Crie sua conta gratuita para finalizar compras mais rápido, salvar favoritos e
                aproveitar ofertas exclusivas.
              </p>
            </div>
            <ul class="aside-points">
              <li>
                <span class="aside-icon"
                  ><span class="icon icon-gift" aria-hidden="true"></span></span
                ><span
                  ><strong>Vantagens para membros</strong>Ofertas exclusivas desde o início.</span
                >
              </li>
              <li>
                <span class="aside-icon"
                  ><span class="icon icon-heart" aria-hidden="true"></span></span
                ><span><strong>Salve favoritos</strong>Crie sua lista de desejos.</span>
              </li>
              <li>
                <span class="aside-icon"
                  ><span class="icon icon-zap" aria-hidden="true"></span></span
                ><span><strong>Compra mais rápida</strong>Seus dados prontos na hora.</span>
              </li>
            </ul>
            <p class="aside-foot">Já tem uma conta? <a routerLink="/login">Entrar</a></p>
          </aside>

          <section class="auth-panel">
            <div class="auth-card">
              <div class="auth-mobile-logo"><app-logo text="MiniMart" /></div>
              <header class="auth-header">
                <span class="eyebrow">Comece agora</span>
                <h1>Criar conta</h1>
                <p>Preencha seus dados para começar a comprar.</p>
              </header>

              @if (error()) {
                <div class="auth-alert error" role="alert">
                  <span class="icon icon-alert" aria-hidden="true"></span>
                  <span>{{ error() }}</span>
                </div>
              }

              @if (success()) {
                <div class="auth-alert success" role="status">
                  <span class="icon icon-check-circle" aria-hidden="true"></span>
                  <span
                    >Conta criada com sucesso. Você já pode <a routerLink="/login">entrar</a>.</span
                  >
                </div>
              }

              @if (!success()) {
                <form [formGroup]="form" (ngSubmit)="onSubmit()" class="auth-form">
                  <app-form-field
                    id="username"
                    formControlName="username"
                    label="Usuário"
                    type="text"
                    placeholder="Escolha um nome de usuário"
                    [required]="true"
                    [hint]="'Máximo de 200 caracteres'"
                    [error]="usernameError()"
                    [disabled]="loading()"
                  />
                  <app-form-field
                    id="email"
                    formControlName="email"
                    label="E-mail"
                    type="email"
                    placeholder="seuemail@exemplo.com"
                    [required]="true"
                    [error]="emailError()"
                    [disabled]="loading()"
                    [hint]="emailHint"
                  />
                  <app-form-field
                    id="password"
                    formControlName="password"
                    label="Senha"
                    type="password"
                    placeholder="Crie uma senha"
                    [required]="true"
                    [hint]="'6-72 caracteres'"
                    [error]="passwordError()"
                    [disabled]="loading()"
                  />
                  <app-form-field
                    id="confirmPassword"
                    formControlName="confirmPassword"
                    label="Confirmar senha"
                    type="password"
                    placeholder="Confirme sua senha"
                    [required]="true"
                    [error]="confirmPasswordError()"
                    [disabled]="loading()"
                  />

                  <label class="check terms">
                    <input
                      type="checkbox"
                      [checked]="agreeTerms()"
                      (change)="agreeTerms.set($event.target.checked)"
                      required
                    />
                    <span
                      >Concordo com os <a href="#">Termos de Uso</a> e a
                      <a href="#">Política de Privacidade</a>.</span
                    >
                  </label>

                  <app-button
                    type="submit"
                    label="Criar conta"
                    variant="primary"
                    size="lg"
                    [fullWidth]="true"
                    [loading]="loading()"
                    [disabled]="form.invalid || loading()"
                  />
                </form>
              }

              <!-- <div class="divider"><span>or continue with</span></div>

              <div class="social-login">
                <button type="button" class="social-btn" (click)="socialLogin('google')" [disabled]="loading()">
                  <span class="social-g" aria-hidden="true">G</span> Google
                </button>
                <button type="button" class="social-btn" (click)="socialLogin('github')" [disabled]="loading()">
                  <span class="icon icon-github" aria-hidden="true"></span> GitHub
                </button>
              </div> -->

              <p class="auth-footer">
                Já tem uma conta?
                <a [routerLink]="'/login'" [queryParams]="{ returnUrl: returnUrl() }">Entrar</a>
              </p>
            </div>
          </section>
        </div>
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .auth-page {
        display: grid;
        place-items: center;
        min-height: 100vh;
        padding: clamp(1rem, 4vw, 2.5rem);
        background:
          radial-gradient(
            60rem 30rem at 10% 0%,
            color-mix(in oklab, var(--accent) 10%, transparent),
            transparent 65%
          ),
          var(--color-bg);
      }
      .auth-shell {
        display: grid;
        grid-template-columns: 1fr;
        width: 100%;
        max-width: 64rem;
        overflow: hidden;
        border: 1px solid var(--color-border);
        border-radius: 1.75rem;
        background: var(--color-surface);
        box-shadow: var(--shadow-xl);
      }

      .auth-aside {
        position: relative;
        display: none;
        flex-direction: column;
        overflow: hidden;
        padding: 2.5rem;
        color: #ecfdf5;
        background: linear-gradient(160deg, #064e3b, #065f46 52%, #0f766e);
      }
      .aside-orb {
        position: absolute;
        border-radius: 999px;
        filter: blur(48px);
        opacity: 0.5;
        pointer-events: none;
      }
      .orb-one {
        top: -4rem;
        right: -3rem;
        width: 16rem;
        height: 16rem;
        background: #34d399;
      }
      .orb-two {
        bottom: -6rem;
        left: -4rem;
        width: 18rem;
        height: 18rem;
        background: #0d9488;
      }
      .aside-brand {
        position: relative;
        display: flex;
        align-items: center;
        gap: 0.6rem;
        font-size: 1.25rem;
        font-weight: 800;
        letter-spacing: -0.04em;
      }
      .aside-mark {
        display: grid;
        place-items: center;
        width: 2.25rem;
        height: 2.25rem;
        border-radius: 0.7rem;
        background: rgb(255 255 255 / 0.14);
        border: 1px solid rgb(255 255 255 / 0.18);
        color: #d1fae5;
      }
      .aside-copy {
        position: relative;
        margin-top: clamp(3rem, 7vh, 5rem);
      }
      .aside-eyebrow {
        display: block;
        margin-bottom: 0.5rem;
        font-size: 0.7rem;
        font-weight: 700;
        letter-spacing: 0.18em;
        text-transform: uppercase;
        color: #a7f3d0;
      }
      .aside-copy h2 {
        margin: 0 0 0.75rem;
        font-size: clamp(1.8rem, 3vw, 2.35rem);
        line-height: 1.08;
        font-weight: 750;
        letter-spacing: -0.035em;
        color: #fff;
      }
      .aside-copy p {
        max-width: 26rem;
        margin: 0;
        font-size: 0.875rem;
        line-height: 1.7;
        color: rgb(236 253 245 / 0.72);
      }
      .aside-points {
        position: relative;
        display: grid;
        gap: 1.1rem;
        margin: auto 0 0;
        padding: 2.5rem 0 0;
        list-style: none;
      }
      .aside-points li {
        display: flex;
        align-items: center;
        gap: 0.75rem;
      }
      .aside-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 2.4rem;
        height: 2.4rem;
        border-radius: 0.8rem;
        background: rgb(255 255 255 / 0.1);
        border: 1px solid rgb(255 255 255 / 0.14);
        color: #a7f3d0;
      }
      .aside-points li > span:last-child {
        display: flex;
        flex-direction: column;
        font-size: 0.75rem;
        color: rgb(236 253 245 / 0.62);
      }
      .aside-points strong {
        font-size: 0.875rem;
        font-weight: 600;
        color: #fff;
      }
      .aside-foot {
        position: relative;
        margin: 2rem 0 0;
        font-size: 0.75rem;
        color: rgb(236 253 245 / 0.6);
      }
      .aside-foot a {
        color: #a7f3d0;
        font-weight: 600;
        text-decoration: none;
      }
      .aside-foot a:hover {
        text-decoration: underline;
      }

      .auth-panel {
        display: grid;
        place-items: center;
        padding: clamp(1.5rem, 3.5vw, 2.5rem);
        background: var(--color-surface);
      }
      .auth-card {
        width: 100%;
        max-width: 24rem;
      }
      .auth-mobile-logo {
        margin-bottom: 1.5rem;
      }
      .auth-header h1 {
        margin: 0.2rem 0 0.4rem;
        font-size: clamp(1.7rem, 3vw, 2rem);
        font-weight: 750;
        letter-spacing: -0.03em;
      }
      .auth-header p {
        margin: 0;
        font-size: 0.875rem;
        color: var(--color-text-muted);
      }
      .auth-alert {
        display: flex;
        align-items: flex-start;
        gap: 0.5rem;
        margin-top: 1.25rem;
        padding: 0.8rem 0.9rem;
        border: 1px solid transparent;
        border-radius: 0.8rem;
        font-size: 0.8125rem;
        line-height: 1.5;
      }
      .auth-alert .icon {
        flex: 0 0 auto;
        margin-top: 0.05rem;
      }
      .auth-alert a {
        font-weight: 600;
        color: inherit;
      }
      .auth-alert.error {
        color: var(--color-danger);
        background: var(--color-danger-bg);
        border-color: color-mix(in oklab, var(--destructive) 35%, transparent);
      }
      .auth-alert.success {
        color: var(--color-accent);
        background: color-mix(in oklab, var(--accent) 10%, transparent);
        border-color: color-mix(in oklab, var(--accent) 30%, transparent);
      }
      .auth-form {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        margin-top: 1.5rem;
      }
      .check {
        display: inline-flex;
        align-items: center;
        gap: 0.45rem;
        color: var(--color-text-muted);
        cursor: pointer;
      }
      .check input {
        width: 0.95rem;
        height: 0.95rem;
        accent-color: var(--color-accent);
      }
      .terms {
        align-items: flex-start;
        font-size: 0.75rem;
        line-height: 1.5;
      }
      .terms input {
        margin-top: 0.15rem;
        flex: 0 0 auto;
      }
      .terms a {
        color: var(--color-accent);
        font-weight: 600;
        text-decoration: none;
      }
      .terms a:hover {
        text-decoration: underline;
      }
      .divider {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin: 1.5rem 0;
      }
      .divider::before,
      .divider::after {
        content: '';
        flex: 1;
        height: 1px;
        background: var(--color-border);
      }
      .divider span {
        font-size: 0.6875rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--color-text-muted);
      }
      .social-login {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 0.5rem;
      }
      .social-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.45rem;
        padding: 0.7rem 0.8rem;
        border: 1px solid var(--color-border);
        border-radius: 0.8rem;
        background: transparent;
        color: var(--color-text);
        font: inherit;
        font-size: 0.8125rem;
        font-weight: 600;
        cursor: pointer;
        transition:
          background 0.2s,
          border-color 0.2s;
      }
      .social-btn:hover:not(:disabled) {
        background: var(--color-surface-muted);
        border-color: color-mix(in oklab, var(--accent) 30%, var(--color-border));
      }
      .social-btn:disabled {
        opacity: 0.45;
        cursor: not-allowed;
      }
      .social-g {
        font-size: 0.95rem;
        font-weight: 800;
        color: #4285f4;
      }
      .auth-footer {
        margin: 1.5rem 0 0;
        text-align: center;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .auth-footer a {
        color: var(--color-accent);
        font-weight: 600;
        text-decoration: none;
      }
      .auth-footer a:hover {
        text-decoration: underline;
      }

      @media (min-width: 900px) {
        .auth-shell {
          grid-template-columns: 0.92fr 1.08fr;
        }
        .auth-aside {
          display: flex;
        }
        .auth-mobile-logo {
          display: none;
        }
      }
      @media (max-width: 899px) {
        .auth-page {
          padding: 0;
        }
        .auth-shell {
          min-height: 100vh;
          border: 0;
          border-radius: 0;
        }
        .auth-panel {
          padding: 2rem 1.25rem;
          align-items: start;
        }
      }
    `,
  ],
})
export class RegisterComponent implements OnInit {
  private readonly authService = inject(AuthService);
  private readonly authStore = inject(AuthStore);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly fb = inject(FormBuilder);

  returnUrl = signal<string>('/');
  agreeTerms = signal(false);
  emailHint = 'Nunca compartilharemos seu e-mail';

  static passwordMatchValidator(form: AbstractControl): ValidationErrors | null {
    const password = form.get('password')?.value;
    const confirmPassword = form.get('confirmPassword')?.value;
    if (password && confirmPassword && password !== confirmPassword) {
      return { passwordMismatch: true };
    }
    return null;
  }

  form = this.fb.nonNullable.group(
    {
      username: ['', [Validators.required, Validators.minLength(1), Validators.maxLength(200)]],
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(6), Validators.maxLength(72)]],
      confirmPassword: ['', [Validators.required]],
    },
    { validators: RegisterComponent.passwordMatchValidator },
  );
  loading = signal(false);
  error = signal<string>('');
  success = signal(false);

  usernameError = computed(() => {
    const control = this.form.get('username');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Username is required';
      if (control.errors['maxlength']) return 'Username must be 200 characters or less';
    }
    return '';
  });

  emailError = computed(() => {
    const control = this.form.get('email');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Email is required';
      if (control.errors['email']) return 'Enter a valid email address';
    }
    return '';
  });

  passwordError = computed(() => {
    const control = this.form.get('password');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Password is required';
      if (control.errors['minlength']) return 'Password must be at least 6 characters';
      if (control.errors['maxlength']) return 'Password must be 72 characters or less';
    }
    return '';
  });

  confirmPasswordError = computed(() => {
    const control = this.form.get('confirmPassword');
    const formErrors = this.form.errors;
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Please confirm your password';
    }
    if (formErrors?.['passwordMismatch'] && (control?.touched || control?.dirty)) {
      return 'Passwords do not match';
    }
    return '';
  });

  ngOnInit(): void {
    const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
    if (returnUrl) {
      this.returnUrl.set(returnUrl);
    }
    if (this.authStore.isAuthenticated()) {
      this.router.navigate([this.returnUrl()]);
    }
  }

  onSubmit(): void {
    if (this.form.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    const data: SignupRequest = {
      username: this.form.value.username!,
      email: this.form.value.email!,
      password: this.form.value.password!,
    };

    this.authService.signup(data).subscribe({
      next: () => {
        this.loading.set(false);
        this.success.set(true);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.message);
      },
    });
  }

  socialLogin(provider: string): void {
    // Social login placeholder
    console.log(`Social login with ${provider}`);
  }
}
