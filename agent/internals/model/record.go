package model

import "time"

// Record is the universal internal event flowing through the pipeline.
// Every pipeline stage reads and/or mutates a Record.
type Record struct {
	// Core log data
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`

	// Feature-aware observability fields
	Service string `json:"service,omitempty"`
	Feature string `json:"feature,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`

	// Source tracking
	Source    string `json:"source,omitempty"`
	Container string `json:"container,omitempty"`

	// Arbitrary key-value metadata attached by enrichment
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewRecord returns a Record with initialized metadata map and current timestamp.
func NewRecord() Record {
	return Record{
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]string),
	}
}

