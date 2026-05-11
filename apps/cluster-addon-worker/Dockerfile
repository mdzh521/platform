FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cluster-addon-worker ./cmd/cluster-addon-worker

FROM alpine:3.22

RUN apk add --no-cache ca-certificates kubectl helm

WORKDIR /app

COPY --from=builder /cluster-addon-worker /usr/local/bin/cluster-addon-worker
COPY config ./config

ENV CLUSTER_ADDON_WORKER_CONFIG_FILE=/app/config/cluster-addon-worker.yaml

ENTRYPOINT ["cluster-addon-worker"]
