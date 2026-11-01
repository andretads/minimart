package module

import "context"

// ReleaseStock adapts the application use case to the published contract. It
// passes errors through unchanged: unlike reserve, there is no shortage case to
// translate.
func (c *Catalog) ReleaseStock(ctx context.Context, orderID string) error {
	return c.svc.ReleaseStock(ctx, orderID)
}
