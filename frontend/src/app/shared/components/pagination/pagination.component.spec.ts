import { TestBed } from '@angular/core/testing';
import { PaginationComponent } from './pagination.component';

describe('PaginationComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [PaginationComponent] }).compileComponents();
  });

  function create(current: number, total: number) {
    const f = TestBed.createComponent(PaginationComponent);
    f.componentRef.setInput('currentPage', current);
    f.componentRef.setInput('totalPages', total);
    f.detectChanges();
    return f;
  }

  it('total <= 7 -> plain list', () => {
    const f = create(1, 5);
    expect(f.componentInstance.visiblePages()).toEqual([1, 2, 3, 4, 5]);
  });

  it('total > 7, current=1 -> no leading ellipsis', () => {
    const f = create(1, 10);
    expect(f.componentInstance.visiblePages()).toEqual([1, 2, '...', 10]);
  });

  it('total > 7, current=4 -> both ellipses', () => {
    const f = create(4, 10);
    expect(f.componentInstance.visiblePages()).toEqual([1, '...', 3, 4, 5, '...', 10]);
  });

  it('total > 7, current=last -> no trailing ellipsis', () => {
    const f = create(10, 10);
    expect(f.componentInstance.visiblePages()).toEqual([1, '...', 9, 10]);
  });

  it('total=0 -> []', () => {
    const f = create(1, 0);
    expect(f.componentInstance.visiblePages()).toEqual([]);
  });

  it('total=1 -> [1]', () => {
    const f = create(1, 1);
    expect(f.componentInstance.visiblePages()).toEqual([1]);
  });

  it('renders nav only when totalPages>1 and emits pageChange', () => {
    const f = create(1, 1);
    expect(f.nativeElement.querySelector('nav')).toBeNull();
    const f2 = create(1, 5);
    const spy = vi.fn();
    f2.componentInstance.pageChange.subscribe(spy);
    const btn = Array.from(f2.nativeElement.querySelectorAll('button')).find(
      (b: unknown) => (b as HTMLElement).textContent?.trim() === '2',
    ) as HTMLElement;
    btn?.click();
    expect(spy).toHaveBeenCalledWith(2);
  });
});
