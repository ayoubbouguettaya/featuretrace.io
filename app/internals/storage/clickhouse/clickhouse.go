package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"featuretrace.io/app/pkg/logger"
)

// Config holds ClickHouse connection parameters.
type Config struct {
	Addr     string // host:port (default "localhost:9000")
	Database string // default "featuretrace"
	Username string
	Password string
}

// Client wraps the ClickHouse driver connection.
type Client struct {
	conn driver.Conn
	log  *logger.Logger
}

// New opens a ClickHouse connection and pings it.
func New(ctx context.Context, cfg Config) (*Client, error) {
	lg := logger.New("clickhouse")

	if cfg.Addr == "" {
		cfg.Addr = "localhost:9000"
	}
	if cfg.Database == "" {
		cfg.Database = "featuretrace"
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	lg.Info("connected to ClickHouse at %s/%s", cfg.Addr, cfg.Database)

	return &Client{conn: conn, log: lg}, nil
}

// Migrate creates the database and the logs table if they do not exist.
func (c *Client) Migrate(ctx context.Context, database string) error {
	if database == "" {
		database = "featuretrace"
	}

	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database)
	if err := c.conn.Exec(ctx, createDB); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	createTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.logs (
			timestamp   DateTime64(3, 'UTC'),
			message     String,
			level       LowCardinality(String),
			service     LowCardinality(String),
			feature     String        DEFAULT '',
			trace_id    String        DEFAULT '',
			span_id     String        DEFAULT '',
			source      LowCardinality(String),
			container   String        DEFAULT '',
			metadata    Map(String, String)
		)
		ENGINE = MergeTree()
		ORDER BY (service, level, timestamp)
		PARTITION BY toYYYYMMDD(timestamp)
		TTL toDateTime(timestamp) + INTERVAL 30 DAY
	`, database)

	if err := c.conn.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("create logs table: %w", err)
	}

	c.log.Info("migrations applied for %s.logs", database)
	return nil
}

// Conn returns the underlying driver connection for direct use by
// the repository layer.
func (c *Client) Conn() driver.Conn {
	return c.conn
}

// Close closes the ClickHouse connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

