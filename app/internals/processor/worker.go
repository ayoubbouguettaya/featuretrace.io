package processor

import (
	"context"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/internals/queue"
	"featuretrace.io/app/internals/storage/repository"
	"featuretrace.io/app/pkg/logger"
)

// Worker consumes log batches from NATS, processes them through the pipeline,
// and writes them to ClickHouse via the batcher.
type Worker struct {
	consumer *queue.Consumer
	pipeline *Pipeline
	batcher  *Batcher
	log      *logger.Logger
}

// NewWorker creates a processor worker wired to a NATS consumer and a
// ClickHouse repository.
func NewWorker(
	natsConn *queue.NATSConn,
	repo repository.LogRepository,
	batchSize int,
) *Worker {
	pipeline := NewPipeline()
	log := logger.New("worker")

	// The batcher flushes to ClickHouse.
	batcher := NewBatcher(batchSize, 0, func(ctx context.Context, records []model.LogRecord) error {
		return repo.InsertBatch(ctx, records)
	})

	w := &Worker{
		pipeline: pipeline,
		batcher:  batcher,
		log:      log,
	}

	// Wire the NATS consumer to our handler method.
	w.consumer = queue.NewConsumer(natsConn.JetStream, w.handle)

	return w
}

// Start begins consuming from NATS. Blocks until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	w.log.Info("processor worker starting")
	return w.consumer.Start(ctx)
}

// Stop flushes any remaining buffered records.
func (w *Worker) Stop(ctx context.Context) {
	w.batcher.Stop(ctx)
	w.log.Info("processor worker stopped")
}

// handle is the MessageHandler called for each consumed batch.
func (w *Worker) handle(ctx context.Context, records []model.LogRecord) error {
	// Run through processing pipeline.
	processed := w.pipeline.Process(records)

	// Buffer for bulk insertion.
	if err := w.batcher.Add(ctx, processed); err != nil {
		return err
	}

	w.log.Debug("handled batch of %d records", len(processed))
	return nil
}

