FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /backend-center ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates aws-cli kubectl

WORKDIR /app
COPY --from=builder /backend-center /usr/local/bin/backend-center
COPY --from=builder /app/config ./config
COPY --from=builder /app/web ./web

EXPOSE 8080
CMD ["backend-center"]
