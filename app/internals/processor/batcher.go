package processor

import (
	"context"
	"sync"
	"time"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/pkg/logger"
)

// FlushFunc is called when the batcher decides it's time to flush.
type FlushFunc func(ctx context.Context, records []model.LogRecord) error

// Batcher accumulates log records and flushes them in bulk when either
// the batch size or the flush interval is reached.
type Batcher struct {
	maxSize       int
	flushInterval time.Duration
	flushFn       FlushFunc

	mu      sync.Mutex
	buf     []model.LogRecord
	timer   *time.Timer
	log     *logger.Logger
}

// NewBatcher creates a Batcher that will call flushFn whenever maxSize records
// accumulate or flushInterval elapses, whichever comes first.
func NewBatcher(maxSize int, flushInterval time.Duration, flushFn FlushFunc) *Batcher {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	b := &Batcher{
		maxSize:       maxSize,
		flushInterval: flushInterval,
		flushFn:       flushFn,
		buf:           make([]model.LogRecord, 0, maxSize),
		log:           logger.New("batcher"),
	}

	b.timer = time.AfterFunc(flushInterval, func() {
		b.Flush(context.Background())
	})

	return b
}

// Add appends records to the buffer and flushes if the batch is full.
func (b *Batcher) Add(ctx context.Context, records []model.LogRecord) error {
	b.mu.Lock()
	b.buf = append(b.buf, records...)
	shouldFlush := len(b.buf) >= b.maxSize
	b.mu.Unlock()

	if shouldFlush {
		return b.Flush(ctx)
	}
	return nil
}

// Flush sends the buffered records to the flush function and resets the buffer.
func (b *Batcher) Flush(ctx context.Context) error {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return nil
	}
	batch := b.buf
	b.buf = make([]model.LogRecord, 0, b.maxSize)
	b.resetTimer()
	b.mu.Unlock()

	b.log.Info("flushing batch of %d records", len(batch))

	if err := b.flushFn(ctx, batch); err != nil {
		b.log.Error("flush failed: %v", err)
		return err
	}

	return nil
}

// resetTimer resets the periodic flush timer. Must be called under lock.
func (b *Batcher) resetTimer() {
	b.timer.Reset(b.flushInterval)
}

// Stop flushes remaining records and stops the timer.
func (b *Batcher) Stop(ctx context.Context) {
	b.timer.Stop()
	_ = b.Flush(ctx)
}

