import { TestBed } from '@angular/core/testing';
import { SkeletonComponent } from './skeleton.component';

describe('SkeletonComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [SkeletonComponent] }).compileComponents();
  });

  it('applies inputs to styles', () => {
    const f = TestBed.createComponent(SkeletonComponent);
    f.componentRef.setInput('variant', 'circular');
    f.componentRef.setInput('width', '2rem');
    f.componentRef.setInput('height', '2rem');
    f.detectChanges();
    const el = f.nativeElement.querySelector('.skeleton') as HTMLElement;
    expect(el.classList.contains('circular')).toBe(true);
    expect(el.style.width).toBe('2rem');
    expect(el.style.height).toBe('2rem');
  });
});
