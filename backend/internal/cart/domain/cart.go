package domain

import "time"

type Status string

const (
	StatusActive          Status = "active"
	StatusCheckoutPending Status = "checkout_pending"
	StatusConsumed        Status = "consumed"
)

type Item struct {
	ProductID              string
	ProductNameSnapshot    string
	UnitPriceSnapshotCents int64
	Quantity               int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Cart struct {
	ID          string
	CustomerID  string
	Status      Status
	CheckoutKey string
	OrderID     string
	Items       []Item
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
}

func NewCart(id, customerID string) (Cart, error) {
	if customerID == "" {
		return Cart{}, ErrEmptyCustomerID
	}
	now := time.Now().UTC()
	return Cart{
		ID:         id,
		CustomerID: customerID,
		Status:     StatusActive,
		Items:      make([]Item, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

func (c *Cart) EstimatedTotalCents() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.UnitPriceSnapshotCents * int64(item.Quantity)
	}
	return total
}

func (c *Cart) findItemIndex(productID string) int {
	for i, item := range c.Items {
		if item.ProductID == productID {
			return i
		}
	}
	return -1
}

func (c *Cart) AddItem(productID, productName string, unitPriceCents int64, quantity int) error {
	if quantity < 1 {
		return ErrInvalidQuantity
	}
	if c.Status != StatusActive {
		return ErrCartNotActive
	}
	idx := c.findItemIndex(productID)
	now := time.Now().UTC()
	if idx >= 0 {
		c.Items[idx].Quantity += quantity
		c.Items[idx].UpdatedAt = now
	} else {
		c.Items = append(c.Items, Item{
			ProductID:              productID,
			ProductNameSnapshot:    productName,
			UnitPriceSnapshotCents: unitPriceCents,
			Quantity:               quantity,
			CreatedAt:              now,
			UpdatedAt:              now,
		})
	}
	c.UpdatedAt = now
	return nil
}

func (c *Cart) UpdateItemQuantity(productID string, quantity int) error {
	if quantity < 1 {
		return ErrInvalidQuantity
	}
	if c.Status != StatusActive {
		return ErrCartNotActive
	}
	idx := c.findItemIndex(productID)
	if idx < 0 {
		return ErrCartItemNotFound
	}
	c.Items[idx].Quantity = quantity
	c.Items[idx].UpdatedAt = time.Now().UTC()
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Cart) RemoveItem(productID string) error {
	if c.Status != StatusActive {
		return ErrCartNotActive
	}
	idx := c.findItemIndex(productID)
	if idx < 0 {
		return ErrCartItemNotFound
	}
	c.Items = append(c.Items[:idx], c.Items[idx+1:]...)
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Cart) Clear() error {
	if c.Status != StatusActive {
		return ErrCartNotActive
	}
	c.Items = make([]Item, 0)
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Cart) BeginCheckout(key string) error {
	if c.Status != StatusActive {
		if c.Status == StatusConsumed {
			return ErrCartConsumed
		}
		return ErrCartCheckoutPending
	}
	c.Status = StatusCheckoutPending
	c.CheckoutKey = key
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Cart) MarkConsumed(orderID string) error {
	if c.Status != StatusCheckoutPending {
		return ErrCartStateChanged
	}
	c.Status = StatusConsumed
	c.OrderID = orderID
	c.CheckoutKey = ""
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Cart) ReleaseCheckout() error {
	if c.Status != StatusCheckoutPending {
		return ErrCartStateChanged
	}
	c.Status = StatusActive
	c.CheckoutKey = ""
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Cart) IsMutable() bool {
	return c.Status == StatusActive
}
