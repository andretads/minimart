package module

import (
	"context"
	"errors"
	"minimart/internal/address/domain"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
)

func (a *Address) GetDefaultAddressByCustomerID(ctx context.Context, customerID string) (addressclient.Address, error) {
	addr, err := a.svc.GetDefaultAddress(ctx, customerID)
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
