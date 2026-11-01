package postgres

import (
	"context"
	"errors"
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/postgres/reader"
	"minimart/internal/address/infrastructure/postgres/writer"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AddressRepository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
}

var _ domain.Repository = (*AddressRepository)(nil)

func NewAddressRepository(readPool, writePool *pgxpool.Pool) *AddressRepository {
	return &AddressRepository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
	}
}

func (r *AddressRepository) GetByID(ctx context.Context, id string) (domain.Address, error) {
	row, err := r.reader.GetAddressByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Address{}, domain.ErrAddressNotFound
		}
		return domain.Address{}, err
	}
	return toDomain(row), nil
}

func (r *AddressRepository) GetByCustomerID(ctx context.Context, customerID string) ([]domain.Address, error) {
	rows, err := r.reader.GetAddressesByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	addrs := make([]domain.Address, len(rows))
	for i, row := range rows {
		addrs[i] = toDomain(row)
	}
	return addrs, nil
}

func (r *AddressRepository) GetDefaultByCustomerID(ctx context.Context, customerID string) (domain.Address, error) {
	row, err := r.reader.GetDefaultAddressByCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Address{}, domain.ErrAddressNotFound
		}
		return domain.Address{}, err
	}
	return toDomain(row), nil
}

func (r *AddressRepository) Save(ctx context.Context, a domain.Address) error {
	rowsAffected, err := r.writer.CreateAddress(ctx, writer.CreateAddressParams{
		ID:         a.ID,
		CustomerID: a.CustomerID,
		IsDefault:  a.IsDefault,
		Label:      a.Label,
		FullName:   a.FullName,
		Street:     a.Street,
		City:       a.City,
		State:      a.State,
		Zip:        a.Zip,
		Country:    a.Country,
		Phone:      pgtype.Text{String: a.Phone, Valid: a.Phone != ""},
		CreatedAt:  pgtype.Timestamptz{Time: a.CreatedAt, Valid: true},
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrAddressExists
	}
	return nil
}

func (r *AddressRepository) Update(ctx context.Context, a domain.Address) (domain.Address, error) {
	row, err := r.writer.UpdateAddress(ctx, writer.UpdateAddressParams{
		ID:       a.ID,
		Label:    a.Label,
		FullName: a.FullName,
		Street:   a.Street,
		City:     a.City,
		State:    a.State,
		Zip:      a.Zip,
		Country:  a.Country,
		Phone:    pgtype.Text{String: a.Phone, Valid: a.Phone != ""},
		Version:  int32(a.Version), //nolint:gosec // version is positive by construction (starts at 1, increments on each write)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.classifyNoMatch(ctx, a.ID)
		}
		return domain.Address{}, err
	}
	return toDomainFromWriter(row), nil
}

func (r *AddressRepository) classifyNoMatch(ctx context.Context, id string) (domain.Address, error) {
	exists, err := r.writer.AddressExists(ctx, id)
	if err != nil {
		return domain.Address{}, err
	}
	if exists {
		return domain.Address{}, domain.ErrInvalidVersion
	}
	return domain.Address{}, domain.ErrAddressNotFound
}

func (r *AddressRepository) Delete(ctx context.Context, id, customerID string) error {
	rowsAffected, err := r.writer.DeleteAddress(ctx, writer.DeleteAddressParams{
		ID:         id,
		CustomerID: customerID,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *AddressRepository) SetDefault(ctx context.Context, id, customerID string) (domain.Address, error) {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return domain.Address{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.writer.WithTx(tx)

	_, err = qtx.UnsetOldDefault(ctx, customerID)
	if err != nil {
		return domain.Address{}, err
	}

	row, err := qtx.SetNewDefault(ctx, writer.SetNewDefaultParams{
		ID:         id,
		CustomerID: customerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Address{}, domain.ErrAddressNotFound
		}
		return domain.Address{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Address{}, err
	}

	return toDomainFromWriter(row), nil
}

func toDomain(row reader.Address) domain.Address {
	phone := ""
	if row.Phone.Valid {
		phone = row.Phone.String
	}
	return domain.Address{
		ID:         row.ID,
		CustomerID: row.CustomerID,
		IsDefault:  row.IsDefault,
		Label:      row.Label,
		FullName:   row.FullName,
		Street:     row.Street,
		City:       row.City,
		State:      row.State,
		Zip:        row.Zip,
		Country:    row.Country,
		Phone:      phone,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
		Version:    int(row.Version),
	}
}

func toDomainFromWriter(row writer.Address) domain.Address {
	phone := ""
	if row.Phone.Valid {
		phone = row.Phone.String
	}
	return domain.Address{
		ID:         row.ID,
		CustomerID: row.CustomerID,
		IsDefault:  row.IsDefault,
		Label:      row.Label,
		FullName:   row.FullName,
		Street:     row.Street,
		City:       row.City,
		State:      row.State,
		Zip:        row.Zip,
		Country:    row.Country,
		Phone:      phone,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
		Version:    int(row.Version),
	}
}
