import { TestBed } from '@angular/core/testing';
import { ButtonComponent } from './button.component';

describe('ButtonComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ButtonComponent] }).compileComponents();
  });

  it('computes classes', () => {
    const f = TestBed.createComponent(ButtonComponent);
    f.componentRef.setInput('label', 'Go');
    f.componentRef.setInput('fullWidth', true);
    f.detectChanges();
    expect(f.componentInstance.computedClasses()).toBe('md primary full-width');
  });

  it('handleClick emits only when not disabled/loading', () => {
    const f = TestBed.createComponent(ButtonComponent);
    f.componentRef.setInput('label', 'Go');
    f.detectChanges();
    const spy = vi.fn();
    f.componentInstance.clicked.subscribe(spy);
    f.componentInstance.handleClick(new MouseEvent('click'));
    expect(spy).toHaveBeenCalled();

    f.componentRef.setInput('disabled', true);
    f.detectChanges();
    spy.mockClear();
    f.componentInstance.handleClick(new MouseEvent('click'));
    expect(spy).not.toHaveBeenCalled();

    f.componentRef.setInput('disabled', false);
    f.componentRef.setInput('loading', true);
    f.detectChanges();
    spy.mockClear();
    f.componentInstance.handleClick(new MouseEvent('click'));
    expect(spy).not.toHaveBeenCalled();
  });
});
