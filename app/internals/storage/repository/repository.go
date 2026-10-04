package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/pkg/logger"
)

// LogRepository defines the contract for persisting and querying log records.
type LogRepository interface {
	InsertBatch(ctx context.Context, records []model.LogRecord) error
	Query(ctx context.Context, filter LogFilter) ([]model.LogRecord, error)
	Facets(ctx context.Context) (Facets, error)
}

// LogFilter defines query parameters for the Query API.
type LogFilter struct {
	Service string
	Level   string
	Feature string
	TraceID string
	SpanID  string
	Search  string // free-text search on message
	From    time.Time
	To      time.Time
	Limit   int
	Offset  int
}

// ClickHouseRepo implements LogRepository backed by ClickHouse.
type ClickHouseRepo struct {
	conn     driver.Conn
	database string
	log      *logger.Logger
}

// NewClickHouseRepo creates a new repository using the given driver connection.
func NewClickHouseRepo(conn driver.Conn, database string) *ClickHouseRepo {
	if database == "" {
		database = "featuretrace"
	}
	return &ClickHouseRepo{
		conn:     conn,
		database: database,
		log:      logger.New("repository"),
	}
}

// InsertBatch performs a columnar batch insert of log records into ClickHouse.
func (r *ClickHouseRepo) InsertBatch(ctx context.Context, records []model.LogRecord) error {
	if len(records) == 0 {
		return nil
	}

	query := fmt.Sprintf(`INSERT INTO %s.logs (
		timestamp, message, level, service, feature,
		trace_id, span_id, source, container, metadata
	)`, r.database)

	r.log.Debug("inserting batch of %d records", len(records))

	batch, err := r.conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, rec := range records {
		meta := rec.Metadata
		if meta == nil {
			meta = map[string]string{}
		}
		if err := batch.Append(
			rec.Timestamp,
			rec.Message,
			rec.Level,
			rec.Service,
			rec.Feature,
			rec.TraceID,
			rec.SpanID,
			rec.Source,
			rec.Container,
			meta,
		); err != nil {
			return fmt.Errorf("append record: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}

	r.log.Debug("inserted %d records", len(records))
	return nil
}

// Query fetches log records matching the filter from ClickHouse.
func (r *ClickHouseRepo) Query(ctx context.Context, filter LogFilter) ([]model.LogRecord, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	q := fmt.Sprintf(`
		SELECT timestamp, message, level, service, feature,
		       trace_id, span_id, source, container, metadata
		FROM %s.logs
		WHERE 1=1
	`, r.database)

	args := make([]any, 0)

	if filter.Service != "" {
		q += " AND service = ?"
		args = append(args, filter.Service)
	}
	if filter.Level != "" {
		q += " AND level = ?"
		args = append(args, filter.Level)
	}
	if filter.Feature != "" {
		q += " AND feature = ?"
		args = append(args, filter.Feature)
	}
	if filter.TraceID != "" {
		q += " AND trace_id = ?"
		args = append(args, filter.TraceID)
	}
	if filter.SpanID != "" {
		q += " AND span_id = ?"
		args = append(args, filter.SpanID)
	}
	if filter.Search != "" {
		q += " AND message ILIKE ?"
		args = append(args, "%"+filter.Search+"%")
	}
	if !filter.From.IsZero() {
		q += " AND timestamp >= ?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		q += " AND timestamp <= ?"
		args = append(args, filter.To)
	}

	q += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	var results []model.LogRecord
	for rows.Next() {
		var rec model.LogRecord
		if err := rows.Scan(
			&rec.Timestamp,
			&rec.Message,
			&rec.Level,
			&rec.Service,
			&rec.Feature,
			&rec.TraceID,
			&rec.SpanID,
			&rec.Source,
			&rec.Container,
			&rec.Metadata,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		results = append(results, rec)
	}

	return results, nil
}

// Facets lists the distinct values the Query API can filter on.
type Facets struct {
	Services []string `json:"services"`
	Features []string `json:"features"`
}

// facetLimit caps each facet list; enough for a dropdown.
const facetLimit = 200

// Facets returns the distinct non-empty services and features in storage.
func (r *ClickHouseRepo) Facets(ctx context.Context) (Facets, error) {
	services, err := r.distinct(ctx, "service")
	if err != nil {
		return Facets{}, err
	}
	features, err := r.distinct(ctx, "feature")
	if err != nil {
		return Facets{}, err
	}
	return Facets{Services: services, Features: features}, nil
}

// distinct returns the sorted distinct non-empty values of column.
// column is always a constant from this package, never user input.
func (r *ClickHouseRepo) distinct(ctx context.Context, column string) ([]string, error) {
	q := fmt.Sprintf(
		"SELECT DISTINCT %[1]s FROM %[2]s.logs WHERE %[1]s != '' ORDER BY %[1]s LIMIT %[3]d",
		column, r.database, facetLimit,
	)

	rows, err := r.conn.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query distinct %s: %w", column, err)
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan %s: %w", column, err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
