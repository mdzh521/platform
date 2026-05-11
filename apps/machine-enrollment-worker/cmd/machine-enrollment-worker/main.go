package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"machine-enrollment-worker/internal/config"
	"machine-enrollment-worker/internal/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := runtime.New(config.Load()).Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
