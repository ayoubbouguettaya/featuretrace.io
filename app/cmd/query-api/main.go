package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"featuretrace.io/app/internals/query"
	chclient "featuretrace.io/app/internals/storage/clickhouse"
	"featuretrace.io/app/internals/storage/repository"
	"featuretrace.io/app/pkg/logger"
)

func main() {
	log := logger.New("query-api")

	// ── Configuration ───────────────────────────────────────────────
	httpAddr := envOr("HTTP_ADDR", ":3008")
	chAddr := envOr("CLICKHOUSE_ADDR", "localhost:9000")
	chDB := envOr("CLICKHOUSE_DB", "featuretrace")
	chUser := envOr("CLICKHOUSE_USER", "default")
	chPass := envOr("CLICKHOUSE_PASSWORD", "")

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

	repo := repository.NewClickHouseRepo(ch.Conn(), chDB)

	// ── Query service + handler ─────────────────────────────────────
	svc := query.NewService(repo)
	handler := query.NewHandler(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ───────────────────────────────────────────
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("shutting down HTTP server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
		cancel()
	}()

	log.Info("query API listening on %s", httpAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("serve: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
