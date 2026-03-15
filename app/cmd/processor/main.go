package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	chclient "featuretrace.io/app/internals/storage/clickhouse"
	"featuretrace.io/app/internals/processor"
	"featuretrace.io/app/internals/queue"
	"featuretrace.io/app/internals/storage/repository"
	"featuretrace.io/app/pkg/logger"
)

func main() {
	log := logger.New("processor")

	// ── Configuration ───────────────────────────────────────────────
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	chAddr := envOr("CLICKHOUSE_ADDR", "localhost:9000")
	chDB := envOr("CLICKHOUSE_DB", "featuretrace")
	chUser := envOr("CLICKHOUSE_USER", "default")
	chPass := envOr("CLICKHOUSE_PASSWORD", "")
	batchSize := intEnvOr("BATCH_SIZE", 1000)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── ClickHouse ──────────────────────────────────────────────────
	ch, err := chclient.New(ctx, chclient.Config{
		Addr:     chAddr,
		Database: chDB,
		Username: chUser,
		Password: chPass,
	})
	if err != nil {
		log.Fatal("clickhouse: %v", err)
	}
	defer ch.Close()

	if err := ch.Migrate(ctx, chDB); err != nil {
		log.Fatal("migrate: %v", err)
	}

	repo := repository.NewClickHouseRepo(ch.Conn(), chDB)

	// ── NATS ────────────────────────────────────────────────────────
	natsConn, err := queue.NewNATSConn(natsURL)
	if err != nil {
		log.Fatal("nats: %v", err)
	}
	defer natsConn.Close()

	if err := natsConn.EnsureStream(ctx); err != nil {
		log.Fatal("ensure stream: %v", err)
	}

	// ── Worker ──────────────────────────────────────────────────────
	worker := processor.NewWorker(natsConn, repo, batchSize)

	// ── Graceful shutdown ───────────────────────────────────────────
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("shutting down processor...")
		worker.Stop(ctx)
		cancel()
	}()

	log.Info("processor worker starting (batch_size=%d)", batchSize)
	if err := worker.Start(ctx); err != nil && ctx.Err() == nil {
		log.Fatal("worker: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intEnvOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
