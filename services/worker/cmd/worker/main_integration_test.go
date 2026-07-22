package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
)

type workerEventQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func TestRunWorkerIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	observer, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool(observer) error = %v", err)
	}
	t.Cleanup(observer.Close)
	applyWorkerRuntimeMigration(t, ctx, observer)

	eventID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New(event) error = %v", err)
	}
	aggregateID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New(aggregate) error = %v", err)
	}
	var storedID string
	if err := observer.QueryRow(ctx, `SELECT id FROM outbox.events WHERE id = $1`, eventID.String()).Scan(&storedID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("worker event lookup error = %v, want pgx.ErrNoRows", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := observer.Exec(cleanupCtx, `DELETE FROM outbox.events WHERE id = $1`, eventID.String()); err != nil {
			t.Errorf("delete worker event: %v", err)
		}
	})
	if _, err := observer.Exec(ctx, `
		INSERT INTO outbox.events (
			id, event_type, aggregate_type, aggregate_id, payload, available_at
		) VALUES ($1, 'device.revoked.v1', 'device', $2, '{}'::jsonb, $3)`,
		eventID.String(), aggregateID.String(), time.Now().UTC(),
	); err != nil {
		t.Fatalf("insert worker event: %v", err)
	}

	workerCtx, cancelWorker := context.WithCancel(ctx)
	workerDone := make(chan error, 1)
	workerExited := make(chan struct{})
	go func() {
		defer close(workerExited)
		workerDone <- runWorker(
			workerCtx,
			func(key string) string {
				if key == "POSTGRES_DSN" {
					return dsn
				}
				return "true"
			},
			func(openCtx context.Context, workerDSN string) (workerPool, error) {
				return postgresx.OpenPool(openCtx, workerDSN)
			},
		)
	}()
	t.Cleanup(func() {
		cancelWorker()
		cleanupTimer := time.NewTimer(5 * time.Second)
		defer cleanupTimer.Stop()
		select {
		case <-workerExited:
		case <-cleanupTimer.C:
			t.Error("worker did not stop during test cleanup")
		}
	})

	processedAt, attemptCount, lastError := waitForWorkerEvent(t, ctx, observer, eventID.String())
	cancelWorker()
	if err := <-workerDone; err != nil {
		t.Fatalf("runWorker() error = %v", err)
	}
	if processedAt == nil || attemptCount != 1 || lastError != nil {
		t.Fatalf(
			"worker event state = processed %v, attempts %d, error %v",
			processedAt, attemptCount, lastError,
		)
	}
}

func applyWorkerRuntimeMigration(t *testing.T, ctx context.Context, database postgresx.TransactionStarter) {
	t.Helper()
	sql, err := os.ReadFile("../../../control-plane/internal/outbox/migrations/000001_init.sql")
	if err != nil {
		t.Fatalf("read outbox migration: %v", err)
	}
	if err := postgresx.ApplyMigrations(ctx, database, []postgresx.Migration{{
		Scope: "outbox", Version: 1, Name: "init", SQL: string(sql),
	}}); err != nil {
		t.Fatalf("apply outbox migration: %v", err)
	}
}

func waitForWorkerEvent(
	t *testing.T,
	ctx context.Context,
	database workerEventQueryer,
	eventID string,
) (*time.Time, int, *string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var processedAt *time.Time
		var attemptCount int
		var lastError *string
		if err := database.QueryRow(ctx, `
			SELECT processed_at, attempt_count, last_error
			FROM outbox.events WHERE id = $1`, eventID,
		).Scan(&processedAt, &attemptCount, &lastError); err != nil {
			t.Fatalf("read worker event: %v", err)
		}
		if processedAt != nil {
			return processedAt, attemptCount, lastError
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait for worker event: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
