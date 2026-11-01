import { TestBed } from '@angular/core/testing';
import { EmptyStateComponent } from './empty-state.component';

describe('EmptyStateComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [EmptyStateComponent] }).compileComponents();
  });

  it('shows skeletons when loading', () => {
    const f = TestBed.createComponent(EmptyStateComponent);
    f.componentRef.setInput('title', 'Empty');
    f.componentRef.setInput('loading', true);
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('app-skeleton').length).toBe(3);
    expect(f.nativeElement.textContent).not.toContain('Empty');
  });

  it('shows icon/title/message/action when not loading', () => {
    const f = TestBed.createComponent(EmptyStateComponent);
    f.componentRef.setInput('title', 'Empty');
    f.componentRef.setInput('message', 'msg');
    f.componentRef.setInput('actionLabel', 'Do it');
    f.componentRef.setInput('icon', '');
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('Empty');
    expect(f.nativeElement.textContent).toContain('msg');
    expect(f.nativeElement.textContent).toContain('Do it');
    // empty icon falls back to 'package'
    expect(f.nativeElement.querySelector('.icon')?.className).toContain('icon-package');
  });

  it('emits action on button click', () => {
    const f = TestBed.createComponent(EmptyStateComponent);
    f.componentRef.setInput('title', 'Empty');
    f.componentRef.setInput('actionLabel', 'Do it');
    f.detectChanges();
    const spy = vi.fn();
    f.componentInstance.action.subscribe(spy);
    (f.nativeElement.querySelector('button') as HTMLElement).click();
    expect(spy).toHaveBeenCalled();
  });
});
