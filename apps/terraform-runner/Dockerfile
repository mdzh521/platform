FROM hashicorp/terraform:1.11.4 AS terraform

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /terraform-runner ./cmd/terraform-runner

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=terraform /bin/terraform /usr/local/bin/terraform
COPY --from=builder /terraform-runner /usr/local/bin/terraform-runner
COPY templates ./templates
COPY config ./config

ENV RUNNER_LISTEN_ADDR=:8091
ENV RUNNER_POLL_INTERVAL=10s
ENV TERRAFORM_WORKSPACE_ROOT=/runner-data/jobs
ENV TERRAFORM_PROVIDER_CACHE_DIR=/runner-data/plugin-cache
ENV TERRAFORM_TEMPLATE_ROOT=/app/templates
ENV RUNNER_CONFIG_FILE=/app/config/terraform-runner.yaml

EXPOSE 8091

ENTRYPOINT ["terraform-runner"]
