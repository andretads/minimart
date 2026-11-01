import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { AuthStore } from '../../../core/auth/auth.store';
import { LoginRequest } from '../../../shared/models/auth.models';
import { ButtonComponent } from '../../../shared/components/button/button.component';
import { FormFieldComponent } from '../../../shared/components/form-field/form-field.component';
import { PageShellComponent } from '../../../shared/components/page-shell/page-shell.component';
import { LogoComponent } from '../../../shared/components/logo/logo.component';

@Component({
  selector: 'app-login',
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
              <span class="aside-eyebrow">Bem-vindo de volta</span>
              <h2>Que bom te ver novamente</h2>
              <p>
                Entre para continuar de onde parou — seu carrinho, pedidos e favoritos estão à sua
                espera.
              </p>
            </div>
            <ul class="aside-points">
              <li>
                <span class="aside-icon"
                  ><span class="icon icon-lock" aria-hidden="true"></span></span
                ><span><strong>Login seguro</strong>Sua conta sempre protegida.</span>
              </li>
              <li>
                <span class="aside-icon"
                  ><span class="icon icon-truck" aria-hidden="true"></span></span
                ><span><strong>Acompanhe seus pedidos</strong>Veja sua entrega em tempo real.</span>
              </li>
              <li>
                <span class="aside-icon"
                  ><span class="icon icon-heart" aria-hidden="true"></span></span
                ><span
                  ><strong>Salvos para depois</strong>Mantenha sua lista de desejos
                  sincronizada.</span
                >
              </li>
            </ul>
            <p class="aside-foot">Novo por aqui? <a routerLink="/register">Criar conta</a></p>
          </aside>

          <section class="auth-panel">
            <div class="auth-card">
              <div class="auth-mobile-logo"><app-logo text="MiniMart" /></div>
              <header class="auth-header">
                <span class="eyebrow">Conta</span>
                <h1>Entrar</h1>
                <p>Digite suas credenciais para acessar sua conta.</p>
              </header>

              @if (error()) {
                <div class="auth-alert error" role="alert">
                  <span class="icon icon-alert" aria-hidden="true"></span>
                  <span>{{ error() }}</span>
                </div>
              }

              <form [formGroup]="form" (ngSubmit)="onSubmit()" class="auth-form">
                <app-form-field
                  id="username"
                  formControlName="username"
                  label="Usuário"
                  type="text"
                  placeholder="Digite seu usuário"
                  [required]="true"
                  [error]="usernameError()"
                  [disabled]="loading()"
                  [hint]="'Seu nome de usuário cadastrado'"
                />
                <app-form-field
                  id="password"
                  formControlName="password"
                  label="Senha"
                  type="password"
                  placeholder="Digite sua senha"
                  [required]="true"
                  [error]="passwordError()"
                  [disabled]="loading()"
                  [hint]="'Mínimo de 6 caracteres'"
                />

                <div class="form-options">
                  <label class="check">
                    <input type="checkbox" (change)="rememberMe.set($event.target.checked)" />
                    <span>Lembrar-me</span>
                  </label>
                  <a routerLink="/forgot-password">Esqueceu a senha?</a>
                </div>

                <app-button
                  type="submit"
                  label="Entrar"
                  variant="primary"
                  size="lg"
                  [fullWidth]="true"
                  [loading]="loading()"
                  [disabled]="form.invalid || loading()"
                />

                @if (showCaptcha()) {
                  <div class="auth-alert warning">
                    <span class="icon icon-shield" aria-hidden="true"></span>
                    <span>Muitas tentativas falhadas. Aguarde antes de tentar novamente.</span>
                  </div>
                }
              </form>

              <!-- <div class="divider"><span>ou continue com</span></div>

              <div class="social-login">
                <button type="button" class="social-btn" (click)="socialLogin('google')" [disabled]="loading()">
                  <span class="social-g" aria-hidden="true">G</span> Google
                </button>
                <button type="button" class="social-btn" (click)="socialLogin('github')" [disabled]="loading()">
                  <span class="icon icon-github" aria-hidden="true"></span> GitHub
                </button>
              </div> -->

              <p class="auth-footer">
                Não tem uma conta?
                <a [routerLink]="'/register'" [queryParams]="{ returnUrl: returnUrl() }"
                  >Criar conta</a
                >
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
        margin-top: clamp(3rem, 8vh, 6rem);
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
        font-size: clamp(1.8rem, 3vw, 2.4rem);
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
        padding: clamp(1.5rem, 4vw, 3rem);
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
        font-size: clamp(1.75rem, 3vw, 2.15rem);
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
      .auth-alert.error {
        color: var(--color-danger);
        background: var(--color-danger-bg);
        border-color: color-mix(in oklab, var(--destructive) 35%, transparent);
      }
      .auth-alert.warning {
        color: var(--color-warning);
        background: var(--color-warning-bg);
        border-color: color-mix(in oklab, #f59e0b 35%, transparent);
      }
      .auth-form {
        display: flex;
        flex-direction: column;
        gap: 1.15rem;
        margin-top: 1.5rem;
      }
      .form-options {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        font-size: 0.8125rem;
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
      .form-options a {
        color: var(--color-accent);
        font-weight: 600;
        text-decoration: none;
      }
      .form-options a:hover {
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
export class LoginComponent implements OnInit {
  private readonly authService = inject(AuthService);
  private readonly authStore = inject(AuthStore);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  form = this.fb.nonNullable.group({
    username: ['', [Validators.required, Validators.minLength(1), Validators.maxLength(200)]],
    password: ['', [Validators.required, Validators.minLength(6), Validators.maxLength(72)]],
  });
  loading = signal(false);
  error = signal<string>('');
  returnUrl = signal('/');
  showCaptcha = signal(false);
  rememberMe = signal(false);

  usernameError = computed(() => {
    const control = this.form.get('username');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Username is required';
      if (control.errors['maxlength']) return 'Username must be 200 characters or less';
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

  ngOnInit(): void {
    this.route.queryParams.subscribe((params) => {
      this.returnUrl.set(params['returnUrl'] || '/');
    });
    if (this.authStore.isAuthenticated()) {
      this.router.navigate([this.returnUrl()]);
    }
  }

  onSubmit(): void {
    if (this.form.invalid || this.loading()) return;
    this.loading.set(true);
    this.error.set('');

    const credentials: LoginRequest = {
      ...this.form.getRawValue(),
      rememberMe: this.rememberMe(),
    };

    this.authService.login(credentials).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate([this.returnUrl()]);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.message);
        if (err.status === 429) this.showCaptcha.set(true);
      },
    });
  }

  socialLogin(provider: string): void {
    // Social login placeholder
    console.log(`Social login with ${provider}`);
  }
}
