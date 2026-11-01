package moderation

import (
	"context"
	"time"
)

type Result struct {
	Decision     string   // "approved" | "rejected" | "needs_review"
	Confidence   float64  // 0.0 - 1.0
	Categories   []string // e.g., ["hate_speech", "profanity"]
	Explanation  string   // Human-readable explanation
	Model        string   // e.g., "llama-guard3"
	ModelVersion string   // e.g., "1b"
	ModeratedAt  time.Time
}

type Provider interface {
	Moderate(ctx context.Context, content string) (Result, error)
	Name() string
}
