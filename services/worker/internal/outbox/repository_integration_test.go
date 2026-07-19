package outbox

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type workerOutboxFixture struct {
	id            string
	eventType     string
	aggregateType string
	aggregateID   string
	payload       string
	availableAt   time.Time
	attemptCount  int
}

func TestRepositoryClaimFinishIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)
	applyWorkerOutboxMigration(t, ctx, pool)
	repository := NewRepository()
	now := time.Date(2026, time.July, 19, 19, 0, 0, 0, time.UTC)

	emptyTx := beginWorkerOutboxTx(t, ctx, pool, "empty")
	if claim, ok, err := repository.Claim(ctx, emptyTx, now); err != nil || ok || claim != (claimedEvent{}) {
		t.Fatalf("Claim(empty) = (%v, %t, %v), want zero, false, nil", claim, ok, err)
	}
	rollbackWorkerOutboxTx(t, emptyTx, "empty")

	future := newWorkerOutboxFixture(t, now.Add(time.Hour), "device.revoked.v1", "device")
	insertWorkerOutboxFixture(t, ctx, pool, future)
	futureTx := beginWorkerOutboxTx(t, ctx, pool, "future")
	if _, ok, err := repository.Claim(ctx, futureTx, now); err != nil || ok {
		t.Fatalf("Claim(future) = (_, %t, %v), want false, nil", ok, err)
	}
	rollbackWorkerOutboxTx(t, futureTx, "future")

	first := newWorkerOutboxFixture(t, now.Add(-2*time.Minute), "device.revoked.v1", "device")
	second := newWorkerOutboxFixture(t, now.Add(-time.Minute), "agent.revoked.v1", "agent")
	insertWorkerOutboxFixture(t, ctx, pool, first)
	insertWorkerOutboxFixture(t, ctx, pool, second)
	firstTx := beginWorkerOutboxTx(t, ctx, pool, "first")
	firstClaim, ok, err := repository.Claim(ctx, firstTx, now)
	if err != nil || !ok {
		t.Fatalf("Claim(first) = (_, %t, %v), want true, nil", ok, err)
	}
	assertWorkerOutboxClaim(t, firstClaim, first, EventDeviceRevoked)
	secondTx := beginWorkerOutboxTx(t, ctx, pool, "second")
	secondClaim, ok, err := repository.Claim(ctx, secondTx, now)
	if err != nil || !ok {
		t.Fatalf("Claim(skip locked) = (_, %t, %v), want true, nil", ok, err)
	}
	assertWorkerOutboxClaim(t, secondClaim, second, EventAgentRevoked)
	if err := repository.Finish(ctx, secondTx, secondClaim, finishRetry, now); err != nil {
		t.Fatalf("Finish(retry) error = %v", err)
	}
	if err := secondTx.Commit(ctx); err != nil {
		t.Fatalf("commit retry: %v", err)
	}
	if err := repository.Finish(ctx, firstTx, firstClaim, finishCompleted, now); err != nil {
		t.Fatalf("Finish(completed) error = %v", err)
	}
	if err := firstTx.Commit(ctx); err != nil {
		t.Fatalf("commit completion: %v", err)
	}
	assertWorkerOutboxState(t, ctx, pool, first.id, &now, now.Add(-2*time.Minute), 1, nil)
	retryError := "retryable handler failure"
	assertWorkerOutboxState(
		t, ctx, pool, second.id, nil, now.Add(repositoryRetryDelay), 1, &retryError,
	)

	beforeRetryTx := beginWorkerOutboxTx(t, ctx, pool, "before retry")
	if _, ok, err := repository.Claim(ctx, beforeRetryTx, now); err != nil || ok {
		t.Fatalf("Claim(before retry) = (_, %t, %v), want false, nil", ok, err)
	}
	rollbackWorkerOutboxTx(t, beforeRetryTx, "before retry")
	retryTx := beginWorkerOutboxTx(t, ctx, pool, "retry")
	retryClaim, ok, err := repository.Claim(ctx, retryTx, now.Add(2*repositoryRetryDelay))
	if err != nil || !ok {
		t.Fatalf("Claim(retry) = (_, %t, %v), want true, nil", ok, err)
	}
	retryFixture := second
	retryFixture.availableAt = now.Add(repositoryRetryDelay)
	assertWorkerOutboxClaim(t, retryClaim, retryFixture, EventAgentRevoked)
	if retryClaim.event.attemptCount != 1 {
		t.Fatalf("retry attempt count = %d, want 1", retryClaim.event.attemptCount)
	}
	if err := repository.Finish(ctx, retryTx, retryClaim, finishCompleted, now.Add(2*repositoryRetryDelay)); err != nil {
		t.Fatalf("Finish(retry completion) error = %v", err)
	}
	if err := retryTx.Commit(ctx); err != nil {
		t.Fatalf("commit retry completion: %v", err)
	}

	invalid := newWorkerOutboxFixture(t, now.Add(-time.Minute), "device.revoked.v1", "device")
	invalid.payload = `{"forged":"payload"}`
	insertWorkerOutboxFixture(t, ctx, pool, invalid)
	invalidTx := beginWorkerOutboxTx(t, ctx, pool, "invalid")
	invalidClaim, ok, err := repository.Claim(ctx, invalidTx, now)
	if err != nil || !ok || !invalidClaim.invalid {
		t.Fatalf("Claim(invalid) = (%v, %t, %v), want invalid, true, nil", invalidClaim, ok, err)
	}
	if err := repository.Finish(ctx, invalidTx, invalidClaim, finishDiscardInvalid, now); err != nil {
		t.Fatalf("Finish(invalid) error = %v", err)
	}
	if err := invalidTx.Commit(ctx); err != nil {
		t.Fatalf("commit invalid discard: %v", err)
	}
	invalidError := "invalid event metadata"
	assertWorkerOutboxState(t, ctx, pool, invalid.id, &now, invalid.availableAt, 1, &invalidError)

	rolledBack := newWorkerOutboxFixture(t, now.Add(-time.Minute), "device.revoked.v1", "device")
	insertWorkerOutboxFixture(t, ctx, pool, rolledBack)
	rollbackTx := beginWorkerOutboxTx(t, ctx, pool, "rollback")
	rollbackClaim, ok, err := repository.Claim(ctx, rollbackTx, now)
	if err != nil || !ok {
		t.Fatalf("Claim(rollback) = (_, %t, %v), want true, nil", ok, err)
	}
	if err := repository.Finish(ctx, rollbackTx, rollbackClaim, finishCompleted, now); err != nil {
		t.Fatalf("Finish(rollback) error = %v", err)
	}
	rollbackWorkerOutboxTx(t, rollbackTx, "rollback")
	assertWorkerOutboxState(t, ctx, pool, rolledBack.id, nil, rolledBack.availableAt, 0, nil)
	recoverTx := beginWorkerOutboxTx(t, ctx, pool, "recover")
	recoveredClaim, ok, err := repository.Claim(ctx, recoverTx, now)
	if err != nil || !ok {
		t.Fatalf("Claim(after rollback) = (_, %t, %v), want true, nil", ok, err)
	}
	if err := repository.Finish(ctx, recoverTx, recoveredClaim, finishCompleted, now); err != nil {
		t.Fatalf("Finish(after rollback) error = %v", err)
	}
	if err := recoverTx.Commit(ctx); err != nil {
		t.Fatalf("commit after rollback: %v", err)
	}

	oversized := newWorkerOutboxFixture(t, now.Add(-time.Minute), "device.revoked.v1", "device")
	oversized.id = strings.Repeat("a", 4096)
	insertWorkerOutboxFixture(t, ctx, pool, oversized)
	oversizedTx := beginWorkerOutboxTx(t, ctx, pool, "oversized")
	oversizedClaim, ok, err := repository.Claim(ctx, oversizedTx, now)
	if err != nil || !ok || !oversizedClaim.invalid {
		t.Fatalf("Claim(oversized) = (_, %t, %v), want invalid, true, nil", ok, err)
	}
	if err := repository.Finish(ctx, oversizedTx, oversizedClaim, finishDiscardInvalid, now); err != nil {
		t.Fatalf("Finish(oversized) error = %v", err)
	}
	if err := oversizedTx.Commit(ctx); err != nil {
		t.Fatalf("commit oversized discard: %v", err)
	}
	assertWorkerOutboxState(t, ctx, pool, oversized.id, &now, oversized.availableAt, 1, &invalidError)
}

func applyWorkerOutboxMigration(t *testing.T, ctx context.Context, database postgresx.TransactionStarter) {
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

func newWorkerOutboxFixture(
	t *testing.T,
	availableAt time.Time,
	eventType string,
	aggregateType string,
) workerOutboxFixture {
	t.Helper()
	return workerOutboxFixture{
		id:            newWorkerOutboxIntegrationID(t),
		eventType:     eventType,
		aggregateType: aggregateType,
		aggregateID:   newWorkerOutboxIntegrationID(t),
		payload:       `{}`,
		availableAt:   availableAt,
	}
}

func insertWorkerOutboxFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture workerOutboxFixture,
) {
	t.Helper()
	var storedID string
	if err := pool.QueryRow(ctx, `SELECT id FROM outbox.events WHERE id = $1`, fixture.id).Scan(&storedID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("outbox fixture %q lookup error = %v, want pgx.ErrNoRows", fixture.id, err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM outbox.events WHERE id = $1`, fixture.id); err != nil {
			t.Errorf("delete outbox fixture: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO outbox.events (
			id, event_type, aggregate_type, aggregate_id, payload, available_at, attempt_count
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)`,
		fixture.id, fixture.eventType, fixture.aggregateType, fixture.aggregateID,
		fixture.payload, fixture.availableAt, fixture.attemptCount,
	); err != nil {
		t.Fatalf("insert outbox fixture: %v", err)
	}
}

func beginWorkerOutboxTx(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin %s worker outbox transaction: %v", name, err)
	}
	t.Cleanup(func() { rollbackWorkerOutboxTx(t, tx, name) })
	return tx
}

func rollbackWorkerOutboxTx(t *testing.T, tx pgx.Tx, name string) {
	t.Helper()
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()
	if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) && !tx.Conn().IsClosed() {
		t.Fatalf("rollback %s worker outbox transaction: %v", name, err)
	}
}

func assertWorkerOutboxClaim(
	t *testing.T,
	claim claimedEvent,
	want workerOutboxFixture,
	wantKind EventKind,
) {
	t.Helper()
	if claim.invalid || claim.locator == "" || claim.event.id.String() != want.id ||
		claim.event.kind != wantKind || claim.event.aggregateID.String() != want.aggregateID ||
		!claim.event.availableAt.Equal(want.availableAt) {
		t.Fatalf("claimed event = %+v, want fixture %s", claim, want.id)
	}
}

func assertWorkerOutboxState(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	eventID string,
	wantProcessedAt *time.Time,
	wantAvailableAt time.Time,
	wantAttempts int,
	wantLastError *string,
) {
	t.Helper()
	var processedAt *time.Time
	var availableAt time.Time
	var attemptCount int
	var lastError *string
	if err := pool.QueryRow(ctx, `
		SELECT processed_at, available_at, attempt_count, last_error
		FROM outbox.events WHERE id = $1`, eventID,
	).Scan(&processedAt, &availableAt, &attemptCount, &lastError); err != nil {
		t.Fatalf("read outbox event state: %v", err)
	}
	if !optionalWorkerOutboxTimeEqual(processedAt, wantProcessedAt) ||
		!availableAt.Equal(wantAvailableAt) || attemptCount != wantAttempts ||
		!optionalWorkerOutboxStringEqual(lastError, wantLastError) {
		t.Fatalf(
			"outbox state = processed %v, available %v, attempts %d, error %v",
			processedAt, availableAt, attemptCount, lastError,
		)
	}
}

func optionalWorkerOutboxTimeEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func optionalWorkerOutboxStringEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func newWorkerOutboxIntegrationID(t *testing.T) string {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New() error = %v", err)
	}
	return id.String()
}
