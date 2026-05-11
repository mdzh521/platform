FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cluster-enrollment-worker ./cmd/cluster-enrollment-worker

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /cluster-enrollment-worker /usr/local/bin/cluster-enrollment-worker
COPY config ./config

ENV CLUSTER_ENROLLMENT_WORKER_CONFIG_FILE=/app/config/cluster-enrollment-worker.yaml

ENTRYPOINT ["cluster-enrollment-worker"]
