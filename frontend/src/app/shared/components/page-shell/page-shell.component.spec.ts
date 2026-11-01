import { TestBed } from '@angular/core/testing';
import { PageShellComponent } from './page-shell.component';

describe('PageShellComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [PageShellComponent] }).compileComponents();
  });

  it('renders header by default and footer when enabled', () => {
    const f = TestBed.createComponent(PageShellComponent);
    f.componentRef.setInput('title', 'T');
    f.componentRef.setInput('footer', true);
    f.detectChanges();
    expect(f.nativeElement.querySelector('header')).not.toBeNull();
    expect(f.nativeElement.querySelector('footer')).not.toBeNull();
  });

  it('hides header and adds bare class when header=false', () => {
    const f = TestBed.createComponent(PageShellComponent);
    f.componentRef.setInput('title', 'T');
    f.componentRef.setInput('header', false);
    f.detectChanges();
    expect(f.nativeElement.querySelector('header')).toBeNull();
    expect(f.nativeElement.querySelector('.page-shell')?.classList.contains('bare')).toBe(true);
  });
});
