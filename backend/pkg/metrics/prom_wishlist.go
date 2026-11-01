package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// wishlistMetrics carries the wishlist-domain collectors.
type wishlistMetrics struct {
	wishlistCreated     *prometheus.CounterVec
	wishlistRetrieved   *prometheus.CounterVec
	wishlistItemAdded   *prometheus.CounterVec
	wishlistItemRemoved *prometheus.CounterVec
	wishlistCleared     *prometheus.CounterVec
}

// newWishlistMetrics builds and registers the wishlist collectors.
func newWishlistMetrics(reg prometheus.Registerer) wishlistMetrics {
	m := wishlistMetrics{
		// Dashboard: sum by (status) (rate(minimart_wishlist_created_total[5m]))
		wishlistCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "wishlist",
			Name:      "created_total",
			Help:      "Total number of wishlists created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_wishlist_retrieved_total[5m]))
		wishlistRetrieved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "wishlist",
			Name:      "retrieved_total",
			Help:      "Total number of wishlist retrievals by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_wishlist_item_added_total[5m]))
		wishlistItemAdded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "wishlist",
			Name:      "item_added_total",
			Help:      "Total number of wishlist item additions by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_wishlist_item_removed_total[5m]))
		wishlistItemRemoved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "wishlist",
			Name:      "item_removed_total",
			Help:      "Total number of wishlist item removals by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_wishlist_cleared_total[5m]))
		wishlistCleared: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "wishlist",
			Name:      "cleared_total",
			Help:      "Total number of wishlist clears by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(
		m.wishlistCreated,
		m.wishlistRetrieved,
		m.wishlistItemAdded,
		m.wishlistItemRemoved,
		m.wishlistCleared,
	)
	return m
}

func (m *wishlistMetrics) IncWishlistCreated(status string) {
	m.wishlistCreated.WithLabelValues(status).Inc()
}

func (m *wishlistMetrics) IncWishlistRetrieved(status string) {
	m.wishlistRetrieved.WithLabelValues(status).Inc()
}

func (m *wishlistMetrics) IncWishlistItemAdded(status string) {
	m.wishlistItemAdded.WithLabelValues(status).Inc()
}

func (m *wishlistMetrics) IncWishlistItemRemoved(status string) {
	m.wishlistItemRemoved.WithLabelValues(status).Inc()
}

func (m *wishlistMetrics) IncWishlistCleared(status string) {
	m.wishlistCleared.WithLabelValues(status).Inc()
}
