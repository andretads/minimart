package domain

import "context"

// Repository is the persistence port for avatars. The domain owns this
// interface; adapters under infrastructure/ implement it.
type Repository interface {
	// GetByAccountID returns the avatar by account ID, or ErrAvatarNotFound.
	GetByAccountID(ctx context.Context, accountID string) (Avatar, error)

	// Save inserts or updates an avatar. On insert it sets version to 1.
	// On update it increments the version and updates updated_at.
	// Returns the persisted avatar with the new version and timestamps.
	Save(ctx context.Context, a Avatar) (Avatar, error)
}
