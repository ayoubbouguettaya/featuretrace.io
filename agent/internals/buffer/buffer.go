package buffer

import (
	"context"
	"log"
	"sync"
	"time"

	"featuretrace.io/agent/internals/model"
)

// Batcher collects Records and flushes them as batches, triggered by either:
//   - batch reaching MaxSize records, or
//   - FlushInterval elapsed since last flush.
//
// This prevents one-by-one network calls and protects the backend from bursts.
type Batcher struct {
	MaxSize       int
	FlushInterval time.Duration

	mu     sync.Mutex
	buf    []model.Record
}

// New creates a Batcher with the given limits.
func New(maxSize int, flushInterval time.Duration) *Batcher {
	return &Batcher{
		MaxSize:       maxSize,
		FlushInterval: flushInterval,
		buf:           make([]model.Record, 0, maxSize),
	}
}

// Run reads Records from in, batches them, and writes complete batches to out.
// It blocks until ctx is cancelled, then flushes any remaining records.
func (b *Batcher) Run(ctx context.Context, in <-chan model.Record, out chan<- []model.Record) {
	ticker := time.NewTicker(b.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case rec, ok := <-in:
			if !ok {
				// Input channel closed — final flush
				b.flush(out)
				return
			}
			b.mu.Lock()
			b.buf = append(b.buf, rec)
			full := len(b.buf) >= b.MaxSize
			b.mu.Unlock()

			if full {
				b.flush(out)
			}

		case <-ticker.C:
			b.flush(out)

		case <-ctx.Done():
			// Graceful shutdown — drain remaining records from input
			b.drainAndFlush(in, out)
			return
		}
	}
}

// flush sends the current buffer contents as a single batch and resets the buffer.
func (b *Batcher) flush(out chan<- []model.Record) {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buf
	b.buf = make([]model.Record, 0, b.MaxSize)
	b.mu.Unlock()

	log.Printf("[batcher] flushing batch of %d records", len(batch))
	out <- batch
}

// drainAndFlush reads any remaining records from in, adds them to the buffer,
// and performs a final flush. Used during graceful shutdown.
func (b *Batcher) drainAndFlush(in <-chan model.Record, out chan<- []model.Record) {
	for {
		select {
		case rec, ok := <-in:
			if !ok {
				break
			}
			b.mu.Lock()
			b.buf = append(b.buf, rec)
			b.mu.Unlock()
		default:
			b.flush(out)
			return
		}
	}
}

