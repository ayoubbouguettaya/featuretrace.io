package ingestion

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"featuretrace.io/app/internals/model"
	"featuretrace.io/app/pkg/logger"
	pb "featuretrace.io/proto/v1"
)

// Handler implements the gRPC IngestServiceServer interface.
type Handler struct {
	pb.UnimplementedIngestServiceServer
	enqueuer *Enqueuer
	log      *logger.Logger
}

// NewHandler creates the ingestion gRPC handler.
func NewHandler(enqueuer *Enqueuer) *Handler {
	return &Handler{
		enqueuer: enqueuer,
		log:      logger.New("ingest-handler"),
	}
}

// SendLogs receives a batch of log records from an agent, validates them,
// and publishes the valid records to the message queue.
func (h *Handler) SendLogs(ctx context.Context, req *pb.SendLogsRequest) (*pb.SendLogsResponse, error) {
	if len(req.GetRecords()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty batch")
	}

	valid := make([]model.LogRecord, 0, len(req.GetRecords()))

	for _, pbRec := range req.GetRecords() {
		rec := protoToModel(pbRec)

		if err := ValidateRecord(&rec); err != nil {
			h.log.Warn("dropping invalid record: %v", err)
			continue
		}

		valid = append(valid, rec)
	}

	if len(valid) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no valid records in batch")
	}

	if err := h.enqueuer.Enqueue(ctx, valid); err != nil {
		h.log.Error("enqueue failed: %v", err)
		return nil, status.Error(codes.Internal, "failed to enqueue logs")
	}

	h.log.Info("accepted %d/%d records", len(valid), len(req.GetRecords()))

	return &pb.SendLogsResponse{
		Accepted: int32(len(valid)),
	}, nil
}

// protoToModel converts a protobuf LogRecord to the internal model.
func protoToModel(pbRec *pb.LogRecord) model.LogRecord {
	meta := pbRec.GetMetadata()
	if meta == nil {
		meta = map[string]string{}
	}

	return model.LogRecord{
		Timestamp: time.Unix(0, pbRec.GetTimestampNs()).UTC(),
		Message:   pbRec.GetMessage(),
		Level:     pbRec.GetLevel(),
		Service:   pbRec.GetService(),
		Feature:   pbRec.GetFeature(),
		TraceID:   pbRec.GetTraceId(),
		SpanID:    pbRec.GetSpanId(),
		Source:    pbRec.GetSource(),
		Container: pbRec.GetContainer(),
		Metadata:  meta,
	}
}
