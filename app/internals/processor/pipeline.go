package processor

import (
	"strings"
	"time"

	"featuretrace.io/app/internals/model"
)

// Stage is a single transformation step in the processing pipeline.
type Stage func(rec *model.LogRecord)

// Pipeline applies an ordered chain of Stages to each log record.
type Pipeline struct {
	stages []Stage
}

// NewPipeline creates a default processing pipeline with standard stages.
func NewPipeline() *Pipeline {
	return &Pipeline{
		stages: []Stage{
			normalizeLevel,
			ensureTimestamp,
			enrichMetadata,
		},
	}
}

// Process runs every record through all pipeline stages in order.
func (p *Pipeline) Process(records []model.LogRecord) []model.LogRecord {
	for i := range records {
		for _, stage := range p.stages {
			stage(&records[i])
		}
	}
	return records
}

// ── Built-in stages ─────────────────────────────────────────────────

// normalizeLevel lower-cases the level and maps common aliases.
func normalizeLevel(rec *model.LogRecord) {
	rec.Level = strings.ToLower(strings.TrimSpace(rec.Level))

	switch rec.Level {
	case "warning":
		rec.Level = "warn"
	case "err":
		rec.Level = "error"
	case "":
		rec.Level = "info"
	}
}

// ensureTimestamp sets the timestamp to now if it was zero-valued.
func ensureTimestamp(rec *model.LogRecord) {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().UTC()
	}
}

// enrichMetadata adds server-side metadata to the record.
func enrichMetadata(rec *model.LogRecord) {
	if rec.Metadata == nil {
		rec.Metadata = make(map[string]string)
	}
	rec.Metadata["processed_at"] = time.Now().UTC().Format(time.RFC3339)
}

