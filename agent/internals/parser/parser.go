package parser

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"featuretrace.io/agent/internals/input"
	"featuretrace.io/agent/internals/model"
)

// dockerLogEntry represents the JSON envelope Docker wraps around every
// container log line:
//
//	{"log":"...actual message...\n","stream":"stdout","time":"..."}
type dockerLogEntry struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
	Time   string `json:"time"`
}

// Parse converts a raw log line (as read from a Docker JSON-log file) into a
// structured Record.
//
// It first unwraps the Docker JSON envelope, then attempts to parse the inner
// message as structured JSON (to extract level, trace_id, etc.). If the inner
// message is plain text it falls back to a simple text record.
func Parse(raw input.RawLog) model.Record {
	rec := model.NewRecord()

	raw.Data = bytes.TrimSpace(raw.Data)
	if len(raw.Data) == 0 {
		return rec
	}

	// --- Step 1: unwrap Docker JSON envelope ---
	var docker dockerLogEntry
	if err := json.Unmarshal(raw.Data, &docker); err == nil && docker.Log != "" {
		raw.Data = []byte(strings.TrimRight(docker.Log, "\n"))
		rec.Source = docker.Stream
		if t, tErr := time.Parse(time.RFC3339Nano, docker.Time); tErr == nil {
			rec.Timestamp = t
		}
	}

	// --- Step 2: try structured JSON inside the log message ---
	var fields map[string]interface{}
	if err := json.Unmarshal(raw.Data, &fields); err == nil {
		rec = applyJSONFields(rec, fields)
		return rec
	}

	// --- Step 3: fallback — plain text ---
	rec.Message = string(raw.Data)
	rec.Level = inferLevel(rec.Message)

	rec.Metadata["container_id"] = raw.ContainerID
	rec.Metadata["container_name"] = raw.ContainerName
	// b, _ := json.MarshalIndent(rec, "", "  ")
	// fmt.Println(string(b))

	return rec
}

// applyJSONFields maps well-known JSON keys to the Record.
func applyJSONFields(rec model.Record, fields map[string]interface{}) model.Record {
	if rec.Metadata == nil {
		rec.Metadata = make(map[string]string)
	}

	for key, val := range fields {
		str, ok := val.(string)
		if !ok {
			continue
		}

		switch strings.ToLower(key) {
		case "msg", "message":
			rec.Message = str
		case "level", "severity", "lvl":
			rec.Level = normalizeLevel(str)
		case "service", "service_name":
			rec.Service = str
		case "feature":
			rec.Feature = str
		case "trace_id", "traceid":
			rec.TraceID = str
		case "span_id", "spanid":
			rec.SpanID = str
		case "timestamp", "time", "ts":
			if t, err := time.Parse(time.RFC3339Nano, str); err == nil {
				rec.Timestamp = t
			}
		default:
			rec.Metadata[key] = str
		}
	}

	if rec.Message == "" {
		// Some apps log the full line as "log" or "text"
		for _, k := range []string{"log", "text"} {
			if v, ok := fields[k].(string); ok && v != "" {
				rec.Message = v
				break
			}
		}
	}

	return rec
}

// normalizeLevel maps common level strings to a canonical form.
func normalizeLevel(raw string) string {
	switch strings.ToUpper(raw) {
	case "DBG", "DEBUG", "TRACE":
		return "debug"
	case "INF", "INFO":
		return "info"
	case "WRN", "WARN", "WARNING":
		return "warn"
	case "ERR", "ERROR":
		return "error"
	case "FTL", "FATAL", "CRITICAL", "CRIT":
		return "fatal"
	default:
		return strings.ToLower(raw)
	}
}

// inferLevel guesses a log level from plain-text content.
func inferLevel(msg string) string {
	upper := strings.ToUpper(msg)
	switch {
	case strings.Contains(upper, "FATAL") || strings.Contains(upper, "CRITICAL"):
		return "fatal"
	case strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERR"):
		return "error"
	case strings.Contains(upper, "WARN"):
		return "warn"
	case strings.Contains(upper, "DEBUG"):
		return "debug"
	default:
		return "info"
	}
}
