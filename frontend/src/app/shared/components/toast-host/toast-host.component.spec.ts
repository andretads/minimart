import { TestBed } from '@angular/core/testing';
import { ToastService } from '@core/toast/toast.service';
import { ToastHostComponent } from './toast-host.component';

describe('ToastHostComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ToastHostComponent],
      providers: [ToastService],
    }).compileComponents();
  });

  it('renders toasts and dismisses', () => {
    const f = TestBed.createComponent(ToastHostComponent);
    const toast = TestBed.inject(ToastService) as ToastService;
    toast.show('hi', 'success', 0);
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('.toast').length).toBe(1);
    expect(f.nativeElement.textContent).toContain('hi');
    const btn = f.nativeElement.querySelector('button') as HTMLElement;
    btn.click();
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('.toast').length).toBe(0);
  });
});
