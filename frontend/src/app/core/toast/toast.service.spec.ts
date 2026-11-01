import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { ToastService } from './toast.service';

describe('ToastService', () => {
  let service: ToastService;

  beforeEach(() => {
    vi.useFakeTimers();
    TestBed.configureTestingModule({});
    service = TestBed.inject(ToastService);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('show appends a toast and returns its id', () => {
    const id = service.show('hi', 'info', 0);
    expect(typeof id).toBe('string');
    expect(service.toasts()).toHaveLength(1);
    expect(service.toasts()[0]).toMatchObject({ message: 'hi', type: 'info', duration: 0 });
    expect(service.toasts()[0].id).toBe(id);
  });

  it('auto-dismisses after duration', () => {
    service.show('hi', 'info', 1000);
    expect(service.toasts()).toHaveLength(1);
    vi.advanceTimersByTime(1000);
    expect(service.toasts()).toHaveLength(0);
  });

  it('does not auto-dismiss when duration <= 0', () => {
    service.show('hi', 'info', 0);
    vi.advanceTimersByTime(10_000);
    expect(service.toasts()).toHaveLength(1);
  });

  it('success with explicit duration passes it through', () => {
    service.success('ok', 2000);
    expect(service.toasts()[0].type).toBe('success');
    expect(service.toasts()[0].duration).toBe(2000);
    vi.advanceTimersByTime(2000);
    expect(service.toasts()).toHaveLength(0);
  });

  it('convenience methods with omitted duration default to 5000ms auto-dismiss', () => {
    service.error('e');
    expect(service.toasts()).toHaveLength(1);
    vi.advanceTimersByTime(5000);
    expect(service.toasts()).toHaveLength(0);
  });

  it('dismiss removes by id and is a no-op for unknown id', () => {
    const id = service.show('a', 'info', 0);
    service.show('b', 'info', 0);
    service.dismiss('nope');
    expect(service.toasts()).toHaveLength(2);
    service.dismiss(id);
    expect(service.toasts()).toHaveLength(1);
    expect(service.toasts()[0].message).toBe('b');
  });

  it('clear empties all toasts', () => {
    service.show('a', 'info', 0);
    service.show('b', 'info', 0);
    service.clear();
    expect(service.toasts()).toHaveLength(0);
  });

  it('warning/info types are stored', () => {
    service.warning('w');
    service.info('i');
    expect(service.toasts().map((t) => t.type)).toEqual(['warning', 'info']);
  });
});
