package ingestion

import (
	"context"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/internals/queue"
	"featuretrace.io/app/pkg/logger"
)

// Enqueuer publishes validated log records to the NATS queue.
type Enqueuer struct {
	queueProducer *queue.Producer
	log           *logger.Logger
}

// NewEnqueuer wraps a NATS producer for ingestion use.
func NewEnqueuer(queueNatsConn *queue.NATSConnection) *Enqueuer {

	queueProducer := queue.NewProducer(queueNatsConn.JetStream)
	return &Enqueuer{
		queueProducer: queueProducer,
		log:           logger.New("enqueue"),
	}
}

// Enqueue publishes a validated batch of log records to the message queue.
func (e *Enqueuer) Enqueue(ctx context.Context, records []model.LogRecord) error {
	if err := e.queueProducer.Publish(ctx, records); err != nil {
		e.log.Error("failed to enqueue %d records: %v", len(records), err)
		return err
	}

	e.log.Debug("enqueued %d records", len(records))
	return nil
}
