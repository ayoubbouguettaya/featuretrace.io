# FeatureTrace.io

```go build -o app ./cmd/api/main.go && ./app ```

```go build -o agent ./cmd/agent/main.go && sudo ./agent```


## 1. FeatureTrace Backend

### 1.1 Project Structure

app/
├── cmd/
│   ├── api/              # ingestion API
│   ├── processor/        # queue consumers
│   └── worker/           # alerting / aggregation workers
│
├── internal/
│   ├── ingestion/
│   ├── processor/
│   ├── storage/
│   │   ├── clickhouse/
│   │   └── models/
│   ├── queue/
│   ├── tracing/
│   ├── alerts/
│   └── metrics/
│
├── pkg/                  # reusable libraries (if needed)
├── api/                  # protobuf / openapi specs
├── deployments/          # docker / k8s manifests
├── scripts/
├── web/                  # React frontend
└── go.mod


## 2. FeatureTrace Agent


### 2.1 Project Structure
agent/
├── cmd/agent/
│   └── main.go
│
├── internal/
│   ├── pipeline/
│   │   ├── engine.go
│   │   └── record.go
│   │
│   ├── input/
│   │   ├── docker/
│   │   └── file/
│   │
│   ├── parser/
│   │   ├── json.go
│   │   └── text.go
│   │
│   ├── enrich/
│   │   └── metadata.go
│   │
│   ├── buffer/
│   │   └── batcher.go
│   │
│   ├── output/
│   │   └── http.go
│   │
│   └── config/
│       └── config.go

### 2.2 Flow


                ┌─────────────┐
                │   INPUTS     │
                │ docker/file  │
                └──────┬───────┘
                       ↓
                ┌─────────────┐
                │   PARSER     │
                └──────┬───────┘
                       ↓
                ┌─────────────┐
                │ ENRICHMENT   │
                └──────┬───────┘
                       ↓
                ┌─────────────┐
                │ CORRELATION  │
                └──────┬───────┘
                       ↓
                ┌─────────────┐
                │   BUFFER     │
                └──────┬───────┘
                       ↓
                ┌─────────────┐
                │  EXPORTER    │
                └─────────────┘



## Docs:

https://dev.to/souvikinator/understanding-goroutines-and-channels-in-golang-with-intuitive-visuals-16gj


https://go.dev/blog/pipelines


## Tools:

- grpc: https://pkg.go.dev/google.golang.org/grpc

- Nats: https://docs.nats.io/

- Clickhouse: https://clickhouse.com/docs

