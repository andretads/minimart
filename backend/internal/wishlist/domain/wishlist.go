package domain

import "time"

type WishlistItem struct {
	ProductID string
	AddedAt   time.Time
}

type Wishlist struct {
	ID         string
	CustomerID string
	Items      []WishlistItem
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewWishlist(id, customerID string) (Wishlist, error) {
	if customerID == "" {
		return Wishlist{}, ErrEmptyCustomerID
	}
	now := time.Now().UTC()
	return Wishlist{
		ID:         id,
		CustomerID: customerID,
		Items:      make([]WishlistItem, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (w *Wishlist) AddItem(productID string) error {
	if w.HasProduct(productID) {
		return ErrProductAlreadyInWishlist
	}
	w.Items = append(w.Items, WishlistItem{
		ProductID: productID,
		AddedAt:   time.Now().UTC(),
	})
	w.UpdatedAt = time.Now().UTC()
	return nil
}

func (w *Wishlist) RemoveItem(productID string) error {
	for i, item := range w.Items {
		if item.ProductID == productID {
			w.Items = append(w.Items[:i], w.Items[i+1:]...)
			w.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrProductNotInWishlist
}

func (w *Wishlist) Clear() error {
	w.Items = make([]WishlistItem, 0)
	w.UpdatedAt = time.Now().UTC()
	return nil
}

func (w *Wishlist) HasProduct(productID string) bool {
	for _, item := range w.Items {
		if item.ProductID == productID {
			return true
		}
	}
	return false
}
