import { TestBed } from '@angular/core/testing';
import { provideAnimations } from '@angular/platform-browser/animations';
import { ReviewFormComponent } from './review-form.component';

describe('ReviewFormComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ReviewFormComponent],
      providers: [provideAnimations()],
    }).compileComponents();
  });

  function create() {
    const f = TestBed.createComponent(ReviewFormComponent);
    f.componentRef.setInput('productId', 'p1');
    f.detectChanges();
    return f;
  }

  it('descriptionError empty when untouched', () => {
    const f = create();
    expect(f.componentInstance.descriptionError()).toBe('');
  });

  it('setRating updates rating, form, focusedStar', () => {
    const f = create();
    f.componentInstance.setRating(3);
    f.detectChanges();
    expect(f.componentInstance.rating()).toBe(3);
    expect(f.componentInstance.form.controls.rating.value).toBe(3);
    expect(f.componentInstance.focusedStar()).toBe(3);
  });

  it('onKeyDown ArrowRight increments and clamps at 5', () => {
    const f = create();
    f.componentInstance.setRating(4);
    const ev = new KeyboardEvent('keydown', { key: 'ArrowRight' });
    const prevent = vi.spyOn(ev, 'preventDefault');
    f.componentInstance.onKeyDown(ev);
    expect(prevent).toHaveBeenCalled();
    expect(f.componentInstance.rating()).toBe(5);
    f.componentInstance.onKeyDown(new KeyboardEvent('keydown', { key: 'ArrowRight' }));
    expect(f.componentInstance.rating()).toBe(5);
  });

  it('onKeyDown ArrowLeft decrements and clamps at 1', () => {
    const f = create();
    f.componentInstance.setRating(2);
    f.componentInstance.onKeyDown(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    expect(f.componentInstance.rating()).toBe(1);
    f.componentInstance.onKeyDown(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    expect(f.componentInstance.rating()).toBe(1);
  });

  it('onKeyDown Home/End set 1/5', () => {
    const f = create();
    f.componentInstance.onKeyDown(new KeyboardEvent('keydown', { key: 'Home' }));
    expect(f.componentInstance.rating()).toBe(1);
    f.componentInstance.onKeyDown(new KeyboardEvent('keydown', { key: 'End' }));
    expect(f.componentInstance.rating()).toBe(5);
  });

  it('onKeyDown default returns without preventDefault', () => {
    const f = create();
    const ev = new KeyboardEvent('keydown', { key: 'Enter' });
    const prevent = vi.spyOn(ev, 'preventDefault');
    f.componentInstance.onKeyDown(ev);
    expect(prevent).not.toHaveBeenCalled();
  });

  it('onSubmit emits when valid', () => {
    const f = create();
    f.componentInstance.setRating(4);
    f.componentInstance.form.controls.description.setValue('a'.repeat(10));
    f.detectChanges();
    const spy = vi.fn();
    f.componentInstance.submitted.subscribe(spy);
    f.componentInstance.onSubmit();
    expect(spy).toHaveBeenCalledWith({ description: 'a'.repeat(10), rating: 4 });
  });

  it('onSubmit does not emit when invalid and marks touched', () => {
    const f = create();
    const spy = vi.fn();
    f.componentInstance.submitted.subscribe(spy);
    f.componentInstance.onSubmit();
    expect(spy).not.toHaveBeenCalled();
    expect(f.componentInstance.form.touched).toBe(true);
  });

  it('disabled effect disables form', () => {
    const f = TestBed.createComponent(ReviewFormComponent);
    f.componentRef.setInput('productId', 'p1');
    f.componentRef.setInput('disabled', true);
    f.detectChanges();
    expect(f.componentInstance.form.disabled).toBe(true);
  });
});
