// Package kafkaadapter provides a thin, typed consumer over
// github.com/segmentio/kafka-go for the worker. It decodes JSON message values
// into a caller-supplied type and dispatches them to a handler, keeping the
// Kafka wiring out of the core.
package kafkaadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// NewReader builds a kafka.Reader configured for consumer-group based,
// manual-commit consumption of a single topic. The byte bounds favour low
// latency while still allowing the broker to batch.
func NewReader(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,        // 1B: respond as soon as any data is available.
		MaxBytes: 10 << 20, // 10MB upper bound per fetch.
	})
}

// Consumer reads messages of type T off a kafka.Reader, unmarshals each value
// from JSON and hands it to handle. It is generic so a worker can wire, for
// example, Consumer[domain.AccountEvent].
type Consumer[T any] struct {
	reader *kafka.Reader
	handle func(ctx context.Context, msg T) error
	log    *slog.Logger
}

// NewConsumer returns a Consumer that dispatches decoded messages to handle.
func NewConsumer[T any](reader *kafka.Reader, handle func(ctx context.Context, msg T) error, log *slog.Logger) *Consumer[T] {
	return &Consumer[T]{
		reader: reader,
		handle: handle,
		log:    log,
	}
}

// Run consumes messages until ctx is cancelled, committing only after a message
// has been handled successfully (at-least-once delivery). Decode and handler
// errors are logged and the message is left uncommitted so it is redelivered;
// the loop continues. It returns nil on context cancellation and any other
// fatal reader error otherwise.
func (c *Consumer[T]) Run(ctx context.Context) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			// A cancelled context is the expected shutdown path.
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		var msg T
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			c.log.ErrorContext(ctx, "kafka: failed to unmarshal message",
				slog.String("topic", m.Topic),
				slog.Int("partition", m.Partition),
				slog.Int64("offset", m.Offset),
				slog.Any("error", err),
			)
			continue
		}

		if err := c.handle(ctx, msg); err != nil {
			c.log.ErrorContext(ctx, "kafka: failed to handle message",
				slog.String("topic", m.Topic),
				slog.Int("partition", m.Partition),
				slog.Int64("offset", m.Offset),
				slog.Any("error", err),
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.ErrorContext(ctx, "kafka: failed to commit message",
				slog.String("topic", m.Topic),
				slog.Int("partition", m.Partition),
				slog.Int64("offset", m.Offset),
				slog.Any("error", err),
			)
		}
	}
}

// Close releases the underlying reader and its consumer-group resources.
func (c *Consumer[T]) Close() error {
	return c.reader.Close()
}
