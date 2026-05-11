FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /machine-enrollment-worker ./cmd/machine-enrollment-worker

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /machine-enrollment-worker /usr/local/bin/machine-enrollment-worker
COPY config ./config

ENV MACHINE_ENROLLMENT_WORKER_CONFIG_FILE=/app/config/machine-enrollment-worker.yaml

ENTRYPOINT ["machine-enrollment-worker"]
