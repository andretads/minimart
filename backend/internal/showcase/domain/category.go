package domain

// Category is the public read-model view of a category exposed by the showcase domain.
// The showcase domain reads the categories table directly (read-only display layer).
type Category struct {
	ID   string
	Name string
	Icon string
}
