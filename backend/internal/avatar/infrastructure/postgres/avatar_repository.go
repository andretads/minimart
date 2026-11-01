package postgres

import (
	"context"
	"errors"
	"minimart/internal/avatar/domain"
	"minimart/internal/avatar/infrastructure/postgres/reader"
	"minimart/internal/avatar/infrastructure/postgres/writer"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AvatarRepository implements domain.Repository using Postgres.
type AvatarRepository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
}

var _ domain.Repository = (*AvatarRepository)(nil)

// NewAvatarRepository builds the repository with the provided read/write pools.
func NewAvatarRepository(readPool, writePool *pgxpool.Pool) *AvatarRepository {
	return &AvatarRepository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
	}
}

// GetByAccountID reads an avatar by account ID from the read pool.
// Returns ErrAvatarNotFound when no row exists.
func (r *AvatarRepository) GetByAccountID(ctx context.Context, accountID string) (domain.Avatar, error) {
	row, err := r.reader.GetAvatar(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Avatar{}, domain.ErrAvatarNotFound
		}
		return domain.Avatar{}, err
	}
	return toDomain(row), nil
}

// Save upserts an avatar using the write pool.
// It preserves created_at on conflict and increments the version.
func (r *AvatarRepository) Save(ctx context.Context, a domain.Avatar) (domain.Avatar, error) {
	row, err := r.writer.UpsertAvatar(ctx, writer.UpsertAvatarParams{
		AccountID:   a.AccountID,
		ObjectKey:   a.ObjectKey,
		Bucket:      a.Bucket,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		CreatedAt:   pgtype.Timestamptz{Time: a.CreatedAt, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: a.UpdatedAt, Valid: true},
	})
	if err != nil {
		return domain.Avatar{}, err
	}
	return toDomainFromWriter(row), nil
}

func toDomain(row reader.Avatar) domain.Avatar {
	return domain.Avatar{
		AccountID:   row.AccountID,
		ObjectKey:   row.ObjectKey,
		Bucket:      row.Bucket,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		Version:     int(row.Version),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func toDomainFromWriter(row writer.Avatar) domain.Avatar {
	return domain.Avatar{
		AccountID:   row.AccountID,
		ObjectKey:   row.ObjectKey,
		Bucket:      row.Bucket,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		Version:     int(row.Version),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
