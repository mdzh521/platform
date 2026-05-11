package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"resource-sync-worker/internal/config"
	"resource-sync-worker/internal/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runtime.New(config.Load()).Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
