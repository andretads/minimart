import { TestBed } from '@angular/core/testing';
import { SpinnerComponent } from './spinner.component';

describe('SpinnerComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [SpinnerComponent] }).compileComponents();
  });

  it('applies size and aria', () => {
    const f = TestBed.createComponent(SpinnerComponent);
    f.componentRef.setInput('size', '2rem');
    f.detectChanges();
    const el = f.nativeElement.querySelector('[role="status"]') as HTMLElement;
    expect(el.style.width).toBe('2rem');
    expect(el.style.height).toBe('2rem');
    expect(el.getAttribute('aria-label')).toBe('Carregando');
  });
});
