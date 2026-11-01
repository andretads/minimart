package moderation

import (
	"context"
	"time"
)

type NoopProvider struct{}

func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

func (n *NoopProvider) Name() string {
	return "noop"
}

func (n *NoopProvider) Moderate(ctx context.Context, content string) (Result, error) {
	return Result{
		Decision:     "approved",
		Confidence:   1.0,
		Categories:   []string{},
		Explanation:  "Moderation disabled (noop provider)",
		Model:        "noop",
		ModelVersion: "1.0",
		ModeratedAt:  time.Now().UTC(),
	}, nil
}
