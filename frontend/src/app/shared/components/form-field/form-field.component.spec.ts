import { TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { FormFieldComponent } from './form-field.component';

describe('FormFieldComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FormFieldComponent, ReactiveFormsModule],
    }).compileComponents();
  });

  it('writeValue sets value (null -> empty)', () => {
    const f = TestBed.createComponent(FormFieldComponent);
    f.componentRef.setInput('id', 'x');
    f.detectChanges();
    f.componentInstance.writeValue('hi');
    expect(f.componentInstance.value()).toBe('hi');
    f.componentInstance.writeValue(null as unknown as string);
    expect(f.componentInstance.value()).toBe('');
  });

  it('onInput updates value and calls onChange', () => {
    const f = TestBed.createComponent(FormFieldComponent);
    f.componentRef.setInput('id', 'x');
    f.detectChanges();
    const cb = vi.fn();
    f.componentInstance.registerOnChange(cb);
    const input = document.createElement('input');
    input.value = 'typed';
    f.componentInstance.onInput({ target: input } as unknown as Event);
    expect(f.componentInstance.value()).toBe('typed');
    expect(cb).toHaveBeenCalledWith('typed');
  });

  it('togglePassword flips showPassword', () => {
    const f = TestBed.createComponent(FormFieldComponent);
    f.componentRef.setInput('id', 'x');
    f.detectChanges();
    expect(f.componentInstance.showPassword()).toBe(false);
    f.componentInstance.togglePassword();
    expect(f.componentInstance.showPassword()).toBe(true);
  });

  it('isDisabled combines disabled input and formControlDisabled', () => {
    const f = TestBed.createComponent(FormFieldComponent);
    f.componentRef.setInput('id', 'x');
    f.componentRef.setInput('disabled', true);
    f.detectChanges();
    expect(f.componentInstance.isDisabled()).toBe(true);
    f.componentInstance.setDisabledState(false);
    f.detectChanges();
    // disabled input still true
    expect(f.componentInstance.isDisabled()).toBe(true);
  });

  it('computedError: explicit error input wins', () => {
    const f = TestBed.createComponent(FormFieldComponent);
    f.componentRef.setInput('id', 'x');
    f.componentRef.setInput('error', 'custom err');
    f.detectChanges();
    expect(f.componentInstance.computedError()).toBe('custom err');
  });
});
