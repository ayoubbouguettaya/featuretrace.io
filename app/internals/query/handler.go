package query

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"featuretrace.io/app/internals/storage/repository"
	"featuretrace.io/app/pkg/logger"
)

// Handler provides HTTP handlers for the Query API.
type Handler struct {
	svc *Service
	log *logger.Logger
}

// NewHandler creates a query HTTP handler backed by the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc: svc,
		log: logger.New("query-handler"),
	}
}

// RegisterRoutes registers all Query API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/logs", h.SearchLogs)
	mux.HandleFunc("GET /health", h.Health)
}

// SearchLogs handles GET /v1/logs with query parameters for filtering.
//
// Query params:
//
//	service, level, feature, search (free-text),
//	from, to (RFC3339), limit, offset.
func (h *Handler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := repository.LogFilter{
		Service: q.Get("service"),
		Level:   q.Get("level"),
		Feature: q.Get("feature"),
		Search:  q.Get("search"),
		Limit:   intParam(q.Get("limit"), 100),
		Offset:  intParam(q.Get("offset"), 0),
	}

	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = t
		}
	}
	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = t
		}
	}

	records, err := h.svc.Search(r.Context(), filter)
	if err != nil {
		h.log.Error("search error: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"count":   len(records),
		"records": records,
	}); err != nil {
		h.log.Error("encode response: %v", err)
	}
}

// Health is a simple liveness probe.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// intParam parses an int from a query string with a default fallback.
func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

