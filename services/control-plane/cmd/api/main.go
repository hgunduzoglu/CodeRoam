package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
)

const (
	databaseStartupTimeout = 10 * time.Second
	serverShutdownTimeout  = 10 * time.Second
)

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Time    string `json:"time"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Printf("CodeRoam control plane stopped: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	config, err := loadAPIConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("load control-plane configuration: %w", err)
	}
	startupCtx, cancelStartup := context.WithTimeout(ctx, databaseStartupTimeout)
	pool, err := postgresx.OpenPool(startupCtx, config.postgresDSN)
	cancelStartup()
	if err != nil {
		return fmt.Errorf("start control-plane database: %w", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{
			Service: "coderoam-control-plane",
			Status:  "ok",
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	})
	server := &http.Server{
		Addr:              config.httpAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("CodeRoam control plane listening on %s", config.httpAddress)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve control-plane HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), serverShutdownTimeout)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			closeErr := server.Close()
			return errors.Join(fmt.Errorf("shutdown control-plane HTTP: %w", err), closeErr)
		}
		select {
		case err := <-serverErrors:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("stop control-plane HTTP: %w", err)
			}
			return nil
		case <-shutdownCtx.Done():
			closeErr := server.Close()
			return errors.Join(fmt.Errorf("wait for control-plane HTTP shutdown: %w", shutdownCtx.Err()), closeErr)
		}
	}
}
