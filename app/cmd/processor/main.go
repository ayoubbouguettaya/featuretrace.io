package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"featuretrace.io/app/internals/processor"
	"featuretrace.io/app/internals/queue"
	clickHouseClient "featuretrace.io/app/internals/storage/clickhouse"
	"featuretrace.io/app/internals/storage/repository"
	"featuretrace.io/app/pkg/logger"
)

func main() {
	log := logger.New("processor")

	// ── Configuration ───────────────────────────────────────────────
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	clickHouseAddr := envOr("CLICKHOUSE_ADDR", "localhost:9000")
	clickHouseDB := envOr("CLICKHOUSE_DB", "featuretrace")
	clickHouseUser := envOr("CLICKHOUSE_USER", "default")
	clickHousePass := envOr("CLICKHOUSE_PASSWORD", "")
	batchSize := intEnvOr("BATCH_SIZE", 1000)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── ClickHouse ──────────────────────────────────────────────────
	clickHouseInstance, err := clickHouseClient.New(ctx, clickHouseClient.Config{
		Addr:     clickHouseAddr,
		Database: clickHouseDB,
		Username: clickHouseUser,
		Password: clickHousePass,
	})
	if err != nil {
		log.Fatal("clickhouse: %v", err)
	}
	defer clickHouseInstance.Close()

	if err := clickHouseInstance.Migrate(ctx, clickHouseDB); err != nil {
		log.Fatal("migrate: %v", err)
	}

	repo := repository.NewClickHouseRepo(clickHouseInstance.Conn(), clickHouseDB)

	// ── NATS ────────────────────────────────────────────────────────
	natsConnection, err := queue.NewNATSConnection(natsURL)
	if err != nil {
		log.Fatal("nats: %v", err)
	}
	defer natsConnection.Close()

	if err := natsConnection.EnsureStream(ctx); err != nil {
		log.Fatal("ensure stream: %v", err)
	}

	// ── Worker ──────────────────────────────────────────────────────
	worker := processor.NewWorker(natsConnection, repo, batchSize)

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
