# FeatureTrace Agent
#
# Build from the repository root (go.mod and proto/ live there):
#   docker build -f agent/Dockerfile -t featuretrace/agent .
#
# The agent tails Docker's JSON log files, so run it with the host's
# container log directory mounted read-only:
#   docker run -v /var/lib/docker/containers:/var/lib/docker/containers:ro featuretrace/agent

FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY proto ./proto
COPY agent ./agent
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/featuretrace-agent ./agent/cmd/agent

FROM alpine:3.22
COPY --from=build /out/featuretrace-agent /usr/local/bin/featuretrace-agent
COPY agent/config.yaml /etc/featuretrace/config.yaml
ENV FEATURETRACE_CONFIG=/etc/featuretrace/config.yaml

# Runs as root: Docker's log files under /var/lib/docker/containers are root-only.
ENTRYPOINT ["featuretrace-agent"]
