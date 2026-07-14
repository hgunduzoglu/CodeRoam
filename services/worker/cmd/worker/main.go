package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Print("CodeRoam worker started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Print("CodeRoam worker stopped")
			return
		case <-ticker.C:
			log.Print("worker heartbeat")
		}
	}
}
