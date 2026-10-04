# FeatureTrace backend: one image holding all three services.
#
# Build from the repository root (go.mod and proto/ live there):
#   docker build -f app/Dockerfile -t featuretrace/app .
#
# Pick the service with the command:
#   docker run featuretrace/app ingest-api   # gRPC  :50051  agent → NATS
#   docker run featuretrace/app processor    #              NATS  → ClickHouse
#   docker run featuretrace/app query-api    # HTTP  :3008  ClickHouse → clients

FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY proto ./proto
COPY app ./app
RUN for svc in ingest-api processor query-api; do \
      CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/$svc ./app/cmd/$svc || exit 1; \
    done

FROM alpine:3.22
RUN addgroup -S featuretrace && adduser -S -G featuretrace featuretrace
COPY --from=build /out/ /usr/local/bin/
USER featuretrace

EXPOSE 50051 3008
CMD ["ingest-api"]
