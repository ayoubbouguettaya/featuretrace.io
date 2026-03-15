package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"featuretrace.io/app/pkg/logger"
)

const (
	// StreamName is the JetStream stream that holds raw ingested logs.
	StreamName = "LOGS"
	// SubjectRaw is the NATS subject for raw (unprocessed) log batches.
	SubjectRaw = "logs.raw"
	// ConsumerName is the durable consumer name used by processor workers.
	ConsumerName = "log-processor"
)

// NATSConn wraps a NATS connection and a JetStream context.
type NATSConn struct {
	Conn      *nats.Conn
	JetStream jetstream.JetStream
	log       *logger.Logger
}

// NewNATSConn connects to NATS and provisions the JetStream stream.
func NewNATSConn(url string) (*NATSConn, error) {
	lg := logger.New("nats")

	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			lg.Warn("disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			lg.Info("reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	lg.Info("connected to NATS at %s", url)

	return &NATSConn{
		Conn:      nc,
		JetStream: js,
		log:       lg,
	}, nil
}

// EnsureStream creates or updates the JetStream stream for log ingestion.
func (n *NATSConn) EnsureStream(ctx context.Context) error {
	cfg := jetstream.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{"logs.>"},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.FileStorage,
	}

	_, err := n.JetStream.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("ensure stream %s: %w", StreamName, err)
	}

	n.log.Info("stream %s ready", StreamName)
	return nil
}

// Close gracefully drains and closes the NATS connection.
func (n *NATSConn) Close() {
	if n.Conn != nil {
		n.Conn.Drain()
	}
}
