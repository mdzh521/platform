package main

import (
	"context"
	"log"

	"terraform-runner/internal/config"
	"terraform-runner/internal/runtime"
)

func main() {
	cfg := config.Load()
	if err := runtime.New(cfg).Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
