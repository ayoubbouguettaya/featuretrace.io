package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/pkg/logger"
)

// Producer publishes log record batches to NATS JetStream.
type Producer struct {
	js  jetstream.JetStream
	log *logger.Logger
}

// NewProducer creates a Producer bound to the given JetStream context.
func NewProducer(js jetstream.JetStream) *Producer {
	return &Producer{
		js:  js,
		log: logger.New("producer"),
	}
}

// Publish serialises the records as JSON and publishes them to the raw logs
// subject. JetStream guarantees at-least-once delivery.
func (p *Producer) Publish(ctx context.Context, records []model.LogRecord) error {
	data, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal records: %w", err)
	}

	ack, err := p.js.Publish(ctx, SubjectRaw, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", SubjectRaw, err)
	}

	p.log.Debug("published %d records (stream=%s seq=%d)", len(records), ack.Stream, ack.Sequence)
	return nil
}

