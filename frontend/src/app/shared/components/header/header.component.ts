import {
  Component,
  inject,
  signal,
  NgZone,
  HostListener,
  ViewChild,
  ElementRef,
  OnInit,
  OnDestroy,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink, RouterLinkActive, NavigationEnd } from '@angular/router';
import { AuthStore } from '../../../core/auth/auth.store';
import { AuthService } from '../../../core/auth/auth.service';
import { CategoriesApiService } from '../../../features/catalog/categories-api.service';
import { Category } from '../../../shared/models/catalog.models';
import { AvatarApiService } from '../../../features/customer/avatar-api.service';
import { OrdersApiService } from '../../../features/orders/orders-api.service';
import { CartStore } from '../../../features/cart/cart.store';
import { CartService } from '../../../features/cart/cart.service';
import { Avatar } from '../../../shared/models/avatar.models';
import { LogoComponent } from '../logo/logo.component';
import { ToastHostComponent } from '../toast-host/toast-host.component';
import { filter, catchError, of, takeUntil } from 'rxjs';
import { Subject } from 'rxjs';

@Component({
  selector: 'app-header',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, LogoComponent, ToastHostComponent],
  template: `
    <header class="site-header" [class.scrolled]="isScrolled()" role="banner">
      @if (isRefreshing()) {
        <div
          class="refresh-bar"
          role="progressbar"
          aria-valuemin="0"
          aria-valuemax="100"
          aria-label="Renovando token de autenticação"
        ></div>
      }
      <div class="announcement">
        <div class="container announcement-inner">
          <span
            ><span class="icon icon-sm icon-truck" aria-hidden="true"></span> Frete grátis em
            pedidos acima de R$ 50</span
          >
          <span class="announcement-right"
            ><span class="icon icon-sm icon-shield" aria-hidden="true"></span> Devoluções fáceis em
            30 dias</span
          >
        </div>
      </div>

      <div class="header-inner container">
        <a class="brand" routerLink="/" aria-label="Página inicial da MiniMart">
          <app-logo />
        </a>

        <button
          class="menu-toggle"
          type="button"
          (click)="mobileMenuOpen.update((v) => !v)"
          [attr.aria-expanded]="mobileMenuOpen()"
          aria-label="Alternar navegação"
          aria-controls="site-nav"
        >
          <span
            [class]="mobileMenuOpen() ? 'icon icon-x' : 'icon icon-menu'"
            aria-hidden="true"
          ></span>
        </button>

        <div class="nav-wrap" id="site-nav" [class.open]="mobileMenuOpen()">
          <nav class="primary-nav" aria-label="Navegação principal">
            <a
              class="nav-link"
              routerLink="/"
              routerLinkActive="active-link"
              [routerLinkActiveOptions]="{ exact: true }"
              >Home</a
            >
            <a class="nav-link" routerLink="/products" routerLinkActive="active-link">Produtos</a>
            <div class="nav-item d-flex category-menu">
              <button
                #categoryTrigger
                class="nav-link category-trigger"
                [class.active-link]="showCategories()"
                type="button"
                (click)="toggleCategories()"
                [attr.aria-expanded]="showCategories()"
                aria-haspopup="menu"
                aria-controls="categories-panel"
              >
                Categorias
                <span
                  class="icon icon-sm icon-chevron-down category-chevron"
                  [class.open]="showCategories()"
                  aria-hidden="true"
                ></span>
              </button>
              @if (showCategories()) {
                <div
                  id="categories-panel"
                  class="categories-panel"
                  role="menu"
                  aria-label="Comprar por categoria"
                >
                  <div class="container categories-panel-inner">
                    <div class="categories-intro">
                      <span class="eyebrow">Explorar a loja</span>
                      <h2>Comprar por categoria</h2>
                      <p>
                        Encontre exatamente o que precisa, dos itens essenciais às novidades em
                        tecnologia.
                      </p>
                      <!-- <a class="categories-all" [routerLink]="['/products']" [queryParams]="{ category_id: null }" role="menuitem">Todas as categorias <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span></a> -->
                    </div>
                    <div class="categories-grid">
                      @for (category of categories(); track category.id; let i = $index) {
                        <a
                          class="category-tile"
                          [routerLink]="['/products']"
                          [queryParams]="{ category_id: category.id }"
                          role="menuitem"
                        >
                          <span class="category-tile-icon"
                            ><span [class]="'icon ' + category.icon" aria-hidden="true"></span
                          ></span>
                          <span class="category-tile-copy">
                            <strong>{{ category.name }}</strong>
                            <small>Comprar agora</small>
                          </span>
                          <span
                            class="icon icon-sm icon-arrow-right category-tile-arrow"
                            aria-hidden="true"
                          ></span>
                        </a>
                      }
                    </div>
                  </div>
                </div>
              }
            </div>
            <!-- <a class="nav-link" routerLink="/brands" routerLinkActive="active-link">Marcas</a> -->
          </nav>

          <div class="header-actions" aria-label="Ações do usuário">
            <a class="icon-button" routerLink="/orders" aria-label="Pedidos">
              <span class="icon icon-truck" aria-hidden="true"></span>
              @if (orderCount() > 0) {
                <span class="count-pill">{{ orderCount() }}</span>
              }
            </a>
            <a class="icon-button" routerLink="/wishlist" aria-label="Lista de desejos">
              <span class="icon icon-heart" aria-hidden="true"></span>
              @if (wishlistCount > 0) {
                <span class="count-pill">{{ wishlistCount }}</span>
              }
            </a>
            <a class="icon-button" routerLink="/cart" aria-label="Carrinho">
              <span class="icon icon-cart" aria-hidden="true"></span>
              @if (cartCount() > 0) {
                <span class="count-pill">{{ cartCount() }}</span>
              }
            </a>
          </div>

          <div class="user-info account-area" aria-label="User account">
            @if (isAuthenticated()) {
              <button
                class="account-button"
                type="button"
                (click)="toggleShow()"
                [attr.aria-expanded]="isClick()"
              >
                @if (avatarUrl()) {
                  <img
                    class="account-avatar"
                    [src]="avatarUrl()!"
                    [alt]="'Avatar de ' + (authStore.user()?.username || 'Usuário')"
                  />
                } @else {
                  <span class="account-avatar" aria-hidden="true">{{
                    (authStore.user()?.username || 'U').charAt(0).toUpperCase()
                  }}</span>
                }
                <span class="account-name">{{ authStore.user()?.username || 'User' }}</span>
                <span class="icon icon-sm icon-chevron-down" aria-hidden="true"></span>
              </button>
              @if (isClick()) {
                <div #settings class="account-menu" role="menu">
                  <a routerLink="/profile" class="account-link" role="menuitem">
                    <span class="icon icon-sm icon-user" aria-hidden="true"></span> Perfil
                  </a>
                  <a routerLink="/notifications" class="account-link" role="menuitem">
                    <span class="icon icon-sm icon-bell" aria-hidden="true"></span> Notificações
                  </a>
                  <button
                    type="button"
                    class="account-link danger"
                    (click)="logout()"
                    role="menuitem"
                  >
                    <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span> Log out
                  </button>
                </div>
              }
            } @else {
              <a routerLink="/login" class="login-link" routerLinkActive="active-link">Entrar</a>
              <a routerLink="/register" class="signup-link" routerLinkActive="active-link"
                >Começar</a
              >
            }
          </div>
        </div>
      </div>
    </header>
    <app-toast-host />
  `,
  styles: [
    `
      :host {
        display: block;
      }
      .site-header {
        position: sticky;
        top: 0;
        z-index: 900;
        background: color-mix(in oklab, var(--background) 82%, transparent);
        border-bottom: 1px solid color-mix(in oklab, var(--border) 80%, transparent);
        -webkit-backdrop-filter: blur(20px) saturate(160%);
        backdrop-filter: blur(20px) saturate(160%);
        transition:
          box-shadow var(--transition),
          background-color var(--transition);
      }
      .site-header.scrolled {
        box-shadow: var(--shadow-md);
        background: color-mix(in oklab, var(--background) 92%, transparent);
      }
      .refresh-bar {
        position: absolute;
        inset: auto 0 0;
        height: 2px;
        background: linear-gradient(90deg, transparent, var(--accent), transparent);
        animation: refresh-pulse 1s ease-in-out infinite;
      }
      @keyframes refresh-pulse {
        0%,
        100% {
          opacity: 0.45;
        }
        50% {
          opacity: 1;
        }
      }
      .announcement {
        background: color-mix(in oklab, var(--accent) 9%, var(--background));
        border-bottom: 1px solid color-mix(in oklab, var(--accent) 16%, transparent);
        color: var(--accent-strong);
        font-size: 0.72rem;
        font-weight: 600;
        letter-spacing: 0.01em;
      }
      .announcement-inner {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        min-height: 2rem;
      }
      .announcement span {
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
      }
      .announcement-right {
        color: color-mix(in oklab, var(--accent-strong) 80%, var(--muted-foreground));
      }
      .header-inner {
        display: flex;
        align-items: center;
        gap: 1.4rem;
        min-height: 4.35rem;
      }
      .brand {
        flex: 0 0 auto;
      }
      .menu-toggle {
        display: none;
        place-items: center;
        width: 2.5rem;
        height: 2.5rem;
        margin-left: auto;
        border: 1px solid var(--border);
        border-radius: var(--radius-md);
        background: var(--card);
        color: var(--foreground);
        cursor: pointer;
      }
      .nav-wrap {
        display: flex;
        align-items: center;
        flex: 1;
        gap: 1.2rem;
        min-width: 0;
      }
      .primary-nav {
        display: flex;
        align-items: center;
        gap: 0.15rem;
      }
      .nav-link {
        display: inline-flex;
        align-items: center;
        min-height: 2.25rem;
        padding: 0.35rem 0.7rem;
        border-radius: var(--radius-full);
        color: var(--muted-foreground);
        font-size: 0.84rem;
        font-weight: 550;
        text-decoration: none;
        transition:
          color var(--transition-fast),
          background-color var(--transition-fast);
      }
      .nav-link:hover {
        color: var(--foreground);
        background: var(--muted);
      }
      .active-link {
        color: var(--accent-strong);
        background: var(--color-primary-bg);
      }
      .category-menu {
        display: flex;
        position: static;
        align-items: center;
        gap: 0.1rem;
      }
      .category-trigger {
        gap: 0.3rem;
        border: 0;
        background: transparent;
        font: inherit;
        cursor: pointer;
      }
      .category-chevron {
        transition: transform var(--transition-fast);
      }
      .category-chevron.open {
        transform: rotate(180deg);
      }
      .category-trigger:focus-visible {
        outline: 2px solid var(--accent);
        outline-offset: 2px;
      }
      .header-actions {
        display: flex;
        align-items: center;
        gap: 0.35rem;
        margin-left: auto;
      }
      .icon-button {
        position: relative;
        display: grid;
        place-items: center;
        width: 2.4rem;
        height: 2.4rem;
        border-radius: var(--radius-full);
        color: var(--foreground);
        transition:
          background-color var(--transition-fast),
          color var(--transition-fast);
      }
      .icon-button .icon {
        width: 1.2rem;
        height: 1.2rem;
      }
      .icon-button:hover {
        background: var(--muted);
        color: var(--accent-strong);
      }
      .count-pill {
        position: absolute;
        top: -0.15rem;
        right: -0.15rem;
      }
      .account-area {
        position: relative;
        display: flex;
        align-items: center;
        gap: 0.5rem;
      }
      .account-button {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        height: 2.5rem;
        padding: 0.2rem 0.6rem 0.2rem 0.25rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-full);
        background: var(--card);
        color: var(--foreground);
        cursor: pointer;
      }
      .account-button:hover {
        border-color: color-mix(in oklab, var(--accent) 40%, var(--border));
      }
      .account-avatar {
        display: grid;
        place-items: center;
        width: 1.9rem;
        height: 1.9rem;
        border-radius: 50%;
        background: linear-gradient(
          145deg,
          var(--accent),
          color-mix(in oklab, var(--accent) 45%, #0f766e)
        );
        color: var(--accent-foreground);
        font-size: 0.75rem;
        font-weight: 800;
        overflow: hidden;
      }
      .account-avatar img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        border-radius: 50%;
      }
      .account-name {
        max-width: 7rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 0.8rem;
        font-weight: 600;
      }
      .account-menu {
        position: absolute;
        top: calc(100% + 0.55rem);
        right: 0;
        z-index: 30;
        width: 12.5rem;
        padding: 0.4rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-lg);
        background: var(--popover);
        box-shadow: var(--shadow-lg);
        animation: scale-in 0.16s ease both;
      }
      .account-link {
        display: flex;
        align-items: center;
        gap: 0.55rem;
        width: 100%;
        padding: 0.55rem 0.6rem;
        border: 0;
        border-radius: var(--radius-sm);
        background: transparent;
        color: var(--popover-foreground);
        font-size: 0.82rem;
        font-weight: 500;
        text-align: left;
        cursor: pointer;
      }
      .account-link:hover {
        background: var(--muted);
      }
      .account-link.danger {
        color: var(--destructive);
      }
      .login-link {
        padding: 0.45rem 0.6rem;
        color: var(--foreground);
        font-size: 0.82rem;
        font-weight: 600;
      }
      .signup-link {
        display: inline-flex;
        align-items: center;
        min-height: 2.3rem;
        padding: 0.35rem 0.9rem;
        border-radius: var(--radius-full);
        background: var(--accent);
        color: var(--accent-foreground);
        font-size: 0.8rem;
        font-weight: 700;
        box-shadow: 0 8px 18px -12px color-mix(in oklab, var(--accent) 80%, transparent);
      }
      .signup-link:hover {
        background: var(--accent-hover);
        color: var(--accent-foreground);
      }
      @media (max-width: 900px) {
        .header-inner {
          min-height: 3.8rem;
        }
        .menu-toggle {
          display: grid;
        }
        .nav-wrap {
          position: absolute;
          top: 100%;
          left: 0;
          right: 0;
          display: none;
          flex-direction: column;
          align-items: stretch;
          gap: 0.9rem;
          padding: 1rem clamp(1rem, 4vw, 2rem) 1.25rem;
          border-bottom: 1px solid var(--border);
          background: var(--background);
          box-shadow: var(--shadow-lg);
        }
        .nav-wrap.open {
          display: flex;
        }
        .primary-nav {
          flex-direction: column;
          align-items: stretch;
          gap: 0.1rem;
        }
        .nav-link {
          justify-content: space-between;
          min-height: 2.6rem;
          padding-inline: 0.75rem;
          border-radius: var(--radius-md);
        }
        .category-menu {
          flex-direction: column;
          align-items: stretch;
        }
        .category-trigger {
          justify-content: space-between;
          width: 100%;
        }
        .header-actions {
          justify-content: center;
          gap: 0.75rem;
          padding: 0.4rem 0;
          border-block: 1px solid var(--border);
        }
        .account-area {
          justify-content: center;
          padding-top: 0.25rem;
        }
        .account-menu {
          position: static;
          width: 100%;
          margin-top: 0.4rem;
          box-shadow: none;
          background: var(--muted);
        }
      }
      @media (max-width: 480px) {
        .announcement-right {
          display: none !important;
        }
        .account-name {
          max-width: 5rem;
        }
      }
    `,
  ],
})
export class HeaderComponent implements OnInit, OnDestroy {
  protected readonly authStore = inject(AuthStore);
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  private readonly ngZone = inject(NgZone);
  private readonly categoriesApi = inject(CategoriesApiService);
  private readonly avatarApi = inject(AvatarApiService);
  private readonly ordersApi = inject(OrdersApiService);
  private readonly cartStore = inject(CartStore);
  private readonly cartService = inject(CartService);
  @ViewChild('categoryTrigger') private categoryTrigger?: ElementRef<HTMLButtonElement>;

  isAuthenticated = this.authStore.isAuthenticated;
  isRefreshing = this.authStore.isRefreshing;
  mobileMenuOpen = signal(false);
  isClick = signal(false);
  showCategories = signal(false);
  isScrolled = signal(false);

  categories = signal<Category[]>([]);
  categoriesLoading = signal(true);
  categoriesError = signal<string | null>(null);

  orderCount = signal(0);
  wishlistCount = 0;
  cartCount = this.cartStore.itemCount;

  // Avatar state
  avatarUrl = signal<string | null>(null);
  private readonly destroy$ = new Subject<void>();

  constructor() {
    this.router.events.pipe(filter((e) => e instanceof NavigationEnd)).subscribe(() => {
      this.mobileMenuOpen.set(false);
      this.isClick.set(false);
      this.showCategories.set(false);
      this.loadOrderCount();
      this.loadCart();
    });
  }

  ngOnInit(): void {
    this.loadCategories();
    this.loadAvatar();
    this.loadOrderCount();
    this.loadCart();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadCategories(): void {
    this.categoriesLoading.set(true);
    this.categoriesError.set(null);
    this.categoriesApi
      .list()
      .pipe(
        catchError((err) => {
          this.categoriesError.set(err.message);
          this.categoriesLoading.set(false);
          return of([] as Category[]);
        }),
      )
      .subscribe((categories) => {
        this.categories.set(categories);
        this.categoriesLoading.set(false);
      });
  }

  loadAvatar(): void {
    if (!this.isAuthenticated()) return;
    const accountId = this.authStore.userId();
    if (!accountId) return;
    this.avatarApi
      .get(accountId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (avatar: Avatar) => {
          if (avatar.imageUrl) {
            this.avatarUrl.set(avatar.imageUrl);
          }
        },
        error: (err: unknown) => {
          const httpErr = err as { status?: number; message?: string };
          if (httpErr.status !== 404) {
            console.warn('Failed to load avatar:', httpErr.message);
          }
        },
      });
  }

  loadOrderCount(): void {
    if (!this.isAuthenticated()) {
      this.orderCount.set(0);
      return;
    }
    this.ordersApi
      .list(1, 1)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (response) => this.orderCount.set(response.meta.total),
        error: () => this.orderCount.set(0),
      });
  }

  loadCart(): void {
    if (!this.isAuthenticated()) {
      this.cartStore.clear();
      return;
    }
    if (this.cartStore.cart()) {
      return;
    }
    const hadCart = this.cartStore.cart();
    this.cartService
      .getCart()
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (cart) => {
          if (!this.cartStore.cart() || hadCart === null) {
            this.cartStore.setCart(cart);
          }
        },
        error: () => {
          if (!this.cartStore.cart()) {
            this.cartStore.setCart(null);
          }
        },
      });
  }

  @HostListener('window:scroll')
  onWindowScroll(): void {
    this.isScrolled.set(window.scrollY > 50);
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (!this.showCategories()) return;
    this.showCategories.set(false);
    this.categoryTrigger?.nativeElement.focus();
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: Event): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.user-info')) {
      this.isClick.set(false);
    }
    if (!target.closest('.nav-item.d-flex')) {
      this.showCategories.set(false);
    }
  }

  toggleShow(): void {
    this.isClick.update((v) => !v);
  }

  toggleCategories(): void {
    this.showCategories.update((v) => !v);
  }

  logout(): void {
    console.log('[HeaderComponent] Logout clicked');
    this.authService.logout().subscribe({
      next: () => {
        console.log('[HeaderComponent] Logout success, navigating to /');
        this.ngZone.run(() => {
          this.router.navigateByUrl('/').then((success) => {
            console.log('[HeaderComponent] Navigation result:', success);
          });
        });
      },
      error: (err) => {
        console.error('[HeaderComponent] Logout error:', err);
        this.ngZone.run(() => {
          this.router.navigateByUrl('/').then((success) => {
            console.log('[HeaderComponent] Navigation result (error path):', success);
          });
        });
      },
    });
    this.isClick.set(false);
    this.orderCount.set(0);
  }
}
