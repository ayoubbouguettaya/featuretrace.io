package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"featuretrace.io/app/internals/ingestion"
	"featuretrace.io/app/internals/queue"
	"featuretrace.io/app/pkg/logger"
	pb "featuretrace.io/app/proto/v1"
)

func main() {
	log := logger.New("ingest-api")

	// ── Configuration (env vars with defaults) ──────────────────────
	grpcAddr := envOr("GRPC_ADDR", ":50051")
	natsURL := envOr("NATS_URL", "nats://localhost:4222")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── NATS connection ─────────────────────────────────────────────
	natsConn, err := queue.NewNATSConnection(natsURL)
	if err != nil {
		log.Fatal("nats: %v", err)
	}
	defer natsConn.Close()

	if err := natsConn.EnsureStream(ctx); err != nil {
		log.Fatal("ensure stream: %v", err)
	}

	// ── Wire ingestion pipeline ─────────────────────────────────────
	producer := queue.NewProducer(natsConn.JetStream)
	enqueuer := ingestion.NewEnqueuer(producer)
	handler := ingestion.NewHandler(enqueuer)

	// ── gRPC server ─────────────────────────────────────────────────
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("listen %s: %v", grpcAddr, err)
	}

	srv := grpc.NewServer()
	pb.RegisterIngestServiceServer(srv, handler)
	reflection.Register(srv) // enable grpcurl / grpcui

	// ── Graceful shutdown ───────────────────────────────────────────
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("shutting down gRPC server...")
		srv.GracefulStop()
		cancel()
	}()

	log.Info("gRPC ingest API listening on %s", grpcAddr)
	if err := srv.Serve(lis); err != nil {
		log.Fatal("serve: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
