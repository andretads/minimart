package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/postgres/reader"
	"minimart/internal/shipping/infrastructure/postgres/writer"
	"minimart/pkg/outbox"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ domain.Repository = (*Repository)(nil)

// EventPublisher writes a domain event into the transactional outbox within the
// caller's transaction, so the event commits atomically with the shipment.
// *outbox.Publisher is the production implementation.
type EventPublisher interface {
	Send(ctx context.Context, msg outbox.Message, tx pgx.Tx) error
}

type Repository struct {
	readPool  *pgxpool.Pool
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
	events    EventPublisher
}

func NewRepository(readPool, writePool *pgxpool.Pool, events EventPublisher) *Repository {
	return &Repository{
		readPool:  readPool,
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
		events:    events,
	}
}

// Shipping Methods
func (r *Repository) GetMethod(ctx context.Context, id string) (domain.ShippingMethod, error) {
	row, err := r.reader.GetMethod(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ShippingMethod{}, domain.ErrMethodNotFound
		}
		return domain.ShippingMethod{}, err
	}
	return domain.ShippingMethod{
		ID:               row.ID,
		Name:             row.Name,
		Description:      row.Description,
		Active:           row.Active,
		EstimatedMinDays: int(row.EstimatedMinDays),
		EstimatedMaxDays: int(row.EstimatedMaxDays),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) ListActiveMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	rows, err := r.reader.ListActiveMethods(ctx)
	if err != nil {
		return nil, err
	}
	return toMethods(rows), nil
}

func (r *Repository) ListAllMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	rows, err := r.reader.ListAllMethods(ctx)
	if err != nil {
		return nil, err
	}
	return toMethods(rows), nil
}

func toMethods(rows []reader.ShippingMethod) []domain.ShippingMethod {
	methods := make([]domain.ShippingMethod, len(rows))
	for i, row := range rows {
		methods[i] = domain.ShippingMethod{
			ID:               row.ID,
			Name:             row.Name,
			Description:      row.Description,
			Active:           row.Active,
			EstimatedMinDays: int(row.EstimatedMinDays),
			EstimatedMaxDays: int(row.EstimatedMaxDays),
			CreatedAt:        row.CreatedAt.Time,
			UpdatedAt:        row.UpdatedAt.Time,
		}
	}
	return methods
}

func (r *Repository) CreateMethod(ctx context.Context, method domain.ShippingMethod) error {
	return r.writer.CreateMethod(ctx, writer.CreateMethodParams{
		ID:               method.ID,
		Name:             method.Name,
		Description:      method.Description,
		Active:           method.Active,
		EstimatedMinDays: clampInt32(method.EstimatedMinDays),
		EstimatedMaxDays: clampInt32(method.EstimatedMaxDays),
		CreatedAt:        pgtype.Timestamptz{Time: method.CreatedAt, Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: method.UpdatedAt, Valid: true},
	})
}

func (r *Repository) UpdateMethod(ctx context.Context, method domain.ShippingMethod) error {
	return r.writer.UpdateMethod(ctx, writer.UpdateMethodParams{
		ID:               method.ID,
		Name:             method.Name,
		Description:      method.Description,
		Active:           method.Active,
		EstimatedMinDays: clampInt32(method.EstimatedMinDays),
		EstimatedMaxDays: clampInt32(method.EstimatedMaxDays),
		UpdatedAt:        pgtype.Timestamptz{Time: method.UpdatedAt, Valid: true},
	})
}

// Shipping Zones
func (r *Repository) GetZone(ctx context.Context, id string) (domain.ShippingZone, error) {
	row, err := r.reader.GetZone(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ShippingZone{}, domain.ErrZoneNotFound
		}
		return domain.ShippingZone{}, err
	}
	return domain.ShippingZone{
		ID:              row.ID,
		Name:            row.Name,
		Country:         row.Country,
		State:           row.State,
		City:            row.City,
		PostalCodeStart: row.PostalCodeStart,
		PostalCodeEnd:   row.PostalCodeEnd,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) ListZones(ctx context.Context) ([]domain.ShippingZone, error) {
	rows, err := r.reader.ListZones(ctx)
	if err != nil {
		return nil, err
	}
	zones := make([]domain.ShippingZone, len(rows))
	for i, row := range rows {
		zones[i] = domain.ShippingZone{
			ID:              row.ID,
			Name:            row.Name,
			Country:         row.Country,
			State:           row.State,
			City:            row.City,
			PostalCodeStart: row.PostalCodeStart,
			PostalCodeEnd:   row.PostalCodeEnd,
			CreatedAt:       row.CreatedAt.Time,
			UpdatedAt:       row.UpdatedAt.Time,
		}
	}
	return zones, nil
}

func (r *Repository) CreateZone(ctx context.Context, zone domain.ShippingZone) error {
	return r.writer.CreateZone(ctx, writer.CreateZoneParams{
		ID:              zone.ID,
		Name:            zone.Name,
		Country:         zone.Country,
		State:           zone.State,
		City:            zone.City,
		PostalCodeStart: zone.PostalCodeStart,
		PostalCodeEnd:   zone.PostalCodeEnd,
		CreatedAt:       pgtype.Timestamptz{Time: zone.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: zone.UpdatedAt, Valid: true},
	})
}

func (r *Repository) UpdateZone(ctx context.Context, zone domain.ShippingZone) error {
	return r.writer.UpdateZone(ctx, writer.UpdateZoneParams{
		ID:              zone.ID,
		Name:            zone.Name,
		Country:         zone.Country,
		State:           zone.State,
		City:            zone.City,
		PostalCodeStart: zone.PostalCodeStart,
		PostalCodeEnd:   zone.PostalCodeEnd,
		UpdatedAt:       pgtype.Timestamptz{Time: zone.UpdatedAt, Valid: true},
	})
}

// Shipping Rates
func (r *Repository) GetRate(ctx context.Context, id string) (domain.ShippingRate, error) {
	row, err := r.reader.GetRate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ShippingRate{}, domain.ErrRateNotFound
		}
		return domain.ShippingRate{}, err
	}
	return domain.ShippingRate{
		ID:                 row.ID,
		ZoneID:             row.ZoneID,
		MethodID:           row.MethodID,
		BaseCostCents:      row.BaseCostCents,
		FreeThresholdCents: row.FreeThresholdCents,
		Active:             row.Active,
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) ListRates(ctx context.Context, zoneID, methodID string) ([]domain.ShippingRate, error) {
	rows, err := r.reader.ListRates(ctx, reader.ListRatesParams{ZoneID: zoneID, MethodID: methodID})
	if err != nil {
		return nil, err
	}
	rates := make([]domain.ShippingRate, len(rows))
	for i, row := range rows {
		rates[i] = domain.ShippingRate{
			ID:                 row.ID,
			ZoneID:             row.ZoneID,
			MethodID:           row.MethodID,
			BaseCostCents:      row.BaseCostCents,
			FreeThresholdCents: row.FreeThresholdCents,
			Active:             row.Active,
			CreatedAt:          row.CreatedAt.Time,
			UpdatedAt:          row.UpdatedAt.Time,
		}
	}
	return rates, nil
}

func (r *Repository) ListRatesByZone(ctx context.Context, zoneID string) ([]domain.ShippingRate, error) {
	return r.ListRates(ctx, zoneID, "")
}

func (r *Repository) ListRatesByMethod(ctx context.Context, methodID string) ([]domain.ShippingRate, error) {
	return r.ListRates(ctx, "", methodID)
}

func (r *Repository) CreateRate(ctx context.Context, rate domain.ShippingRate) error {
	return r.writer.CreateRate(ctx, writer.CreateRateParams{
		ID:                 rate.ID,
		ZoneID:             rate.ZoneID,
		MethodID:           rate.MethodID,
		BaseCostCents:      rate.BaseCostCents,
		FreeThresholdCents: rate.FreeThresholdCents,
		Active:             rate.Active,
		CreatedAt:          pgtype.Timestamptz{Time: rate.CreatedAt, Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: rate.UpdatedAt, Valid: true},
	})
}

func (r *Repository) UpdateRate(ctx context.Context, rate domain.ShippingRate) error {
	return r.writer.UpdateRate(ctx, writer.UpdateRateParams{
		ID:                 rate.ID,
		BaseCostCents:      rate.BaseCostCents,
		FreeThresholdCents: rate.FreeThresholdCents,
		Active:             rate.Active,
		UpdatedAt:          pgtype.Timestamptz{Time: rate.UpdatedAt, Valid: true},
	})
}

// Shipping Quotes
func (r *Repository) GetQuote(ctx context.Context, id string) (domain.ShippingQuote, error) {
	row, err := r.reader.GetQuote(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ShippingQuote{}, domain.ErrQuoteNotFound
		}
		return domain.ShippingQuote{}, err
	}
	return toQuote(row)
}

// GetQuoteByIdempotencyKey returns the quote created under an idempotency key,
// or ErrQuoteNotFound when none carries it. This is the authoritative idempotency
// lookup: it holds even if the Redis cache is empty.
func (r *Repository) GetQuoteByIdempotencyKey(ctx context.Context, key string) (domain.ShippingQuote, error) {
	row, err := r.reader.GetQuoteByIdempotencyKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ShippingQuote{}, domain.ErrQuoteNotFound
		}
		return domain.ShippingQuote{}, err
	}
	return toQuote(row)
}

// toQuote maps a stored quote row onto the domain entity.
func toQuote(row reader.ShippingQuote) (domain.ShippingQuote, error) {
	dest, err := parseDestinationSnapshot(row.DestinationSnapshot)
	if err != nil {
		return domain.ShippingQuote{}, fmt.Errorf("parse destination: %w", err)
	}
	return domain.ShippingQuote{
		ID:                  row.ID,
		CustomerID:          row.CustomerID,
		MethodID:            row.MethodID,
		AddressID:           row.AddressID,
		IdempotencyKey:      row.IdempotencyKey,
		DestinationSnapshot: dest,
		ShippingCostCents:   row.ShippingCostCents,
		SubtotalCents:       row.SubtotalCents,
		FreeThresholdCents:  row.FreeThresholdCents,
		EstimatedMinDays:    int(row.EstimatedMinDays),
		EstimatedMaxDays:    int(row.EstimatedMaxDays),
		Status:              domain.QuoteStatus(row.Status),
		ExpiresAt:           row.ExpiresAt.Time,
		CreatedAt:           row.CreatedAt.Time,
	}, nil
}

func (r *Repository) CreateQuote(ctx context.Context, quote domain.ShippingQuote) error {
	destJSON, err := json.Marshal(quote.DestinationSnapshot)
	if err != nil {
		return fmt.Errorf("marshal destination: %w", err)
	}
	rows, err := r.writer.CreateQuote(ctx, writer.CreateQuoteParams{
		ID:                  quote.ID,
		CustomerID:          quote.CustomerID,
		MethodID:            quote.MethodID,
		AddressID:           quote.AddressID,
		IdempotencyKey:      quote.IdempotencyKey,
		DestinationSnapshot: destJSON,
		ShippingCostCents:   quote.ShippingCostCents,
		SubtotalCents:       quote.SubtotalCents,
		FreeThresholdCents:  quote.FreeThresholdCents,
		EstimatedMinDays:    clampInt32(quote.EstimatedMinDays),
		EstimatedMaxDays:    clampInt32(quote.EstimatedMaxDays),
		Status:              string(quote.Status),
		ExpiresAt:           pgtype.Timestamptz{Time: quote.ExpiresAt, Valid: true},
		CreatedAt:           pgtype.Timestamptz{Time: quote.CreatedAt, Valid: true},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		// Another quote already holds this idempotency key.
		return domain.ErrIdempotencyConflict
	}
	return nil
}

func (r *Repository) UpdateQuote(ctx context.Context, quote domain.ShippingQuote) error {
	return r.writer.UpdateQuote(ctx, writer.UpdateQuoteParams{
		ID:                 quote.ID,
		Status:             string(quote.Status),
		ExpiresAt:          pgtype.Timestamptz{Time: quote.ExpiresAt, Valid: true},
		ShippingCostCents:  quote.ShippingCostCents,
		SubtotalCents:      quote.SubtotalCents,
		FreeThresholdCents: quote.FreeThresholdCents,
	})
}

func (r *Repository) ExpireQuotes(ctx context.Context) error {
	return r.writer.ExpireQuotes(ctx)
}

// Shipments

// toShipment maps a persisted shipment row into the domain model, decoding
// the destination snapshot embedded as JSON.
func toShipment(row reader.Shipment) (domain.Shipment, error) {
	dest, err := parseDestinationSnapshot(row.DestinationSnapshot)
	if err != nil {
		return domain.Shipment{}, fmt.Errorf("parse destination: %w", err)
	}
	var shippedAt, deliveredAt *time.Time
	if row.ShippedAt.Valid {
		t := row.ShippedAt.Time
		shippedAt = &t
	}
	if row.DeliveredAt.Valid {
		t := row.DeliveredAt.Time
		deliveredAt = &t
	}
	return domain.Shipment{
		ID:                  row.ID,
		OrderID:             row.OrderID,
		UserID:              row.UserID,
		MethodID:            row.MethodID,
		TrackingCode:        row.TrackingCode,
		Carrier:             row.Carrier,
		ShippingCostCents:   row.ShippingCostCents,
		DestinationSnapshot: dest,
		Status:              domain.ShipmentStatus(row.Status),
		ShippedAt:           shippedAt,
		DeliveredAt:         deliveredAt,
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) GetShipment(ctx context.Context, id string) (domain.Shipment, error) {
	row, err := r.reader.GetShipment(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Shipment{}, domain.ErrShipmentNotFound
		}
		return domain.Shipment{}, err
	}
	return toShipment(row)
}

func (r *Repository) GetShipmentByOrderID(ctx context.Context, orderID string) (domain.Shipment, error) {
	row, err := r.reader.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Shipment{}, domain.ErrShipmentNotFound
		}
		return domain.Shipment{}, err
	}
	return toShipment(row)
}

func (r *Repository) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]domain.Shipment, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.reader.ListShipments(ctx, reader.ListShipmentsParams{
		UserID: userID,
		Limit:  clampInt32(pageSize),
		Offset: clampInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.reader.CountShipments(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	shipments := make([]domain.Shipment, len(rows))
	for i, row := range rows {
		shipment, err := toShipment(row)
		if err != nil {
			return nil, 0, fmt.Errorf("parse destination snapshot: %w", err)
		}
		shipments[i] = shipment
	}
	return shipments, int(total), nil
}

// CreateShipment persists a new shipment and its domain events in one
// transaction: the shipment row and the outbox row commit together, so there is
// never a shipment without its shipment-created event, nor an event without its
// shipment.
func (r *Repository) CreateShipment(ctx context.Context, shipment domain.Shipment) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	if err := r.insertShipment(ctx, tx, shipment); err != nil {
		return err
	}
	if err := r.appendEvents(ctx, tx, shipment.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CreateShipmentForQuote persists a shipment created from quote, consumes the
// quote, and records the shipment's domain events - all in one transaction. A
// quote can therefore never be spent twice, and a shipment never exists without
// its quote consumed.
func (r *Repository) CreateShipmentForQuote(ctx context.Context, shipment domain.Shipment, quote domain.ShippingQuote) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	if err := r.insertShipment(ctx, tx, shipment); err != nil {
		return err
	}
	if err := r.consumeQuote(ctx, tx, quote); err != nil {
		return err
	}
	if err := r.appendEvents(ctx, tx, shipment.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) insertShipment(ctx context.Context, tx pgx.Tx, shipment domain.Shipment) error {
	destJSON, err := json.Marshal(shipment.DestinationSnapshot)
	if err != nil {
		return fmt.Errorf("marshal destination: %w", err)
	}
	var shippedAt, deliveredAt pgtype.Timestamptz
	if shipment.ShippedAt != nil {
		shippedAt = pgtype.Timestamptz{Time: *shipment.ShippedAt, Valid: true}
	}
	if shipment.DeliveredAt != nil {
		deliveredAt = pgtype.Timestamptz{Time: *shipment.DeliveredAt, Valid: true}
	}
	err = r.writer.WithTx(tx).CreateShipment(ctx, writer.CreateShipmentParams{
		ID:                  shipment.ID,
		OrderID:             shipment.OrderID,
		UserID:              shipment.UserID,
		MethodID:            shipment.MethodID,
		TrackingCode:        shipment.TrackingCode,
		Carrier:             shipment.Carrier,
		ShippingCostCents:   shipment.ShippingCostCents,
		DestinationSnapshot: destJSON,
		Status:              string(shipment.Status),
		ShippedAt:           shippedAt,
		DeliveredAt:         deliveredAt,
		CreatedAt:           pgtype.Timestamptz{Time: shipment.CreatedAt, Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Time: shipment.UpdatedAt, Valid: true},
	})
	if err != nil {
		return mapDuplicateShipment(err)
	}
	return nil
}

// consumeQuote marks the quote as spent. It is a conditional update, so two
// concurrent shipments racing for the same quote cannot both win: the loser
// sees zero updated rows and the quote is reported as consumed.
func (r *Repository) consumeQuote(ctx context.Context, tx pgx.Tx, quote domain.ShippingQuote) error {
	rows, err := r.writer.WithTx(tx).ConsumeQuote(ctx, writer.ConsumeQuoteParams{
		ID:        quote.ID,
		NewStatus: string(domain.QuoteStatusConsumed),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrQuoteConsumed
	}
	return nil
}

// UpdateShipment persists a change to a shipment's state together with the
// shipment's domain events, in one transaction.
func (r *Repository) UpdateShipment(ctx context.Context, shipment domain.Shipment) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	destJSON, err := json.Marshal(shipment.DestinationSnapshot)
	if err != nil {
		return fmt.Errorf("marshal destination: %w", err)
	}
	var shippedAt, deliveredAt pgtype.Timestamptz
	if shipment.ShippedAt != nil {
		shippedAt = pgtype.Timestamptz{Time: *shipment.ShippedAt, Valid: true}
	}
	if shipment.DeliveredAt != nil {
		deliveredAt = pgtype.Timestamptz{Time: *shipment.DeliveredAt, Valid: true}
	}
	if err := r.writer.WithTx(tx).UpdateShipment(ctx, writer.UpdateShipmentParams{
		ID:                  shipment.ID,
		TrackingCode:        shipment.TrackingCode,
		Carrier:             shipment.Carrier,
		ShippingCostCents:   shipment.ShippingCostCents,
		DestinationSnapshot: destJSON,
		Status:              string(shipment.Status),
		ShippedAt:           shippedAt,
		DeliveredAt:         deliveredAt,
		UpdatedAt:           pgtype.Timestamptz{Time: shipment.UpdatedAt, Valid: true},
	}); err != nil {
		return err
	}
	if err := r.appendEvents(ctx, tx, shipment.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// appendEvents writes each domain event to the outbox inside tx. Writing
// through the shared outbox store (which participates in the same tx) is what
// makes the state change and its events atomic.
func (r *Repository) appendEvents(ctx context.Context, tx pgx.Tx, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}
	if r.events == nil {
		return errors.New("shipping repository: no event publisher configured")
	}
	for _, ev := range events {
		msg, err := outboxMessage(ev)
		if err != nil {
			return err
		}
		if err := r.events.Send(ctx, msg, tx); err != nil {
			return fmt.Errorf("enqueue shipping event: %w", err)
		}
	}
	return nil
}

// mapDuplicateShipment translates Postgres' unique-violation on
// shipments.order_id into the domain's sentinel, so callers can tell "this
// order already has a shipment" from a real storage failure.
func mapDuplicateShipment(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrShipmentExists
	}
	return err
}

// Tracking Events
func (r *Repository) AddTrackingEvent(ctx context.Context, event domain.TrackingEvent) error {
	return r.writer.AddTrackingEvent(ctx, writer.AddTrackingEventParams{
		ID:          event.ID,
		ShipmentID:  event.ShipmentID,
		Status:      string(event.Status),
		Description: event.Description,
		Location:    event.Location,
		OccurredAt:  pgtype.Timestamptz{Time: event.OccurredAt, Valid: true},
		CreatedAt:   pgtype.Timestamptz{Time: event.CreatedAt, Valid: true},
	})
}

func (r *Repository) GetTrackingEvents(ctx context.Context, shipmentID string) ([]domain.TrackingEvent, error) {
	rows, err := r.reader.GetTrackingEvents(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	events := make([]domain.TrackingEvent, len(rows))
	for i, row := range rows {
		events[i] = domain.TrackingEvent{
			ID:          row.ID,
			ShipmentID:  row.ShipmentID,
			Status:      domain.ShipmentStatus(row.Status),
			Description: row.Description,
			Location:    row.Location,
			OccurredAt:  row.OccurredAt.Time,
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return events, nil
}

func parseDestinationSnapshot(data []byte) (domain.DestinationSnapshot, error) {
	var dest domain.DestinationSnapshot
	if len(data) == 0 {
		return dest, nil
	}
	err := json.Unmarshal(data, &dest)
	return dest, err
}

func clampInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}
