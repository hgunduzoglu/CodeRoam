package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/hgunduzoglu/coderoam/services/worker/internal/outbox"
	"github.com/jackc/pgx/v5"
)

const (
	workerDatabaseStartupTimeout = 10 * time.Second
	workerPollDelay              = time.Second
)

var ErrWorkerUnavailable = errors.New("worker unavailable")

type workerPool interface {
	Begin(context.Context) (pgx.Tx, error)
	Close()
}

type workerPoolOpener func(context.Context, string) (workerPool, error)

type nextProcessor interface {
	ProcessNext(context.Context) (outbox.ProcessResult, error)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	openPool := func(ctx context.Context, dsn string) (workerPool, error) {
		return postgresx.OpenPool(ctx, dsn)
	}
	if err := runWorker(ctx, os.Getenv, openPool); err != nil {
		log.Print("CodeRoam worker stopped unexpectedly")
		os.Exit(1)
	}
	log.Print("CodeRoam worker stopped")
}

func runWorker(ctx context.Context, getenv func(string) string, openPool workerPoolOpener) error {
	if ctx == nil || openPool == nil {
		return ErrWorkerUnavailable
	}
	config, err := loadWorkerConfig(getenv)
	if err != nil {
		return fmt.Errorf("%w: configuration: %w", ErrWorkerUnavailable, err)
	}
	startupCtx, cancelStartup := context.WithTimeout(ctx, workerDatabaseStartupTimeout)
	pool, err := openPool(startupCtx, config.postgresDSN)
	cancelStartup()
	if err != nil || pool == nil {
		return fmt.Errorf("%w: database startup", ErrWorkerUnavailable)
	}
	defer pool.Close()

	log.Print("CodeRoam worker started")
	if !config.processingEnabled {
		log.Print("worker outbox processing disabled")
		<-ctx.Done()
		return nil
	}
	processor, err := outbox.NewProcessor(
		pool,
		outbox.NewRepository(),
		outbox.HandlerFunc(handleM2Revocation),
		time.Now,
	)
	if err != nil {
		return fmt.Errorf("%w: processor startup", ErrWorkerUnavailable)
	}
	return runProcessingLoop(ctx, processor)
}

func runProcessingLoop(ctx context.Context, processor nextProcessor) error {
	if ctx == nil || processor == nil {
		return ErrWorkerUnavailable
	}
	for {
		result, err := processor.ProcessNext(ctx)
		if ctx.Err() != nil {
			return nil
		}
		wait := err != nil || result == outbox.ProcessNoEvent
		if err != nil {
			log.Print("worker outbox delivery failed; retrying")
		}
		if result > outbox.ProcessDiscarded {
			log.Print("worker outbox processor returned an invalid result; retrying")
			wait = true
		}
		if !wait {
			continue
		}
		timer := time.NewTimer(workerPollDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func handleM2Revocation(ctx context.Context, kind outbox.EventKind, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch kind {
	case outbox.EventDeviceRevoked, outbox.EventAgentRevoked:
		// M2 has no relay sessions to terminate. M4 must replace this acknowledgement
		// before it enables relay sessions.
		return nil
	default:
		return outbox.ErrPermanentHandler
	}
}
