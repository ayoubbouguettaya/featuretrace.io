package output

import (
	"context"
	"log"
	"math"
	"time"

	"featuretrace.io/agent/internals/model"
	pb "featuretrace.io/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcExporter struct {
	Timeout      time.Duration
	MaxRetries   int
	Compression  bool
	IngestClient pb.IngestServiceClient
	ClientConn   *grpc.ClientConn
}

func NewGrpcExporter(timeout time.Duration, maxRetries int, compression bool) *GrpcExporter {
	addr := "localhost:50051"

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	ingestServiceClient := pb.NewIngestServiceClient(conn)

	return &GrpcExporter{
		Timeout:      timeout,
		MaxRetries:   maxRetries,
		Compression:  compression,
		ClientConn:   conn,
		IngestClient: ingestServiceClient,
	}
}

func recordsToProto(batch []model.Record) []*pb.LogRecord {
	out := make([]*pb.LogRecord, 0, len(batch))
	for _, r := range batch {
		out = append(out, &pb.LogRecord{
			TimestampNs: r.Timestamp.UnixNano(),
			Message:     r.Message,
			Level:       r.Level,
			Service:     r.Service,
			Feature:     r.Feature,
			TraceId:     r.TraceID,
			SpanId:      r.SpanID,
			Source:      r.Source,
			Container:   r.Container,
			Metadata:    r.Metadata,
		})
	}
	return out
}

// Send transmits a batch with retries and backpressure.
func (g *GrpcExporter) Send(ctx context.Context, batch []model.Record) error {

	var lastErr error
	for attempt := 0; attempt <= g.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
			log.Printf("[exporter] retry %d/%d in %s", attempt, g.MaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		_, lastErr = g.IngestClient.SendLogs(ctx, &pb.SendLogsRequest{Records: recordsToProto(batch)})

		if lastErr == nil {
			log.Printf("[exporter] sent batch of %d records", len(batch))
			return nil
		}

		log.Printf("[exporter] attempt %d failed: %v", attempt+1, lastErr)
	}

	// All retries exhausted — drop batch (backpressure: never crash)
	log.Printf("[exporter] dropping batch of %d records after %d retries: %v",
		len(batch), g.MaxRetries, lastErr)
	return lastErr
}
