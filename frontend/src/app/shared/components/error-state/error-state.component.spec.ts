import { TestBed } from '@angular/core/testing';
import { ErrorStateComponent } from './error-state.component';

describe('ErrorStateComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ErrorStateComponent] }).compileComponents();
  });

  it('renders title and message', () => {
    const f = TestBed.createComponent(ErrorStateComponent);
    f.componentRef.setInput('title', 'Oops');
    f.componentRef.setInput('message', 'something broke');
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('Oops');
    expect(f.nativeElement.textContent).toContain('something broke');
  });

  it('shows retry button only when retryLabel truthy and emits retry', () => {
    const f = TestBed.createComponent(ErrorStateComponent);
    f.componentRef.setInput('title', 'Oops');
    f.componentRef.setInput('message', 'm');
    f.detectChanges();
    expect(f.nativeElement.querySelector('button')).toBeNull();

    f.componentRef.setInput('retryLabel', 'Retry');
    f.detectChanges();
    const spy = vi.fn();
    f.componentInstance.retry.subscribe(spy);
    (f.nativeElement.querySelector('button') as HTMLElement).click();
    expect(spy).toHaveBeenCalled();
  });
});
