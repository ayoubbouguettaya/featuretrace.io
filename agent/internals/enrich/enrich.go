package enrich

import (
	"os"
	"strings"

	"featuretrace.io/agent/internals/model"
)

// Enricher attaches host-level and container metadata to every Record flowing
// through the pipeline. It also performs trace-correlation: if the record
// already carries trace_id / feature / span_id (injected by the SDK) the
// enricher preserves them; otherwise it tries to extract them from metadata.
type Enricher struct {
	hostname string
	labels   map[string]string // static labels from config
}

// New creates an Enricher, caching immutable host info once.
func New(labels map[string]string) *Enricher {
	host, _ := os.Hostname()
	if labels == nil {
		labels = make(map[string]string)
	}
	return &Enricher{
		hostname: host,
		labels:   labels,
	}
}

// Enrich mutates a Record in-place, adding environment context and ensuring
// trace-correlation fields are populated.
func (e *Enricher) Enrich(rec *model.Record) {
	if rec.Metadata == nil {
		rec.Metadata = make(map[string]string)
	}

	// --- Host metadata ---
	rec.Metadata["host"] = e.hostname

	// --- Static config labels ---
	for k, v := range e.labels {
		if _, exists := rec.Metadata[k]; !exists {
			rec.Metadata[k] = v
		}
	}

	// --- Service detection ---
	if rec.Service == "" {
		rec.Service = e.detectService(rec)
	}

	// --- Trace correlation (the FeatureTrace secret weapon) ---
	e.correlate(rec)
}

// detectService tries to infer the service name from container metadata,
// environment, or the record's own fields.
func (e *Enricher) detectService(rec *model.Record) string {
	// 1. Container name embedded in metadata by input layer
	if v := rec.Metadata["container_name"]; v != "" {
		return sanitizeServiceName(v)
	}

	// 2. Environment variable set by orchestrator
	if v := os.Getenv("FEATURETRACE_SERVICE"); v != "" {
		return v
	}

	// 3. Config label
	if v := e.labels["service"]; v != "" {
		return v
	}

	return "unknown"
}

// correlate ensures trace_id, feature, and span_id are promoted to first-class
// Record fields even if they were only present in Metadata.
func (e *Enricher) correlate(rec *model.Record) {
	promote := func(dst *string, keys ...string) {
		if *dst != "" {
			return
		}
		for _, k := range keys {
			if v, ok := rec.Metadata[k]; ok && v != "" {
				*dst = v
				return
			}
		}
	}

	promote(&rec.TraceID, "trace_id", "traceid", "x-trace-id")
	promote(&rec.SpanID, "span_id", "spanid", "x-span-id")
	promote(&rec.Feature, "feature", "x-feature-flag", "feature_flag")
}

// sanitizeServiceName cleans up container names (Docker prefixes with /).
func sanitizeServiceName(name string) string {
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimSpace(name)
	return name
}

