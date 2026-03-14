.PHONY: proto build run-ingest run-processor run-query clean

# ── Proto code generation ────────────────────────────────────────────
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/v1/ingest.proto

# ── Build all binaries ───────────────────────────────────────────────
build:
	go build -o bin/ingest-api   ./cmd/ingest-api
	go build -o bin/processor    ./cmd/processor
	go build -o bin/query-api    ./cmd/query-api

# ── Run individual services ──────────────────────────────────────────
run-ingest:
	go run ./cmd/ingest-api

run-processor:
	go run ./cmd/processor

run-query:
	go run ./cmd/query-api

# ── Infrastructure ───────────────────────────────────────────────────
infra-up:
	docker compose -f deployments/docker-compose.yml up -d

infra-down:
	docker compose -f deployments/docker-compose.yml down

# ── Clean ────────────────────────────────────────────────────────────
clean:
	rm -rf bin/

