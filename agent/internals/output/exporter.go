package output

import (
	"context"

	"featuretrace.io/agent/internals/model"
)

// Exporter is the contract for sending record batches to a backend.
type Exporter interface {
	// Send transmits a batch of records. Implementations handle retries internally.
	Send(ctx context.Context, batch []model.Record) error
}

