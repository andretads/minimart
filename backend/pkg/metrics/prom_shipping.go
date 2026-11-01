package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// shippingMetrics carries the shipping-domain collectors: methods, zones,
// rates, quotes, shipments, and tracking events.
type shippingMetrics struct {
	shippingMethodsList  *prometheus.CounterVec
	shippingMethodGet    *prometheus.CounterVec
	shippingMethodCreate *prometheus.CounterVec
	shippingMethodUpdate *prometheus.CounterVec

	shippingZonesList  *prometheus.CounterVec
	shippingZoneGet    *prometheus.CounterVec
	shippingZoneCreate *prometheus.CounterVec
	shippingZoneUpdate *prometheus.CounterVec

	shippingRatesList  *prometheus.CounterVec
	shippingRateGet    *prometheus.CounterVec
	shippingRateCreate *prometheus.CounterVec
	shippingRateUpdate *prometheus.CounterVec

	shippingQuoteCreate   *prometheus.CounterVec
	shippingQuoteGet      *prometheus.CounterVec
	shippingQuoteValidate *prometheus.CounterVec
	shippingQuotePrice    *prometheus.CounterVec
	shippingQuotesExpire  *prometheus.CounterVec

	shipmentCreate              *prometheus.CounterVec
	shipmentCreateFromOrderPaid *prometheus.CounterVec
	shipmentGet                 *prometheus.CounterVec
	shipmentList                *prometheus.CounterVec

	shipmentProcess  *prometheus.CounterVec
	shipmentDispatch *prometheus.CounterVec
	shipmentDeliver  *prometheus.CounterVec

	trackingEventCreate *prometheus.CounterVec
	trackingEventList   *prometheus.CounterVec
}

// newShippingMetrics builds and registers the shipping collectors.
func newShippingMetrics(reg prometheus.Registerer) shippingMetrics {
	m := shippingMetrics{
		// Dashboard: sum by (status) (rate(minimart_shipping_methods_list_total[5m]))
		shippingMethodsList: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "methods_list_total",
			Help:      "Total number of shipping methods list requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_method_get_total[5m]))
		shippingMethodGet: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "method_get_total",
			Help:      "Total number of shipping method get requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_method_create_total[5m]))
		shippingMethodCreate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "method_create_total",
			Help:      "Total number of shipping method create requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_method_update_total[5m]))
		shippingMethodUpdate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "method_update_total",
			Help:      "Total number of shipping method update requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_zones_list_total[5m]))
		shippingZonesList: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "zones_list_total",
			Help:      "Total number of shipping zones list requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_zone_get_total[5m]))
		shippingZoneGet: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "zone_get_total",
			Help:      "Total number of shipping zone get requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_zone_create_total[5m]))
		shippingZoneCreate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "zone_create_total",
			Help:      "Total number of shipping zone create requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_zone_update_total[5m]))
		shippingZoneUpdate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "zone_update_total",
			Help:      "Total number of shipping zone update requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_rates_list_total[5m]))
		shippingRatesList: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "rates_list_total",
			Help:      "Total number of shipping rates list requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_rate_get_total[5m]))
		shippingRateGet: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "rate_get_total",
			Help:      "Total number of shipping rate get requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_rate_create_total[5m]))
		shippingRateCreate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "rate_create_total",
			Help:      "Total number of shipping rate create requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_rate_update_total[5m]))
		shippingRateUpdate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "rate_update_total",
			Help:      "Total number of shipping rate update requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_quote_create_total[5m]))
		shippingQuoteCreate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "quote_create_total",
			Help:      "Total number of shipping quote create requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_quote_get_total[5m]))
		shippingQuoteGet: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "quote_get_total",
			Help:      "Total number of shipping quote get requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_quote_validate_total[5m]))
		shippingQuoteValidate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "quote_validate_total",
			Help:      "Total number of shipping quote validate requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_quote_price_total[5m]))
		shippingQuotePrice: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "quote_price_total",
			Help:      "Total number of shipping quote pricing requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipping_quotes_expire_total[5m]))
		shippingQuotesExpire: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "quotes_expire_total",
			Help:      "Total number of shipping quote expiration operations by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_create_total[5m]))
		shipmentCreate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_create_total",
			Help:      "Total number of shipment create requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_create_from_order_paid_total[5m]))
		shipmentCreateFromOrderPaid: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_create_from_order_paid_total",
			Help:      "Total number of shipment create from order paid requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_get_total[5m]))
		shipmentGet: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_get_total",
			Help:      "Total number of shipment get requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_list_total[5m]))
		shipmentList: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_list_total",
			Help:      "Total number of shipment list requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_process_total[5m]))
		shipmentProcess: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_process_total",
			Help:      "Total number of shipment process requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_dispatch_total[5m]))
		shipmentDispatch: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_dispatch_total",
			Help:      "Total number of shipment dispatch requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_shipment_deliver_total[5m]))
		shipmentDeliver: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "shipment_deliver_total",
			Help:      "Total number of shipment deliver requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_tracking_event_create_total[5m]))
		trackingEventCreate: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "tracking_event_create_total",
			Help:      "Total number of tracking event create requests by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_tracking_event_list_total[5m]))
		trackingEventList: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "shipping",
			Name:      "tracking_event_list_total",
			Help:      "Total number of tracking event list requests by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(
		m.shippingMethodsList,
		m.shippingMethodGet,
		m.shippingMethodCreate,
		m.shippingMethodUpdate,

		m.shippingZonesList,
		m.shippingZoneGet,
		m.shippingZoneCreate,
		m.shippingZoneUpdate,

		m.shippingRatesList,
		m.shippingRateGet,
		m.shippingRateCreate,
		m.shippingRateUpdate,

		m.shippingQuoteCreate,
		m.shippingQuoteGet,
		m.shippingQuoteValidate,
		m.shippingQuotePrice,
		m.shippingQuotesExpire,

		m.shipmentCreate,
		m.shipmentCreateFromOrderPaid,
		m.shipmentGet,
		m.shipmentList,

		m.shipmentProcess,
		m.shipmentDispatch,
		m.shipmentDeliver,

		m.trackingEventCreate,
		m.trackingEventList,
	)
	return m
}

func (m *shippingMetrics) IncShippingMethodsList(status string) {
	m.shippingMethodsList.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingMethodGet(status string) {
	m.shippingMethodGet.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingMethodCreate(status string) {
	m.shippingMethodCreate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingMethodUpdate(status string) {
	m.shippingMethodUpdate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingZonesList(status string) {
	m.shippingZonesList.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingZoneGet(status string) {
	m.shippingZoneGet.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingZoneCreate(status string) {
	m.shippingZoneCreate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingZoneUpdate(status string) {
	m.shippingZoneUpdate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingRatesList(status string) {
	m.shippingRatesList.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingRateGet(status string) {
	m.shippingRateGet.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingRateCreate(status string) {
	m.shippingRateCreate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingRateUpdate(status string) {
	m.shippingRateUpdate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingQuoteCreate(status string) {
	m.shippingQuoteCreate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingQuoteGet(status string) {
	m.shippingQuoteGet.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingQuoteValidate(status string) {
	m.shippingQuoteValidate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingQuotePrice(status string) {
	m.shippingQuotePrice.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShippingQuotesExpire(status string) {
	m.shippingQuotesExpire.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentCreate(status string) {
	m.shipmentCreate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentCreateFromOrderPaid(status string) {
	m.shipmentCreateFromOrderPaid.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentGet(status string) {
	m.shipmentGet.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentList(status string) {
	m.shipmentList.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentProcess(status string) {
	m.shipmentProcess.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentDispatch(status string) {
	m.shipmentDispatch.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncShipmentDeliver(status string) {
	m.shipmentDeliver.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncTrackingEventCreate(status string) {
	m.trackingEventCreate.WithLabelValues(status).Inc()
}

func (m *shippingMetrics) IncTrackingEventList(status string) {
	m.trackingEventList.WithLabelValues(status).Inc()
}
