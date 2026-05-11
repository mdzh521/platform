FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /resource-sync-worker ./cmd/resource-sync-worker

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /resource-sync-worker /usr/local/bin/resource-sync-worker
COPY config ./config

ENV SYNC_WORKER_CONFIG_FILE=/app/config/resource-sync-worker.yaml

ENTRYPOINT ["resource-sync-worker"]
