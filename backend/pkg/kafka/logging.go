package kafka

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
)

// watermillLogAdapter bridges watermill's internal logs into the process slog
// logger, so Kafka client logs are emitted as the same JSON records as the rest
// of the service. Watermill's verbose info chatter is demoted to debug.
func watermillLogAdapter() watermill.LoggerAdapter {
	return watermill.NewSlogLoggerWithLevelMapping(slog.Default(), map[slog.Level]slog.Level{
		slog.LevelInfo: slog.LevelDebug,
	})
}
