package model

import "time"

// LogRecord is the canonical internal representation of a log entry flowing
// through the FeatureTrace backend pipeline. It mirrors the agent's Record.
type LogRecord struct {
	// Core log data
	Timestamp time.Time `json:"timestamp" ch:"timestamp"`
	Message   string    `json:"message"   ch:"message"`
	Level     string    `json:"level"     ch:"level"`

	// Feature-aware observability fields
	Service string `json:"service,omitempty"  ch:"service"`
	Feature string `json:"feature,omitempty"  ch:"feature"`
	TraceID string `json:"trace_id,omitempty" ch:"trace_id"`
	SpanID  string `json:"span_id,omitempty"  ch:"span_id"`

	// Source tracking
	Source    string `json:"source,omitempty"    ch:"source"`
	Container string `json:"container,omitempty" ch:"container"`

	// Arbitrary key-value metadata attached by enrichment
	Metadata map[string]string `json:"metadata,omitempty" ch:"metadata"`
}

// NewLogRecord returns a LogRecord with an initialised metadata map and the
// current UTC timestamp.
func NewLogRecord() LogRecord {
	return LogRecord{
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]string),
	}
}

