package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"cluster-enrollment-worker/internal/config"
	"cluster-enrollment-worker/internal/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := runtime.New(config.Load()).Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
