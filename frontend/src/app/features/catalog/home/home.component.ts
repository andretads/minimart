import { Component, inject, signal, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { map, catchError, of, switchMap, Observable, distinctUntilChanged } from 'rxjs';
import { ShowcaseApiService } from '../showcase-api.service';
import { CategoriesApiService } from '../categories-api.service';
import { ProductCardComponent } from '../product-card/product-card.component';
import { ButtonComponent } from '@shared/components/button/button.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { Product, ProductListResponse, Category } from '@shared/models/catalog.models';
import { AuthStore } from '@core/auth/auth.store';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { ToastService } from '@core/toast/toast.service';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ProductCardComponent,
    ButtonComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
  ],
  template: `
    <app-page-shell title="Início" [header]="false">
      <!-- <section class="hero-section" aria-labelledby="hero-title">
        <div class="container hero-layout">
          <div class="hero-slider" role="region" aria-label="Featured collections slider">
            <div class="aurora" aria-hidden="true"></div>

            <div class="hero-slide" [class.active]="currentSlide() === 0">
              <div class="hero-content">
                <span class="eyebrow"><span class="icon icon-sm icon-sparkles" aria-hidden="true"></span> Nova temporada · 2026</span>
                <h1 id="hero-title" class="hero-title">Tudo o que você precisa,<br><span class="text-gradient">com preços incríveis.</span></h1>
                <p class="hero-subtitle">Essenciais selecionados para sua casa, guarda-roupa e dia a dia — entrega rápida, devolução fácil.</p>
                <div class="hero-actions">
                  <a routerLink="/products" class="hero-cta">Comprar a coleção <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span></a>
                  <a routerLink="/categories" class="hero-secondary">Explorar categorias</a>
                </div>
                <div class="hero-proof">
                  <span class="proof-icon" aria-hidden="true"><span class="icon icon-star"></span></span>
                  <div><strong>4,9/5</strong><span>de 2.400+ clientes satisfeitos</span></div>
                </div>
              </div>
              <div class="hero-visual" aria-hidden="true">
                <div class="visual-tile tile-main"><span class="icon icon-bag"></span><span>Essenciais<br>do dia a dia</span></div>
                <div class="visual-tile tile-a"><span class="icon icon-cat-1"></span></div>
                <div class="visual-tile tile-b"><span class="icon icon-cat-2"></span></div>
              </div>
            </div>

            <div class="hero-slide" [class.active]="currentSlide() === 1">
              <div class="hero-content">
                <span class="eyebrow"><span class="icon icon-sm icon-zap" aria-hidden="true"></span> Novas ofertas diariamente</span>
                <h2 class="hero-title">Novidades,<br><span class="text-gradient">ofertas exclusivas.</span></h2>
                <p class="hero-subtitle">Descubra produtos selecionados a preços que fazem sentido. Novidades e economias exclusivas para membros toda semana.</p>
                <div class="hero-actions">
                  <a routerLink="/products" class="hero-cta">Explorar novidades <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span></a>
                  <a routerLink="/wishlist" class="hero-secondary">Ver lista de desejos</a>
                </div>
                <div class="hero-proof">
                  <span class="proof-icon" aria-hidden="true"><span class="icon icon-gift"></span></span>
                  <div><strong>Membros economizam mais</strong><span>Ofertas exclusivas semanais</span></div>
                </div>
              </div>
              <div class="hero-visual alt" aria-hidden="true">
                <div class="visual-tile tile-main"><span class="icon icon-gift"></span><span>Ofertas<br>semanais</span></div>
                <div class="visual-tile tile-a"><span class="icon icon-cat-5"></span></div>
                <div class="visual-tile tile-b"><span class="icon icon-cat-4"></span></div>
              </div>
            </div>

            <div class="hero-slide" [class.active]="currentSlide() === 2">
              <div class="hero-content">
                <span class="eyebrow"><span class="icon icon-sm icon-truck" aria-hidden="true"></span> Entrega rápida e grátis</span>
                <h2 class="hero-title">Da nossa loja<br><span class="text-gradient">até sua porta.</span></h2>
                <p class="hero-subtitle">Frete grátis em pedidos acima de R$ 50. Compra simples, pagamentos seguros e devolução em 30 dias.</p>
                <div class="hero-actions">
                  <a routerLink="/products" class="hero-cta">Começar a comprar <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span></a>
                  <a routerLink="/orders" class="hero-secondary">Rastrear um pedido</a>
                </div>
                <div class="hero-proof">
                  <span class="proof-icon" aria-hidden="true"><span class="icon icon-shield"></span></span>
                  <div><strong>Compre com confiança</strong><span>Finalização segura, criptografada</span></div>
                </div>
              </div>
              <div class="hero-visual third" aria-hidden="true">
                <div class="visual-tile tile-main"><span class="icon icon-truck"></span><span>Entrega<br>rápida</span></div>
                <div class="visual-tile tile-a"><span class="icon icon-package"></span></div>
                <div class="visual-tile tile-b"><span class="icon icon-cat-3"></span></div>
              </div>
            </div>

            <div class="hero-controls">
              <button class="hero-arrow" type="button" (click)="goToSlide((currentSlide() + 2) % 3)" aria-label="Slide anterior">
                <span class="icon icon-sm icon-arrow-left" aria-hidden="true"></span>
              </button>
              <div class="hero-dots" aria-label="Slider navigation">
                <button class="slider-dot" [class.active]="currentSlide() === 0" (click)="goToSlide(0)" aria-label="Slide 1"></button>
                <button class="slider-dot" [class.active]="currentSlide() === 1" (click)="goToSlide(1)" aria-label="Slide 2"></button>
                <button class="slider-dot" [class.active]="currentSlide() === 2" (click)="goToSlide(2)" aria-label="Slide 3"></button>
              </div>
              <button class="hero-arrow" type="button" (click)="goToSlide((currentSlide() + 1) % 3)" aria-label="Próximo slide">
                <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span>
              </button>
            </div>
          </div>

          <aside class="hero-aside" aria-label="Store benefits">
            <div class="aside-card">
              <span class="aside-icon" aria-hidden="true"><span class="icon icon-truck"></span></span>
              <div><h3>Frete grátis</h3><p>Em pedidos acima de R$ 50, sem código.</p></div>
            </div>
            <div class="aside-card">
              <span class="aside-icon" aria-hidden="true"><span class="icon icon-rotate-ccw"></span></span>
              <div><h3>Devoluções em 30 dias</h3><p>Mudou de ideia? Devolva sem complicação.</p></div>
            </div>
            <div class="aside-card accent">
              <span class="aside-icon" aria-hidden="true"><span class="icon icon-sparkles"></span></span>
              <div><h3>Novidades toda semana</h3><p>Novidades e ofertas selecionadas para você.</p></div>
            </div>
          </aside>
        </div>
      </section> -->

      <section class="section categories-section" aria-labelledby="categories-heading">
        <div class="container">
          <div class="section-heading center">
            <div>
              <span class="eyebrow">Explorar a loja</span>
              <h2 id="categories-heading" class="section-title">Comprar por categoria</h2>
              <p class="section-subtitle">
                Encontre exatamente o que precisa, da tecnologia aos itens essenciais para casa.
              </p>
            </div>
          </div>

          @if (categoriesLoading()) {
            <div class="categories-grid" aria-busy="true" aria-label="Carregando categorias">
              @for (i of skeletonCategories; track i) {
                <div class="category-card skeleton">
                  <div class="category-icon skeleton"></div>
                  <div class="category-name skeleton"></div>
                  <div class="category-link skeleton"></div>
                </div>
              }
            </div>
          } @else if (categoriesError()) {
            <app-error-state
              title="Falha ao carregar categorias"
              [message]="categoriesError()!"
              retryLabel="Tentar novamente"
              (retry)="loadCategories()"
            />
          } @else if (categories().length === 0) {
            <app-empty-state
              title="Nenhuma categoria encontrada"
              message="Nenhuma categoria disponível no momento."
              icon="folder"
            />
          } @else {
            <div class="categories-grid">
              @for (category of categories(); track category.id; let i = $index) {
                <a
                  [routerLink]="['/products']"
                  [queryParams]="{ category_id: category.id }"
                  class="category-card"
                >
                  <span class="category-icon" aria-hidden="true"
                    ><span [class]="'icon ' + category.icon"></span
                  ></span>
                  <span class="category-name">{{ category.name }}</span>
                  <span class="category-link"
                    >Comprar agora
                    <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span
                  ></span>
                </a>
              }
            </div>
          }
        </div>
      </section>

      <section
        class="section products-section"
        aria-labelledby="products-heading"
        id="productsHome"
      >
        <div class="container">
          <div class="section-heading">
            <div>
              <span class="eyebrow">Selecionamos para você</span>
              <h2 id="products-heading" class="section-title">Os melhores produtos</h2>
            </div>
            <a [routerLink]="['/products']" [queryParams]="{ category_id: null }" class="view-all"
              >Ver todos os produtos
              <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span
            ></a>
          </div>

          @if (loading()) {
            <div class="products-grid" aria-busy="true" aria-label="Carregando produtos">
              @for (i of skeletonItems; track i) {
                <app-product-card [product]="skeletonProduct" [loading]="true" />
              }
            </div>
          } @else if (error()) {
            <app-error-state
              title="Falha ao carregar produtos"
              [message]="error()!"
              retryLabel="Tentar novamente"
              (retry)="reload()"
            />
          } @else if (products().length === 0) {
            <app-empty-state
              title="Nenhum produto encontrado"
              message="Nenhum produto disponível no momento."
              icon="bag"
            />
          } @else {
            <div class="products-grid">
              @for (product of products(); track product.id) {
                <app-product-card [product]="product" (addToCart)="onAddToCart($event)" />
              }
            </div>

            @if (hasMore()) {
              <div class="pagination-wrapper">
                <app-button
                  variant="outline"
                  size="md"
                  [disabled]="loadingMore()"
                  (clicked)="loadMore()"
                  [label]="loadingMore() ? 'Carregando...' : 'Carregar mais'"
                />
              </div>
            }
          }
        </div>
      </section>
    </app-page-shell>
  `,
  styles: [
    `
      .hero-section {
        padding: clamp(1.25rem, 3vw, 2.25rem) 0 0;
      }
      .hero-layout {
        display: grid;
        grid-template-columns: minmax(0, 1fr) 17.5rem;
        gap: 1.15rem;
        align-items: stretch;
      }
      .hero-slider {
        position: relative;
        min-height: 34rem;
        overflow: hidden;
        border: 1px solid var(--border);
        border-radius: var(--radius-2xl);
        background: linear-gradient(
          135deg,
          color-mix(in oklab, var(--accent) 7%, var(--card)),
          var(--card) 55%
        );
        box-shadow: var(--shadow-sm);
        isolation: isolate;
      }
      .hero-slide {
        position: relative;
        z-index: 1;
        display: none;
        align-items: center;
        gap: 2rem;
        min-height: 34rem;
        padding: clamp(2rem, 5vw, 4rem) clamp(1.4rem, 4vw, 3.5rem) 4.75rem;
      }
      .hero-slide.active {
        display: grid;
        grid-template-columns: minmax(0, 1.05fr) minmax(15rem, 0.95fr);
        animation: fade-up 0.5s ease both;
      }
      .hero-content {
        max-width: 35rem;
      }
      .hero-title {
        margin: 0.75rem 0 1.15rem;
        font-size: clamp(2.4rem, 5vw, 4.15rem);
        font-weight: 800;
        line-height: 1.05;
        letter-spacing: -0.055em;
      }
      .hero-subtitle {
        max-width: 34rem;
        margin-bottom: 1.6rem;
        color: var(--muted-foreground);
        font-size: clamp(0.95rem, 1.5vw, 1.05rem);
      }
      .hero-actions {
        display: flex;
        align-items: center;
        gap: 0.8rem;
        flex-wrap: wrap;
      }
      .hero-cta {
        display: inline-flex;
        align-items: center;
        gap: 0.45rem;
        min-height: 3rem;
        padding: 0.6rem 1.35rem;
        border-radius: var(--radius-full);
        background: var(--accent);
        color: var(--accent-foreground);
        font-size: 0.88rem;
        font-weight: 700;
        box-shadow: var(--shadow-accent);
        transition:
          background-color var(--transition-fast),
          transform var(--transition-fast);
      }
      .hero-cta:hover {
        background: var(--accent-hover);
        color: var(--accent-foreground);
        transform: translateY(-1px);
      }
      .hero-secondary {
        color: var(--foreground);
        font-size: 0.85rem;
        font-weight: 650;
        padding: 0.6rem 0.9rem;
        border-bottom: 1px solid var(--border);
      }
      .hero-secondary:hover {
        color: var(--accent-strong);
        border-color: var(--accent);
      }
      .hero-proof {
        display: flex;
        align-items: center;
        gap: 0.8rem;
        margin-top: 2.1rem;
      }
      .hero-proof strong {
        display: block;
        font-size: 0.82rem;
      }
      .hero-proof span {
        color: var(--muted-foreground);
        font-size: 0.75rem;
      }
      .proof-icon {
        display: grid;
        place-items: center;
        width: 2.2rem;
        height: 2.2rem;
        border-radius: var(--radius-lg);
        background: var(--color-primary-bg);
        color: var(--accent-strong);
      }
      .hero-visual {
        position: relative;
        min-height: 23rem;
      }
      .visual-tile {
        position: absolute;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 0.7rem;
        border: 1px solid color-mix(in oklab, var(--accent) 24%, var(--border));
        border-radius: var(--radius-2xl);
        color: var(--accent-foreground);
        box-shadow: var(--shadow-lg);
      }
      .visual-tile .icon {
        width: 2.4rem;
        height: 2.4rem;
      }
      .tile-main {
        top: 50%;
        left: 50%;
        z-index: 2;
        width: 12.5rem;
        height: 15.5rem;
        background: linear-gradient(150deg, var(--accent), #0f766e 75%);
        transform: translate(-50%, -50%) rotate(-5deg);
        animation: float 7s ease-in-out infinite;
      }
      .tile-main span:last-child {
        font-size: 0.82rem;
        font-weight: 650;
        line-height: 1.35;
        text-align: center;
      }
      .tile-a {
        top: 0.5rem;
        right: 0.5rem;
        width: 7rem;
        height: 7rem;
        background: linear-gradient(150deg, oklch(0.72 0.14 170), oklch(0.56 0.13 190));
        transform: rotate(8deg);
      }
      .tile-b {
        bottom: 0.5rem;
        left: 0;
        width: 6rem;
        height: 6rem;
        background: linear-gradient(150deg, oklch(0.8 0.12 80), oklch(0.68 0.16 55));
        transform: rotate(-10deg);
      }
      .hero-visual.alt .tile-main {
        background: linear-gradient(150deg, oklch(0.7 0.17 45), oklch(0.55 0.17 25));
      }
      .hero-visual.alt .tile-a {
        background: linear-gradient(150deg, oklch(0.75 0.14 330), oklch(0.6 0.17 300));
      }
      .hero-visual.third .tile-main {
        background: linear-gradient(150deg, oklch(0.68 0.12 230), oklch(0.5 0.15 255));
      }
      .hero-visual.third .tile-b {
        background: linear-gradient(150deg, oklch(0.78 0.11 160), oklch(0.62 0.13 175));
      }
      .hero-controls {
        position: absolute;
        left: clamp(1.4rem, 4vw, 3.5rem);
        bottom: 1.3rem;
        z-index: 4;
        display: flex;
        align-items: center;
        gap: 0.7rem;
      }
      .hero-arrow {
        display: grid;
        place-items: center;
        width: 2rem;
        height: 2rem;
        border: 1px solid var(--border);
        border-radius: 50%;
        background: var(--card);
        color: var(--foreground);
        cursor: pointer;
      }
      .hero-arrow:hover {
        border-color: var(--accent);
        color: var(--accent-strong);
      }
      .hero-dots {
        display: flex;
        gap: 0.35rem;
      }
      .slider-dot {
        width: 1.55rem;
        height: 0.3rem;
        padding: 0;
        border: 0;
        border-radius: var(--radius-full);
        background: color-mix(in oklab, var(--muted-foreground) 30%, transparent);
        cursor: pointer;
        transition:
          width var(--transition),
          background-color var(--transition);
      }
      .slider-dot.active {
        width: 2.6rem;
        background: var(--accent);
      }
      .hero-aside {
        display: flex;
        flex-direction: column;
        gap: 0.85rem;
      }
      .aside-card {
        display: flex;
        flex: 1;
        align-items: flex-start;
        gap: 0.75rem;
        padding: 1.15rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-xl);
        background: var(--card);
        box-shadow: var(--shadow-xs);
        transition:
          transform var(--transition),
          box-shadow var(--transition);
      }
      .aside-card:hover {
        transform: translateY(-3px);
        box-shadow: var(--shadow-md);
      }
      .aside-card.accent {
        background: linear-gradient(
          150deg,
          color-mix(in oklab, var(--accent) 13%, var(--card)),
          var(--card)
        );
        border-color: color-mix(in oklab, var(--accent) 24%, var(--border));
      }
      .aside-icon {
        display: grid;
        place-items: center;
        width: 2.35rem;
        height: 2.35rem;
        flex: 0 0 auto;
        border-radius: var(--radius-lg);
        background: var(--color-primary-bg);
        color: var(--accent-strong);
      }
      .aside-card h3 {
        margin: 0 0 0.2rem;
        font-size: 0.88rem;
        font-weight: 700;
      }
      .aside-card p {
        margin: 0;
        color: var(--muted-foreground);
        font-size: 0.76rem;
        line-height: 1.5;
      }
      .categories-section {
        background: color-mix(in oklab, var(--muted) 45%, var(--background));
        border-block: 1px solid var(--border);
      }
      .section-heading.center {
        justify-content: center;
        text-align: center;
      }
      .section-heading.center .section-subtitle {
        margin-inline: auto;
      }
      .categories-grid {
        display: grid;
        grid-template-columns: repeat(6, minmax(0, 1fr));
        gap: 0.85rem;
      }
      .category-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.65rem;
        padding: 1.5rem 0.75rem 1.15rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-xl);
        background: var(--card);
        text-align: center;
        transition:
          transform var(--transition),
          border-color var(--transition),
          box-shadow var(--transition);
      }
      .category-card:hover {
        transform: translateY(-4px);
        border-color: color-mix(in oklab, var(--accent) 40%, var(--border));
        box-shadow: var(--shadow-md);
      }
      .category-icon {
        display: grid;
        place-items: center;
        width: 3.35rem;
        height: 3.35rem;
        border-radius: var(--radius-xl);
        background: var(--color-primary-bg);
        color: var(--accent-strong);
        transition:
          background-color var(--transition),
          color var(--transition);
      }
      .category-icon .icon {
        width: 1.45rem;
        height: 1.45rem;
      }
      .category-card:hover .category-icon {
        background: var(--accent);
        color: var(--accent-foreground);
      }
      .category-name {
        color: var(--foreground);
        font-size: 0.82rem;
        font-weight: 650;
      }
      .category-link {
        display: inline-flex;
        align-items: center;
        gap: 0.2rem;
        color: var(--muted-foreground);
        font-size: 0.68rem;
      }
      .category-card:hover .category-link {
        color: var(--accent-strong);
      }

      .category-card.skeleton {
        pointer-events: none;
      }
      .category-card.skeleton .category-icon.skeleton {
        width: 3.35rem;
        height: 3.35rem;
        border-radius: var(--radius-xl);
        background: linear-gradient(
          90deg,
          var(--muted) 25%,
          var(--muted-foreground) 37%,
          var(--muted) 63%
        );
        background-size: 400% 100%;
        animation: shimmer 1.4s ease infinite;
      }
      .category-card.skeleton .category-name.skeleton {
        height: 1rem;
        width: 80%;
        border-radius: var(--radius);
        background: linear-gradient(
          90deg,
          var(--muted) 25%,
          var(--muted-foreground) 37%,
          var(--muted) 63%
        );
        background-size: 400% 100%;
        animation: shimmer 1.4s ease infinite;
      }
      .category-card.skeleton .category-link.skeleton {
        height: 0.7rem;
        width: 60%;
        border-radius: var(--radius);
        background: linear-gradient(
          90deg,
          var(--muted) 25%,
          var(--muted-foreground) 37%,
          var(--muted) 63%
        );
        background-size: 400% 100%;
        animation: shimmer 1.4s ease infinite;
      }
      @keyframes shimmer {
        0% {
          background-position: 100% 50%;
        }
        100% {
          background-position: 0 50%;
        }
      }

      .view-all {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        color: var(--foreground);
        font-size: 0.82rem;
        font-weight: 650;
      }
      .view-all:hover {
        color: var(--accent-strong);
      }
      .products-grid {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 1.1rem;
      }
      .pagination-wrapper {
        display: flex;
        justify-content: center;
      }
      @media (max-width: 1100px) {
        .hero-layout {
          grid-template-columns: minmax(0, 1fr) 14.5rem;
        }
        .hero-aside {
          gap: 0.6rem;
        }
        .aside-card {
          flex-direction: column;
          padding: 1rem;
        }
        .hero-slide.active {
          grid-template-columns: minmax(0, 1fr) 13rem;
        }
        .tile-main {
          width: 10.5rem;
          height: 13.5rem;
        }
        .categories-grid {
          grid-template-columns: repeat(3, minmax(0, 1fr));
        }
        .products-grid {
          grid-template-columns: repeat(3, minmax(0, 1fr));
        }
      }
      @media (max-width: 860px) {
        .hero-layout {
          grid-template-columns: 1fr;
        }
        .hero-aside {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
        }
        .hero-aside .aside-card {
          flex-direction: row;
        }
        .hero-slider,
        .hero-slide {
          min-height: 31rem;
        }
        .hero-slide.active {
          grid-template-columns: 1fr;
        }
        .hero-visual {
          display: none;
        }
        .hero-content {
          max-width: 38rem;
        }
        .products-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }
      }
      @media (max-width: 620px) {
        .hero-slider,
        .hero-slide {
          min-height: 28rem;
        }
        .hero-title {
          font-size: 2.35rem;
        }
        .hero-aside {
          grid-template-columns: 1fr;
        }
        .categories-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }
        .products-grid {
          grid-template-columns: 1fr;
        }
      }
    `,
  ],
})
export class HomeComponent implements OnInit, OnDestroy {
  private readonly api = inject(ShowcaseApiService);
  private readonly categoriesApi = inject(CategoriesApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly authStore = inject(AuthStore);
  private readonly cartService = inject(CartService);
  private readonly cartStore = inject(CartStore);
  private readonly toast = inject(ToastService);

  products = signal<Product[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  hasMore = signal(false);
  nextCursor = signal<string | null>(null);
  loadingMore = signal(false);
  categoryId = signal<string | null>(null);

  categories = signal<Category[]>([]);
  categoriesLoading = signal(true);
  categoriesError = signal<string | null>(null);

  skeletonItems = Array.from({ length: 8 }, (_, i) => i);
  skeletonProduct = {
    id: 'skeleton',
    name: 'Carregando...',
    priceCents: 0,
  } as Product;

  skeletonCategories = Array.from({ length: 6 }, (_, i) => i);

  currentSlide = signal(0);
  slideInterval: ReturnType<typeof setInterval> | undefined;

  ngOnInit(): void {
    this.route.queryParams
      .pipe(
        map((params) => params['category_id'] || null),
        distinctUntilChanged(),
        switchMap((categoryId) => {
          this.categoryId.set(categoryId);
          this.products.set([]);
          this.nextCursor.set(null);
          this.hasMore.set(false);
          return this.loadProducts(categoryId);
        }),
      )
      .subscribe((response) => {
        this.products.set(response.data);
        this.hasMore.set(response.meta.hasMore ?? false);
        this.nextCursor.set(response.meta.nextCursor ?? null);
        this.loading.set(false);
      });

    this.loadCategories();
    this.startAutoSlide();
  }

  loadCategories(): void {
    this.categoriesLoading.set(true);
    this.categoriesError.set(null);
    this.categoriesApi.list().subscribe({
      next: (categories) => {
        this.categories.set(categories);
        this.categoriesLoading.set(false);
      },
      error: (err) => {
        this.categoriesError.set(err.message);
        this.categoriesLoading.set(false);
      },
    });
  }

  ngOnDestroy(): void {
    if (this.slideInterval) {
      clearInterval(this.slideInterval);
    }
  }

  startAutoSlide(): void {
    this.slideInterval = setInterval(() => {
      this.nextSlide();
    }, 5000);
  }

  nextSlide(): void {
    const next = (this.currentSlide() + 1) % 3;
    this.goToSlide(next);
  }

  goToSlide(index: number): void {
    this.currentSlide.set(index);
    // Reset interval
    if (this.slideInterval) {
      clearInterval(this.slideInterval);
    }
    this.startAutoSlide();
  }

  protected loadProducts(categoryId?: string | null): Observable<ProductListResponse> {
    this.loading.set(true);
    this.error.set(null);
    return this.api.list({ pageSize: 20, categoryId: categoryId || undefined }).pipe(
      catchError((err) => {
        this.error.set(err.message);
        this.loading.set(false);
        return of({ data: [], meta: { pageSize: 20, nextCursor: null, hasMore: false } });
      }),
    );
  }

  loadMore(): void {
    if (this.loadingMore() || !this.hasMore()) {
      return;
    }
    this.loadingMore.set(true);
    this.api
      .list({ pageSize: 20, cursor: this.nextCursor(), categoryId: this.categoryId() || undefined })
      .subscribe({
        next: (response) => {
          this.products.update((current) => [...current, ...response.data]);
          this.hasMore.set(response.meta.hasMore ?? false);
          this.nextCursor.set(response.meta.nextCursor ?? null);
          this.loadingMore.set(false);
        },
        error: (err: { message?: string }) => {
          this.toast.error(err.message || 'Não foi possível carregar mais produtos');
          this.loadingMore.set(false);
        },
      });
  }

  reload(): void {
    this.loading.set(true);
    this.error.set(null);
    this.products.set([]);
    this.nextCursor.set(null);
    this.hasMore.set(false);
    this.loadProducts(this.categoryId()).subscribe((response) => {
      this.products.set(response.data);
      this.hasMore.set(response.meta.hasMore ?? false);
      this.nextCursor.set(response.meta.nextCursor ?? null);
      this.loading.set(false);
    });
  }

  onAddToCart(productId: string): void {
    if (!this.authStore.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { returnUrl: this.router.url } });
      return;
    }
    this.cartService.addItem(productId, 1).subscribe({
      next: (cart) => {
        this.cartStore.setCart(cart);
        this.toast.success('Produto adicionado ao carrinho');
      },
      error: (err) => this.toast.error(err.message || 'Não foi possível adicionar ao carrinho'),
    });
  }
}
