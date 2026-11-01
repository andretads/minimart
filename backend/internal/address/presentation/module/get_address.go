package module

import (
	"context"
	"errors"
	"minimart/internal/address/domain"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
)

// GetAddress is the cross-domain entry point. It calls the service, then maps the
// domain result into the published client types, so callers depend only on the client package.
func (a *Address) GetAddress(ctx context.Context, req addressclient.GetAddressRequest) (addressclient.Address, error) {
	addr, err := a.svc.GetAddress(ctx, req.ID, req.CustomerID)
	if err != nil {
		if errors.Is(err, domain.ErrAddressNotFound) {
			return addressclient.Address{}, addressclient.ErrAddressNotFound
		}
		return addressclient.Address{}, err
	}
	return addressclient.Address{
		ID:         addr.ID,
		CustomerID: addr.CustomerID,
		Label:      addr.Label,
		FullName:   addr.FullName,
		Street:     addr.Street,
		City:       addr.City,
		State:      addr.State,
		Zip:        addr.Zip,
		Country:    addr.Country,
		Phone:      addr.Phone,
		IsDefault:  addr.IsDefault,
		Version:    addr.Version,
		CreatedAt:  addr.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  addr.UpdatedAt.Format(time.RFC3339),
	}, nil
}
