import { TestBed } from '@angular/core/testing';
import { QuantityStepperComponent } from './quantity-stepper.component';

describe('QuantityStepperComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [QuantityStepperComponent],
    }).compileComponents();
  });

  function create(value = 1, min = 1, max = 99) {
    const f = TestBed.createComponent(QuantityStepperComponent);
    f.componentRef.setInput('value', value);
    f.componentRef.setInput('min', min);
    f.componentRef.setInput('max', max);
    f.detectChanges();
    return f;
  }

  it('ngOnInit sets lastValidValue', () => {
    const f = create(5);
    expect(f.componentInstance['lastValidValue']()).toBe(5);
  });

  it('increment emits when value < max and not disabled', () => {
    const f = create(1, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    f.componentInstance.increment();
    expect(spy).toHaveBeenCalledWith(2);
  });

  it('increment does nothing at max', () => {
    const f = create(5, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    f.componentInstance.increment();
    expect(spy).not.toHaveBeenCalled();
  });

  it('decrement emits when value > min', () => {
    const f = create(3, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    f.componentInstance.decrement();
    expect(spy).toHaveBeenCalledWith(2);
  });

  it('decrement does nothing at min', () => {
    const f = create(1, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    f.componentInstance.decrement();
    expect(spy).not.toHaveBeenCalled();
  });

  it('disabled blocks increment/decrement', () => {
    const f = create(3, 1, 5);
    f.componentRef.setInput('disabled', true);
    f.detectChanges();
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    f.componentInstance.increment();
    f.componentInstance.decrement();
    expect(spy).not.toHaveBeenCalled();
  });

  it('onInput accepts valid in-range integer', () => {
    const f = create(1, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    const input = document.createElement('input');
    input.value = '3';
    f.componentInstance.onInput({ target: input } as unknown as Event);
    expect(spy).toHaveBeenCalledWith(3);
  });

  it('onInput ignores invalid/out-of-range', () => {
    const f = create(1, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    const input = document.createElement('input');
    input.value = 'abc';
    f.componentInstance.onInput({ target: input } as unknown as Event);
    input.value = '10';
    f.componentInstance.onInput({ target: input } as unknown as Event);
    expect(spy).not.toHaveBeenCalled();
  });

  it('onBlur re-emits lastValidValue', () => {
    const f = create(1, 1, 5);
    const spy = vi.fn();
    f.componentInstance.valueChange.subscribe(spy);
    f.componentInstance.onBlur();
    expect(spy).toHaveBeenCalledWith(1);
  });
});
