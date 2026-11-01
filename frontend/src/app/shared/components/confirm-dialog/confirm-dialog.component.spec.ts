import { TestBed } from '@angular/core/testing';
import { ConfirmDialogComponent } from './confirm-dialog.component';

describe('ConfirmDialogComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ConfirmDialogComponent] }).compileComponents();
  });

  it('renders nothing when open is false', () => {
    const f = TestBed.createComponent(ConfirmDialogComponent);
    f.componentRef.setInput('open', false);
    f.componentRef.setInput('title', 'T');
    f.componentRef.setInput('message', 'M');
    f.detectChanges();
    expect(f.nativeElement.querySelector('.overlay')).toBeNull();
  });

  it('emits close on overlay click and confirm on button click', () => {
    const f = TestBed.createComponent(ConfirmDialogComponent);
    f.componentRef.setInput('open', true);
    f.componentRef.setInput('title', 'T');
    f.componentRef.setInput('message', 'M');
    f.detectChanges();
    const closeSpy = vi.fn();
    const confirmSpy = vi.fn();
    f.componentInstance.close.subscribe(closeSpy);
    f.componentInstance.confirm.subscribe(confirmSpy);

    const overlay = f.nativeElement.querySelector('.overlay') as HTMLElement;
    overlay.click();
    expect(closeSpy).toHaveBeenCalled();

    const btn = Array.from(f.nativeElement.querySelectorAll('button')).find((b: unknown) =>
      (b as HTMLElement).textContent?.includes('Confirmar'),
    ) as HTMLElement;
    btn.click();
    expect(confirmSpy).toHaveBeenCalled();
  });
});
