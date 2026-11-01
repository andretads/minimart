import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { LogoComponent } from './logo.component';

describe('LogoComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LogoComponent],
      providers: [provideRouter([])],
    }).compileComponents();
  });

  it('renders default text and links home', () => {
    const f = TestBed.createComponent(LogoComponent);
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('MiniMart');
    expect(f.nativeElement.querySelector('a')?.getAttribute('href')).toBe('/');
  });

  it('renders custom text', () => {
    const f = TestBed.createComponent(LogoComponent);
    f.componentRef.setInput('text', 'X');
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('X');
  });
});
