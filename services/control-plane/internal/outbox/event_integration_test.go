package outbox

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

func TestEnqueueIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	migrationSQL, err := os.ReadFile("migrations/000001_init.sql")
	if err != nil {
		t.Fatalf("read outbox migration: %v", err)
	}
	if err := postgresx.ApplyMigrations(ctx, pool, []postgresx.Migration{{
		Scope: "outbox", Version: 1, Name: "init", SQL: string(migrationSQL),
	}}); err != nil {
		t.Fatalf("apply outbox migration: %v", err)
	}

	availableAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	rolledBackEvent := newIntegrationEvent(t, availableAt)
	rollbackTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}
	cleanupTransaction(t, rollbackTx)
	if err := Enqueue(ctx, rollbackTx, rolledBackEvent); err != nil {
		t.Fatalf("Enqueue(rollback) error = %v", err)
	}
	if err := rollbackTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback transaction: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox.events WHERE id = $1`, rolledBackEvent.id.String()).Scan(&count); err != nil {
		t.Fatalf("count rolled-back event: %v", err)
	}
	if count != 0 {
		t.Fatalf("rolled-back event count = %d, want 0", count)
	}

	committedEvent := newIntegrationEvent(t, availableAt)
	commitTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin commit transaction: %v", err)
	}
	cleanupTransaction(t, commitTx)
	if err := Enqueue(ctx, commitTx, committedEvent); err != nil {
		t.Fatalf("Enqueue(commit) error = %v", err)
	}
	if err := commitTx.Commit(ctx); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM outbox.events WHERE id = $1`, committedEvent.id.String()); err != nil {
			t.Errorf("delete committed test event: %v", err)
		}
	})

	var eventType, aggregateType, aggregateID, payload string
	var storedAvailableAt time.Time
	var processedAt *time.Time
	var attemptCount int
	var lastError *string
	if err := pool.QueryRow(ctx, `
		SELECT event_type, aggregate_type, aggregate_id, payload::text, available_at,
		       processed_at, attempt_count, last_error
		FROM outbox.events
		WHERE id = $1`, committedEvent.id.String()).Scan(
		&eventType,
		&aggregateType,
		&aggregateID,
		&payload,
		&storedAvailableAt,
		&processedAt,
		&attemptCount,
		&lastError,
	); err != nil {
		t.Fatalf("read committed event: %v", err)
	}
	if eventType != committedEvent.eventType || aggregateType != committedEvent.aggregateType ||
		aggregateID != committedEvent.aggregateID.String() || payload != "{}" ||
		!storedAvailableAt.Equal(committedEvent.availableAt) || processedAt != nil ||
		attemptCount != 0 || lastError != nil {
		t.Fatal("committed outbox event did not preserve metadata-only identity")
	}

	duplicateTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin duplicate transaction: %v", err)
	}
	cleanupTransaction(t, duplicateTx)
	duplicateErr := Enqueue(ctx, duplicateTx, committedEvent)
	if err := duplicateTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback duplicate transaction: %v", err)
	}
	if !errors.Is(duplicateErr, ErrEventAlreadyExists) {
		t.Fatalf("Enqueue(duplicate) error = %v, want ErrEventAlreadyExists", duplicateErr)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox.events WHERE id = $1`, committedEvent.id.String()).Scan(&count); err != nil {
		t.Fatalf("count event after duplicate: %v", err)
	}
	if count != 1 {
		t.Fatalf("event count after duplicate = %d, want 1", count)
	}
}

func cleanupTransaction(t *testing.T, tx pgx.Tx) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = tx.Rollback(cleanupCtx)
	})
}

func newIntegrationEvent(t *testing.T, availableAt time.Time) Event {
	t.Helper()
	aggregateID, err := ids.Parse("1123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ids.Parse() error = %v", err)
	}
	event, err := NewEvent(EventDeviceRevoked, aggregateID, availableAt)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	return event
}
