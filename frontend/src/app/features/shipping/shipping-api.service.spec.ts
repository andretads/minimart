import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '@core/api/api-client';
import { AuthStore } from '@core/auth/auth.store';
import { ShippingApiService } from './shipping-api.service';

describe('ShippingApiService', () => {
  let api: {
    get: ReturnType<typeof vi.fn>;
    post: ReturnType<typeof vi.fn>;
    patch: ReturnType<typeof vi.fn>;
  };
  let authStore: { userId: ReturnType<typeof vi.fn> };
  let service: ShippingApiService;

  beforeEach(() => {
    api = { get: vi.fn(() => of({})), post: vi.fn(() => of({})), patch: vi.fn(() => of({})) };
    authStore = { userId: vi.fn(() => 'u1') };
    TestBed.configureTestingModule({
      providers: [
        ShippingApiService,
        { provide: ApiClient, useValue: api },
        { provide: AuthStore, useValue: authStore },
      ],
    });
    service = TestBed.inject(ShippingApiService);
  });

  it('getMethods GET /v1/shipping/methods', () => {
    service.getMethods().subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/shipping/methods');
  });

  it('createQuote POST with snake_case body + idempotency header', () => {
    service.createQuote('m1', 'a1', 1000, 'idem-1').subscribe();
    expect(api.post).toHaveBeenCalledWith(
      '/v1/shipping/quotes',
      { method_id: 'm1', address_id: 'a1', subtotal_cents: 1000 },
      { headers: { 'Idempotency-Key': 'idem-1' } },
    );
  });

  it('createQuote throws when unauthenticated', () => {
    authStore.userId.mockReturnValue(null);
    expect(() => service.createQuote('m1', 'a1', 1000, 'idem')).toThrow('User not authenticated');
  });

  it('getQuote GET /v1/shipping/quotes/:id', () => {
    service.getQuote('q1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/shipping/quotes/q1');
  });

  it('getShipments GET /v1/shipping/shipments?order_id=:id', () => {
    api.get.mockReturnValue(of({ data: [] }));
    service.getShipments('o1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/shipping/shipments?order_id=o1');
  });

  it('getShipment GET /v1/shipping/shipments/:id', () => {
    service.getShipment('s1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/shipping/shipments/s1');
  });

  it('getTrackingEvents GET events', () => {
    service.getTrackingEvents('s1').subscribe();
    expect(api.get).toHaveBeenCalledWith('/v1/shipping/shipments/s1/events');
  });

  it('createMethod POST /v1/shipping/methods', () => {
    service
      .createMethod({
        name: 'n',
        description: 'd',
        active: true,
        estimatedMinDays: 1,
        estimatedMaxDays: 3,
      })
      .subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/methods', expect.any(Object));
  });

  it('updateMethod PATCH /v1/shipping/methods/:id', () => {
    service
      .updateMethod('m1', {
        name: 'n',
        description: 'd',
        active: true,
        estimatedMinDays: 1,
        estimatedMaxDays: 3,
      })
      .subscribe();
    expect(api.patch).toHaveBeenCalledWith('/v1/shipping/methods/m1', expect.any(Object));
  });

  it('createZone POST /v1/shipping/zones', () => {
    service
      .createZone({
        name: 'n',
        country: 'BR',
        state: 'SP',
        city: 'C',
        postalCodeStart: '0',
        postalCodeEnd: '9',
      })
      .subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/zones', expect.any(Object));
  });

  it('updateZone PATCH /v1/shipping/zones/:id', () => {
    service
      .updateZone('z1', {
        name: 'n',
        country: 'BR',
        state: 'SP',
        city: 'C',
        postalCodeStart: '0',
        postalCodeEnd: '9',
      })
      .subscribe();
    expect(api.patch).toHaveBeenCalledWith('/v1/shipping/zones/z1', expect.any(Object));
  });

  it('createRate POST /v1/shipping/rates', () => {
    service
      .createRate({
        zoneId: 'z',
        methodId: 'm',
        baseCostCents: 10,
        freeThresholdCents: 0,
        active: true,
      })
      .subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/rates', expect.any(Object));
  });

  it('updateRate PATCH /v1/shipping/rates/:id', () => {
    service
      .updateRate('r1', { baseCostCents: 10, freeThresholdCents: 0, active: true })
      .subscribe();
    expect(api.patch).toHaveBeenCalledWith('/v1/shipping/rates/r1', expect.any(Object));
  });

  it('processShipment POST /process', () => {
    service.processShipment('s1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/shipments/s1/process', {});
  });

  it('dispatchShipment POST /dispatch with tracking_code + carrier', () => {
    service.dispatchShipment('s1', 'TRK', 'Correios').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/shipments/s1/dispatch', {
      tracking_code: 'TRK',
      carrier: 'Correios',
    });
  });

  it('deliverShipment POST /deliver', () => {
    service.deliverShipment('s1').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/shipments/s1/deliver', {});
  });

  it('addTrackingEvent POST events with occurred_at', () => {
    service.addTrackingEvent('s1', 'shipped', 'desc', 'loc', '2026-01-01T00:00:00Z').subscribe();
    expect(api.post).toHaveBeenCalledWith('/v1/shipping/shipments/s1/events', {
      shipment_id: 's1',
      status: 'shipped',
      description: 'desc',
      location: 'loc',
      occurred_at: '2026-01-01T00:00:00Z',
    });
  });
});
