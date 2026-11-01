import { Component, input, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-product-image',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div
      class="product-image"
      [style.background]="gradient()"
      [style.aspect-ratio]="'1 / 1'"
      [attr.aria-label]="'Imagem do produto ' + name()"
      [attr.role]="showImage() ? null : 'img'"
    >
      @if (showImage()) {
        <img
          class="photo"
          [src]="imageUrl()"
          [alt]="'Imagem do produto ' + name()"
          loading="lazy"
          (error)="onImageError(imageUrl()!)"
        />
      } @else {
        <span class="monogram">{{ monogram() }}</span>
      }
    </div>
  `,
  styles: [
    `
      .product-image {
        position: relative;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 100%;
        overflow: hidden;
        border-radius: inherit;
        background-size: cover;
        background-position: center;
        isolation: isolate;
      }
      .product-image::after {
        content: '';
        position: absolute;
        inset: 0;
        background:
          radial-gradient(circle at 30% 20%, rgb(255 255 255 / 0.22), transparent 45%),
          linear-gradient(160deg, transparent 35%, rgb(0 0 0 / 0.18));
        mix-blend-mode: soft-light;
        pointer-events: none;
      }
      .photo {
        position: absolute;
        inset: 0;
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
      .monogram {
        position: relative;
        z-index: 1;
        font-size: clamp(2rem, 5vw, 4.25rem);
        font-weight: 800;
        color: rgb(255 255 255 / 0.92);
        text-shadow: 0 2px 20px rgb(0 0 0 / 0.18);
        letter-spacing: -0.04em;
      }
    `,
  ],
})
export class ProductImageComponent {
  productId = input.required<string>();
  name = input.required<string>();
  imageUrl = input<string | null | undefined>(null);

  private readonly failedUrl = signal<string | null>(null);

  monogram = computed(() => this.name().trim().charAt(0).toUpperCase());

  showImage = computed(() => {
    const url = this.imageUrl();
    return !!url && this.failedUrl() !== url;
  });

  onImageError(url: string): void {
    this.failedUrl.set(url);
  }

  gradient = computed(() => {
    const hash = this.hashString(this.productId());
    const hue1 = hash % 360;
    const hue2 = (hue1 + 60 + (hash % 60)) % 360;
    return `linear-gradient(135deg, hsl(${hue1} 65% 45%), hsl(${hue2} 65% 35%))`;
  });

  private hashString(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = (hash << 5) - hash + str.charCodeAt(i);
      hash |= 0;
    }
    return Math.abs(hash);
  }
}
