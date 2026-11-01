package cmd

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"minimart/conf"
	kafkapkg "minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"os/signal"
	"syscall"
	"time"
)

// RunDLQReplay replays <topic>.dlq back onto the original topic. It is an
// operator tool: the same binary, a different role. Progress is committed per
// message under a stable consumer group, so re-running resumes after the last
// replayed message.
func RunDLQReplay(args []string) error {
	fs := flag.NewFlagSet("dlq-replay", flag.ContinueOnError)
	topic := fs.String("topic", "", "original topic; the tool consumes <topic>.dlq (required)")
	group := fs.String("group", "dlq-replay", "consumer group used to remember replay progress")
	idleTimeout := fs.Duration("idle-timeout", 10*time.Second, "stop after this long without a message")
	maxMessages := fs.Int("max", 0, "maximum messages to replay per run (0 = all currently available)")
	maxReplays := fs.Int("max-replays", 3, "maximum times a single message may be replayed (0 = unlimited)")
	dryRun := fs.Bool("dry-run", false, "log messages without republishing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *topic == "" {
		return errors.New("dlq-replay: -topic is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	producer, err := kafkapkg.NewProducerWithAuth(conf.KafkaBrokers(), kafkapkg.Auth{
		Username:  conf.KafkaUsername(),
		Password:  conf.KafkaPassword(),
		Mechanism: conf.KafkaSASLMechanism(),
		EnableTLS: conf.KafkaTLSEnabled(),
	}, metrics.Noop{})
	if err != nil {
		return err
	}
	defer func() { _ = producer.Close() }()

	result, err := kafkapkg.Replay(ctx, conf.KafkaBrokers(), kafkapkg.Topic(*topic), kafkapkg.Topic(*topic), producer, kafkapkg.ReplayOptions{
		Auth: kafkapkg.Auth{
			Username:  conf.KafkaUsername(),
			Password:  conf.KafkaPassword(),
			Mechanism: conf.KafkaSASLMechanism(),
			EnableTLS: conf.KafkaTLSEnabled(),
		},
		Group:       *group,
		IdleTimeout: *idleTimeout,
		DryRun:      *dryRun,
		MaxMessages: *maxMessages,
		MaxReplays:  *maxReplays,
	})
	slog.Info("dlq replay finished",
		"topic", *topic,
		"replayed", result.Replayed,
		"skipped", result.Skipped,
		"parked", result.Parked,
		"dry_run", *dryRun,
	)
	return err
}
