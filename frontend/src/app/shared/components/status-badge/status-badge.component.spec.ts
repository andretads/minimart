import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StatusBadgeComponent } from './status-badge.component';

describe('StatusBadgeComponent', () => {
  let fixture: ComponentFixture<StatusBadgeComponent>;

  function createComponent(
    inputs: Record<string, unknown> = {},
  ): ComponentFixture<StatusBadgeComponent> {
    const f = TestBed.createComponent(StatusBadgeComponent);
    Object.entries(inputs).forEach(([key, value]) => {
      f.componentRef.setInput(key, value);
    });
    f.detectChanges();
    return f;
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StatusBadgeComponent],
    }).compileComponents();
  });

  it('shows default label and variant when no label/variant provided', () => {
    fixture = createComponent({ status: 'placed' });
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge.textContent.trim()).toBe('placed');
    expect(badge.classList.contains('placed')).toBe(true);
  });

  it('uses custom label when provided', () => {
    fixture = createComponent({ status: 'paid', label: 'Pagamento confirmado' });
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge.textContent.trim()).toBe('Pagamento confirmado');
  });

  it('uses custom variant when provided', () => {
    fixture = createComponent({ status: 'placed', variant: 'custom-variant' });
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge.classList.contains('custom-variant')).toBe(true);
  });

  it('reactively updates label and variant when inputs change', () => {
    fixture = createComponent({ status: 'placed', label: 'Aguardando pagamento' });
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge.textContent.trim()).toBe('Aguardando pagamento');
    expect(badge.classList.contains('placed')).toBe(true);

    // Simulate order cancellation - status and label change
    fixture.componentRef.setInput('status', 'cancelled');
    fixture.componentRef.setInput('label', 'Cancelado');
    fixture.detectChanges();

    expect(badge.textContent.trim()).toBe('Cancelado');
    expect(badge.classList.contains('cancelled')).toBe(true);
    expect(badge.classList.contains('placed')).toBe(false);
  });

  it('reactively updates when only status changes (label auto-derived)', () => {
    fixture = createComponent({ status: 'payment_failed' });
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge.textContent.trim()).toBe('payment failed');
    expect(badge.classList.contains('payment_failed')).toBe(true);

    fixture.componentRef.setInput('status', 'paid');
    fixture.detectChanges();

    expect(badge.textContent.trim()).toBe('paid');
    expect(badge.classList.contains('paid')).toBe(true);
    expect(badge.classList.contains('payment_failed')).toBe(false);
  });

  it('applies size classes', () => {
    fixture = createComponent({ status: 'placed', size: 'sm' });
    const badge = fixture.nativeElement.querySelector('.badge');
    expect(badge.classList.contains('badge-sm')).toBe(true);

    fixture.componentRef.setInput('size', 'lg');
    fixture.detectChanges();
    expect(badge.classList.contains('badge-lg')).toBe(true);
  });
});
