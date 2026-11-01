import { TestBed } from '@angular/core/testing';
import { ProductImageComponent } from './product-image.component';

describe('ProductImageComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ProductImageComponent] }).compileComponents();
  });

  function create(name: string, imageUrl: string | null = null, productId = 'p1') {
    const f = TestBed.createComponent(ProductImageComponent);
    f.componentRef.setInput('productId', productId);
    f.componentRef.setInput('name', name);
    f.componentRef.setInput('imageUrl', imageUrl);
    f.detectChanges();
    return f;
  }

  it('monogram trims and uppercases first char', () => {
    expect(create('  alice  ').componentInstance.monogram()).toBe('A');
  });

  it('monogram empty for blank name', () => {
    expect(create('   ').componentInstance.monogram()).toBe('');
  });

  it('showImage false for null/empty url', () => {
    expect(create('A', null).componentInstance.showImage()).toBe(false);
    expect(create('A', '').componentInstance.showImage()).toBe(false);
  });

  it('showImage tracks failure per url', () => {
    const f = create('A', 'http://img/x.png');
    expect(f.componentInstance.showImage()).toBe(true);
    f.componentInstance.onImageError('http://img/x.png');
    f.detectChanges();
    expect(f.componentInstance.showImage()).toBe(false);
    // different url resets
    f.componentRef.setInput('imageUrl', 'http://img/y.png');
    f.detectChanges();
    expect(f.componentInstance.showImage()).toBe(true);
  });

  it('gradient is deterministic per productId', () => {
    const a = create('A', null, 'p1').componentInstance.gradient();
    const b = create('A', null, 'p1').componentInstance.gradient();
    expect(a).toBe(b);
    expect(a).toContain('linear-gradient');
  });
});
