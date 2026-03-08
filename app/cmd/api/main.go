package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Record mirrors the agent's model.Record so we can deserialise incoming
// batches. Keep in sync with agent/internals/model/record.go.
type Record struct {
	Timestamp time.Time         `json:"timestamp"`
	Message   string            `json:"message"`
	Level     string            `json:"level"`
	Service   string            `json:"service,omitempty"`
	Feature   string            `json:"feature,omitempty"`
	TraceID   string            `json:"trace_id,omitempty"`
	SpanID    string            `json:"span_id,omitempty"`
	Source    string            `json:"source,omitempty"`
	Container string            `json:"container,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func main() {
	http.HandleFunc("/ingest/logs", handleIngestLogs)

	log.Println("Starting server on port 3010")
	log.Fatal(http.ListenAndServe(":3010", nil))
}

func handleIngestLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If the agent sent gzip-compressed data, decompress transparently.
	var reader io.ReadCloser
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "failed to decompress gzip body: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()
		reader = gz
	default:
		reader = r.Body
	}
	defer r.Body.Close()

	// Decode the JSON array of records.
	var records []Record
	if err := json.NewDecoder(reader).Decode(&records); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Pretty-print the batch so you can inspect it in the console.
	pretty, _ := json.MarshalIndent(records, "", "  ")
	log.Printf("[ingest] received batch of %d records:\n%s", len(records), pretty)

	// TODO: persist records to database here.
	// e.g. db.InsertBatch(ctx, records)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logs ingested"))
}
