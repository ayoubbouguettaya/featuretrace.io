package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/pkg/logger"
)

// MessageHandler is called for every batch of records consumed from NATS.
type MessageHandler func(ctx context.Context, records []model.LogRecord) error

// Consumer pulls messages from the JetStream durable consumer and dispatches
// them to a handler.
type Consumer struct {
	js      jetstream.JetStream
	handler MessageHandler
	log     *logger.Logger
}

// NewConsumer creates a Consumer that will dispatch to handler.
func NewConsumer(js jetstream.JetStream, handler MessageHandler) *Consumer {
	return &Consumer{
		js:      js,
		handler: handler,
		log:     logger.New("consumer"),
	}
}

// Start creates a durable consumer and begins consuming messages. It blocks
// until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	cons, err := c.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: SubjectRaw,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", ConsumerName, err)
	}

	c.log.Info("consuming from stream=%s subject=%s", StreamName, SubjectRaw)

	iter, err := cons.Messages(jetstream.PullMaxMessages(10))
	if err != nil {
		return fmt.Errorf("create message iterator: %w", err)
	}
	defer iter.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("consumer shutting down")
			return ctx.Err()
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			c.log.Error("next message: %v", err)
			continue
		}

		var records []model.LogRecord
		if err := json.Unmarshal(msg.Data(), &records); err != nil {
			c.log.Error("unmarshal message: %v", err)
			// Ack bad messages so they don't block the queue.
			_ = msg.Ack()
			continue
		}

		if err := c.handler(ctx, records); err != nil {
			c.log.Error("handler failed for %d records: %v", len(records), err)
			_ = msg.Nak()
			continue
		}

		_ = msg.Ack()
		c.log.Debug("processed batch of %d records", len(records))
	}
}

