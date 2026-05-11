package main

import (
	"context"
	"log"

	"cluster-addon-worker/internal/config"
	"cluster-addon-worker/internal/runtime"
)

func main() {
	cfg := config.Load()
	if err := runtime.New(cfg).Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
