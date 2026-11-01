package domain

import "time"

// Cursor is the keyset pagination token's decoded payload. It anchors the next
// page to the last item of the previous page using the same sort order
// (created_at DESC, id DESC).
type Cursor struct {
	CreatedAt time.Time
	ID        string
}
