package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	wkafka "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	wmessage "github.com/ThreeDotsLabs/watermill/message"
)

// ReplayOptions controls a dead-letter replay run.
type ReplayOptions struct {
	// Auth carries optional Kafka authentication settings.
	Auth Auth
	// Group is the consumer group used to remember replay progress. The same
	// group resumes where the previous run stopped. Dry-run uses a separate
	// "<group>-dry-run" group so it never advances the real one.
	Group string
	// IdleTimeout stops the run after this long without a message. DLQ topics do
	// not signal "caught up", so this is how a replay run terminates.
	IdleTimeout time.Duration
	// DryRun logs messages without republishing them.
	DryRun bool
	// MaxMessages stops after this many messages (0 = all currently available).
	// The next unprocessed message is left unacknowledged for the next run.
	MaxMessages int
	// MaxReplays caps how many times a single DLQ message may be replayed. When
	// the cap is reached the message is parked (acked without republishing) so
	// it cannot loop forever. Zero disables the cap.
	MaxReplays int
}

// ReplayResult reports what a replay run did.
type ReplayResult struct {
	Replayed int
	Skipped  int
	Parked   int
}

// Replay consumes source's dead-letter topic and republishes each message to
// target, preserving the original key and payload. Dead-letter metadata is
// stripped; a replay marker is added instead. Progress is committed per message,
// so an interrupted run resumes safely.
func Replay(ctx context.Context, brokers []string, source Topic, target Topic, publisher HeaderPublisher, opts ReplayOptions) (ReplayResult, error) {
	var result ReplayResult
	if publisher == nil {
		return result, errors.New("dlq replay: publisher is nil")
	}
	if target == "" {
		return result, errors.New("dlq replay: target topic is empty")
	}
	if opts.Group == "" {
		opts.Group = "dlq-replay"
	}
	if opts.IdleTimeout <= 0 {
		opts.IdleTimeout = 10 * time.Second
	}
	if opts.DryRun {
		opts.Group += "-dry-run"
	}

	cfg := wkafka.DefaultSaramaSubscriberConfig()
	cfg.Version = sarama.V2_1_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	// Commit on every ack so a stopped run resumes at the next message instead of
	// replaying the whole DLQ.
	cfg.Consumer.Offsets.AutoCommit.Enable = false
	applyAuth(cfg, opts.Auth)

	sub, err := wkafka.NewSubscriber(wkafka.SubscriberConfig{
		Brokers:               brokers,
		Unmarshaler:           wkafka.DefaultMarshaler{},
		OverwriteSaramaConfig: cfg,
		ConsumerGroup:         opts.Group,
	}, watermill.NopLogger{})
	if err != nil {
		return result, fmt.Errorf("dlq replay: new subscriber: %w", err)
	}
	defer func() { _ = sub.Close() }()

	sourceTopic := DLQTopic(source)
	messages, err := sub.Subscribe(ctx, sourceTopic.String())
	if err != nil {
		return result, fmt.Errorf("dlq replay: subscribe %s: %w", sourceTopic, err)
	}

	idle := time.NewTimer(opts.IdleTimeout)
	defer idle.Stop()

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return result, nil
			}
			resetTimer(idle, opts.IdleTimeout)

			if opts.MaxMessages > 0 && result.Replayed+result.Skipped >= opts.MaxMessages {
				// Leave this message unacked so the next run starts with it.
				return result, nil
			}

			key := msg.Metadata.Get("key")
			if opts.DryRun {
				slog.Info("dlq replay dry run",
					"source", sourceTopic.String(),
					"target", target.String(),
					"key", key,
					"payload_bytes", len(msg.Payload),
					"headers", replayHeaders(msg.Metadata, sourceTopic),
				)
				result.Skipped++
				msg.Ack()
				continue
			}

			replayCount := replayCountOf(msg.Metadata)
			if opts.MaxReplays > 0 && replayCount >= opts.MaxReplays {
				slog.Warn("dlq replay parked: max replays reached",
					"source", sourceTopic.String(),
					"target", target.String(),
					"key", key,
					"replay_count", replayCount,
				)
				result.Parked++
				msg.Ack()
				continue
			}

			if err := publishReplay(ctx, target, sourceTopic, key, msg, publisher); err != nil {
				return result, err
			}
			result.Replayed++
			msg.Ack()
		case <-idle.C:
			return result, nil
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}
}

func publishReplay(ctx context.Context, target, source Topic, key string, msg *wmessage.Message, publisher HeaderPublisher) error {
	if err := publisher.Publish(ctx, Message{
		Topic:   target,
		Key:     key,
		Payload: msg.Payload, // unchanged, so consumers see the original event
		Headers: replayHeaders(msg.Metadata, source),
	}); err != nil {
		return fmt.Errorf("dlq replay: publish to %s: %w", target, err)
	}
	return nil
}

func replayHeaders(metadata map[string]string, source Topic) map[string]string {
	headers := make(map[string]string, len(metadata)+3)
	for k, v := range metadata {
		if strings.HasPrefix(k, minimartHeaderPrefix) {
			continue
		}
		headers[k] = v
	}
	headers[minimartHeaderPrefix+"replayed-from"] = source.String()
	headers[minimartHeaderPrefix+"replayed-at"] = time.Now().UTC().Format(time.RFC3339Nano)
	headers[DLQHeaderReplayCount] = strconv.Itoa(replayCountOf(metadata) + 1)
	return headers
}

func replayCountOf(metadata map[string]string) int {
	n, _ := strconv.Atoi(metadata[DLQHeaderReplayCount])
	return n
}

func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}
