import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '@core/api/api-client';
import { AuthStore } from '@core/auth/auth.store';
import { ShippingMethod, ShippingQuote, Shipment } from '@shared/models/shipping.models';

@Injectable({ providedIn: 'root' })
export class ShippingApiService {
  private readonly api = inject(ApiClient);
  private readonly authStore = inject(AuthStore);

  getMethods(): Observable<ShippingMethod[]> {
    return this.api.get<ShippingMethod[]>('/v1/shipping/methods');
  }

  createQuote(
    methodId: string,
    addressId: string,
    subtotalCents: number,
    idempotencyKey: string,
  ): Observable<ShippingQuote> {
    const userId = this.authStore.userId();
    if (!userId) throw new Error('User not authenticated');
    return this.api.post<ShippingQuote>(
      `/v1/shipping/quotes`,
      { method_id: methodId, address_id: addressId, subtotal_cents: subtotalCents },
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
  }

  getQuote(id: string): Observable<ShippingQuote> {
    return this.api.get<ShippingQuote>(`/v1/shipping/quotes/${id}`);
  }

  getShipments(orderId: string): Observable<Shipment[]> {
    return this.api
      .get<{ data: Shipment[] }>(`/v1/shipping/shipments?order_id=${orderId}`)
      .pipe(map((response) => response.data));
  }

  getShipment(id: string): Observable<Shipment> {
    return this.api.get<Shipment>(`/v1/shipping/shipments/${id}`);
  }

  getTrackingEvents(shipmentId: string): Observable<TrackingEvent[]> {
    return this.api.get<TrackingEvent[]>(`/v1/shipping/shipments/${shipmentId}/events`);
  }

  // Admin methods
  createMethod(data: {
    name: string;
    description: string;
    active: boolean;
    estimatedMinDays: number;
    estimatedMaxDays: number;
  }): Observable<ShippingMethod> {
    return this.api.post<ShippingMethod>('/v1/shipping/methods', data);
  }

  updateMethod(
    id: string,
    data: {
      name: string;
      description: string;
      active: boolean;
      estimatedMinDays: number;
      estimatedMaxDays: number;
    },
  ): Observable<ShippingMethod> {
    return this.api.patch<ShippingMethod>(`/v1/shipping/methods/${id}`, data);
  }

  createZone(data: {
    name: string;
    country: string;
    state: string;
    city: string;
    postalCodeStart: string;
    postalCodeEnd: string;
  }): Observable<ShippingZone> {
    return this.api.post<ShippingZone>('/v1/shipping/zones', data);
  }

  updateZone(
    id: string,
    data: {
      name: string;
      country: string;
      state: string;
      city: string;
      postalCodeStart: string;
      postalCodeEnd: string;
    },
  ): Observable<ShippingZone> {
    return this.api.patch<ShippingZone>(`/v1/shipping/zones/${id}`, data);
  }

  createRate(data: {
    zoneId: string;
    methodId: string;
    baseCostCents: number;
    freeThresholdCents: number;
    active: boolean;
  }): Observable<ShippingRate> {
    return this.api.post<ShippingRate>('/v1/shipping/rates', data);
  }

  updateRate(
    id: string,
    data: { baseCostCents: number; freeThresholdCents: number; active: boolean },
  ): Observable<ShippingRate> {
    return this.api.patch<ShippingRate>(`/v1/shipping/rates/${id}`, data);
  }

  processShipment(id: string): Observable<Shipment> {
    return this.api.post<Shipment>(`/v1/shipping/shipments/${id}/process`, {});
  }

  dispatchShipment(id: string, trackingCode: string, carrier: string): Observable<Shipment> {
    return this.api.post<Shipment>(`/v1/shipping/shipments/${id}/dispatch`, {
      tracking_code: trackingCode,
      carrier,
    });
  }

  deliverShipment(id: string): Observable<Shipment> {
    return this.api.post<Shipment>(`/v1/shipping/shipments/${id}/deliver`, {});
  }

  addTrackingEvent(
    shipmentId: string,
    status: string,
    description: string,
    location: string,
    occurredAt: string,
  ): Observable<TrackingEvent> {
    return this.api.post<TrackingEvent>(`/v1/shipping/shipments/${shipmentId}/events`, {
      shipment_id: shipmentId,
      status,
      description,
      location,
      occurred_at: occurredAt,
    });
  }
}

import { map } from 'rxjs/operators';
import { ShippingZone, ShippingRate, TrackingEvent } from '@shared/models/shipping.models';
