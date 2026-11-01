package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"syscall"

	kafkapkg "minimart/pkg/kafka"
	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
)

// RunOutboxReplay requeues abandoned outbox rows for the normal relays. It never
// sends to Kafka directly, so duplicate windows and per-key ordering stay intact.
func RunOutboxReplay(args []string) error {
	fs := flag.NewFlagSet("outbox-replay", flag.ContinueOnError)
	topic := fs.String("topic", "", "only replay rows from this source topic (empty = all)")
	maxRows := fs.Int("max", 100, "maximum abandoned rows to consider (0 = all)")
	dryRun := fs.Bool("dry-run", false, "log selected rows without requeueing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected outbox-replay arguments: %v", fs.Args())
	}
	if *maxRows < 0 {
		return errors.New("outbox-replay: -max must be zero or positive")
	}
	if *topic != "" && !slices.Contains(kafkapkg.AllTopics(), kafkapkg.Topic(*topic)) {
		return fmt.Errorf("outbox-replay: unknown topic %q", *topic)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := newWritePool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	replayer := outboxpkg.NewReplayer(outboxpg.NewStore(pool))
	result, err := replayer.Run(ctx, outboxpkg.ReplayOptions{
		Topic:  *topic,
		Limit:  *maxRows,
		DryRun: *dryRun,
	})
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "outbox replay finished",
		"topic", *topic,
		"selected", result.Selected,
		"requeued", result.Requeued,
		"dry_run", *dryRun,
	)
	return nil
}
