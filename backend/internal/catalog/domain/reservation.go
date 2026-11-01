package domain

// ReservationItem is one product and the quantity an order wants to reserve from
// it. A reservation is all-or-nothing across its items: if any item lacks stock,
// none of them are reserved.
type ReservationItem struct {
	ProductID string
	Quantity  int
}
