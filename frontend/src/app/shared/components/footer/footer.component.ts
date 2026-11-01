import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { LogoComponent } from '../logo/logo.component';

@Component({
  selector: 'app-footer',
  standalone: true,
  imports: [CommonModule, RouterLink, LogoComponent],
  template: `
    <footer class="site-footer" role="contentinfo">
      <div class="container">
        <div class="perks-grid">
          <div class="perk">
            <span class="perk-icon" aria-hidden="true"><span class="icon icon-truck"></span></span>
            <div><strong>Frete grátis</strong><span>Em pedidos acima de R$ 50</span></div>
          </div>
          <div class="perk">
            <span class="perk-icon" aria-hidden="true"
              ><span class="icon icon-rotate-ccw"></span
            ></span>
            <div><strong>Devoluções fáceis</strong><span>Prazo de 30 dias</span></div>
          </div>
          <div class="perk">
            <span class="perk-icon" aria-hidden="true"><span class="icon icon-shield"></span></span>
            <div><strong>Compra segura</strong><span>Pagamentos criptografados</span></div>
          </div>
          <div class="perk">
            <span class="perk-icon" aria-hidden="true"
              ><span class="icon icon-headset"></span
            ></span>
            <div>
              <strong>Estamos aqui para ajudar</strong><span>Atendimento 7 dias por semana</span>
            </div>
          </div>
        </div>

        <div class="footer-grid">
          <div class="footer-brand">
            <app-logo text="MiniMart" />
            <p>
              Produtos de qualidade, preços justos e entrega rápida. Tudo o que você precisa,
              selecionado com cuidado.
            </p>
            <div class="socials">
              <a href="#" class="social-link" aria-label="Facebook"
                ><span class="icon icon-facebook"></span
              ></a>
              <a href="#" class="social-link" aria-label="Twitter"
                ><span class="icon icon-twitter"></span
              ></a>
              <a href="#" class="social-link" aria-label="Instagram"
                ><span class="icon icon-instagram"></span
              ></a>
              <a href="#" class="social-link" aria-label="LinkedIn"
                ><span class="icon icon-linkedin"></span
              ></a>
            </div>
          </div>

          <nav class="footer-col" aria-label="Loja">
            <h5>Loja</h5>
            <a routerLink="/products">Todos os produtos</a>
            <a routerLink="/categories">Categorias</a>
            <!-- <a routerLink="/brands">Marcas</a> -->
            <a routerLink="/wishlist">Lista de desejos</a>
          </nav>

          <nav class="footer-col" aria-label="Atendimento ao cliente">
            <h5>Suporte</h5>
            <a routerLink="/cart">Seu carrinho</a>
            <a routerLink="/orders">Pedidos</a>
            <a routerLink="/notifications">Notificações</a>
            <a routerLink="/profile">Conta</a>
          </nav>

          <div class="footer-col contact">
            <h5>Contato</h5>
            <p>
              <span class="icon icon-sm icon-map-pin" aria-hidden="true"></span> Av. Paulista, 1000
              — Bela Vista, São Paulo — SP, 01310-100
            </p>
            <a href="tel:+551140028922"
              ><span class="icon icon-sm icon-phone" aria-hidden="true"></span> +55 (11)
              4002-8922</a
            >
            <a href="mailto:suporte@minimart.com.br"
              ><span class="icon icon-sm icon-mail" aria-hidden="true"></span>
              suporte@minimart.com.br</a
            >
            <p><span class="icon icon-sm icon-clock" aria-hidden="true"></span> Seg–Sex, 9h–18h</p>
          </div>
        </div>

        <div class="footer-bottom">
          <p>© {{ currentYear }} MiniMart. Todos os direitos reservados.</p>
          <div class="payment-badges" aria-label="Formas de pagamento aceitas">
            <span class="pay-badge">Visa</span>
            <span class="pay-badge">Mastercard</span>
            <span class="pay-badge">Amex</span>
            <span class="pay-badge">PayPal</span>
          </div>
        </div>
      </div>
    </footer>
  `,
  styles: [
    `
      .site-footer {
        margin-top: auto;
        padding: clamp(2.5rem, 5vw, 4rem) 0 1.5rem;
        border-top: 1px solid var(--border);
        background: color-mix(in oklab, var(--muted) 55%, var(--background));
      }
      .perks-grid {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 1rem;
        padding-bottom: 2.25rem;
        margin-bottom: 2.25rem;
        border-bottom: 1px solid var(--border);
      }
      .perk {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        min-width: 0;
      }
      .perk-icon {
        display: grid;
        place-items: center;
        width: 2.5rem;
        height: 2.5rem;
        flex: 0 0 auto;
        border-radius: var(--radius-lg);
        background: var(--color-primary-bg);
        color: var(--accent-strong);
      }
      .perk-icon .icon {
        width: 1.15rem;
        height: 1.15rem;
      }
      .perk strong {
        display: block;
        font-size: 0.82rem;
        font-weight: 650;
      }
      .perk div span {
        display: block;
        color: var(--muted-foreground);
        font-size: 0.74rem;
      }
      .footer-grid {
        display: grid;
        grid-template-columns: 2fr 1fr 1fr 1.3fr;
        gap: clamp(1.5rem, 4vw, 3.5rem);
      }
      .footer-brand p {
        max-width: 34ch;
        margin: 1rem 0;
        color: var(--muted-foreground);
        font-size: 0.85rem;
      }
      .socials {
        display: flex;
        gap: 0.45rem;
      }
      .social-link {
        display: grid;
        place-items: center;
        width: 2rem;
        height: 2rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-full);
        background: var(--card);
        color: var(--muted-foreground);
      }
      .social-link .icon {
        width: 0.9rem;
        height: 0.9rem;
      }
      .social-link:hover {
        color: var(--accent-strong);
        border-color: color-mix(in oklab, var(--accent) 40%, var(--border));
      }
      .footer-col {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 0.55rem;
      }
      .footer-col h5 {
        margin-bottom: 0.45rem;
        font-size: 0.78rem;
        font-weight: 750;
        letter-spacing: 0.1em;
        text-transform: uppercase;
        color: var(--foreground);
      }
      .footer-col a,
      .footer-col p {
        display: flex;
        align-items: center;
        gap: 0.45rem;
        margin: 0;
        color: var(--muted-foreground);
        font-size: 0.83rem;
        line-height: 1.55;
      }
      .footer-col a:hover {
        color: var(--accent-strong);
      }
      .footer-bottom {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        flex-wrap: wrap;
        padding-top: 1.5rem;
        margin-top: 2.25rem;
        border-top: 1px solid var(--border);
      }
      .footer-bottom p {
        margin: 0;
        color: var(--muted-foreground);
        font-size: 0.78rem;
      }
      .payment-badges {
        display: flex;
        gap: 0.35rem;
        flex-wrap: wrap;
      }
      @media (max-width: 900px) {
        .perks-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }
        .footer-grid {
          grid-template-columns: 1fr 1fr;
        }
      }
      @media (max-width: 560px) {
        .perks-grid {
          grid-template-columns: 1fr;
        }
        .footer-grid {
          grid-template-columns: 1fr;
          gap: 1.75rem;
        }
        .footer-bottom {
          flex-direction: column;
          align-items: flex-start;
        }
      }
    `,
  ],
})
export class FooterComponent {
  currentYear = new Date().getFullYear();
}
