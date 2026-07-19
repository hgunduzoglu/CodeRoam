package outbox

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type failingFinishStore struct {
	repository *Repository
	fail       bool
}

func (store *failingFinishStore) Claim(
	ctx context.Context,
	tx pgx.Tx,
	availableAt time.Time,
) (claimedEvent, bool, error) {
	return store.repository.Claim(ctx, tx, availableAt)
}

func (store *failingFinishStore) Finish(
	ctx context.Context,
	tx pgx.Tx,
	claim claimedEvent,
	outcome finishOutcome,
	finishedAt time.Time,
) error {
	if store.fail {
		store.fail = false
		return errors.New("simulated finish failure")
	}
	return store.repository.Finish(ctx, tx, claim, outcome, finishedAt)
}

type ambiguousWorkerCommitStarter struct {
	pool *pgxpool.Pool
}

func (starter *ambiguousWorkerCommitStarter) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := starter.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &ambiguousWorkerCommitTx{Tx: tx}, nil
}

type ambiguousWorkerCommitTx struct {
	pgx.Tx
}

func (tx *ambiguousWorkerCommitTx) Commit(ctx context.Context) error {
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	return errors.New("simulated lost worker commit acknowledgement")
}

func TestProcessorIntegration(t *testing.T) {
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
	current := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	clock := func() time.Time { return current }

	poisonFixture := newWorkerOutboxFixture(t, current.Add(-2*time.Minute), "device.revoked.v1", "device")
	insertNonFiniteWorkerOutboxFixture(t, ctx, pool, poisonFixture, "-infinity")
	followingFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "agent.revoked.v1", "agent")
	insertWorkerOutboxFixture(t, ctx, pool, followingFixture)
	positivePoisonFixture := newWorkerOutboxFixture(t, current.Add(time.Hour), "device.revoked.v1", "device")
	insertNonFiniteWorkerOutboxFixture(t, ctx, pool, positivePoisonFixture, "infinity")
	followingCalls := 0
	poisonProcessor := newProcessorIntegration(t, pool, repository, HandlerFunc(
		func(_ context.Context, kind EventKind, aggregateID string) error {
			followingCalls++
			if kind != EventAgentRevoked || aggregateID != followingFixture.aggregateID {
				t.Fatal("handler received poison or unexpected event metadata")
			}
			return nil
		},
	), clock)
	result, err := poisonProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessDiscarded || followingCalls != 0 {
		t.Fatalf("ProcessNext(poison) = (%v, %v), calls %d", result, err, followingCalls)
	}
	invalidError := "invalid event metadata"
	assertWorkerOutboxTerminalState(t, ctx, pool, poisonFixture.id, &current, 1, &invalidError)
	result, err = poisonProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessCompleted || followingCalls != 1 {
		t.Fatalf("ProcessNext(after poison) = (%v, %v), calls %d", result, err, followingCalls)
	}
	assertWorkerOutboxState(t, ctx, pool, followingFixture.id, &current, followingFixture.availableAt, 1, nil)
	result, err = poisonProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessDiscarded || followingCalls != 1 {
		t.Fatalf("ProcessNext(positive-infinity poison) = (%v, %v), calls %d", result, err, followingCalls)
	}
	assertWorkerOutboxTerminalState(t, ctx, pool, positivePoisonFixture.id, &current, 1, &invalidError)

	completedFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "device.revoked.v1", "device")
	insertWorkerOutboxFixture(t, ctx, pool, completedFixture)
	completedCalls := 0
	completedHandler := HandlerFunc(func(_ context.Context, kind EventKind, aggregateID string) error {
		completedCalls++
		if kind != EventDeviceRevoked || aggregateID != completedFixture.aggregateID {
			t.Fatal("completed handler received unexpected metadata")
		}
		return nil
	})
	processor := newProcessorIntegration(t, pool, repository, completedHandler, clock)
	result, err = processor.ProcessNext(ctx)
	if err != nil || result != ProcessCompleted || completedCalls != 1 {
		t.Fatalf("ProcessNext(completed) = (%v, %v), calls %d", result, err, completedCalls)
	}
	assertWorkerOutboxState(t, ctx, pool, completedFixture.id, &current, completedFixture.availableAt, 1, nil)

	retryFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "agent.revoked.v1", "agent")
	insertWorkerOutboxFixture(t, ctx, pool, retryFixture)
	retryCalls := 0
	retryHandler := HandlerFunc(func(_ context.Context, kind EventKind, aggregateID string) error {
		retryCalls++
		if kind != EventAgentRevoked || aggregateID != retryFixture.aggregateID {
			t.Fatal("retry handler received unexpected metadata")
		}
		if retryCalls == 1 {
			return errors.New("temporary downstream failure")
		}
		return nil
	})
	retryProcessor := newProcessorIntegration(t, pool, repository, retryHandler, clock)
	result, err = retryProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessRetryScheduled || retryCalls != 1 {
		t.Fatalf("ProcessNext(retry) = (%v, %v), calls %d", result, err, retryCalls)
	}
	retryError := "retryable handler failure"
	assertWorkerOutboxState(
		t, ctx, pool, retryFixture.id, nil, current.Add(repositoryRetryDelay), 1, &retryError,
	)
	current = current.Add(2 * repositoryRetryDelay)
	result, err = retryProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessCompleted || retryCalls != 2 {
		t.Fatalf("ProcessNext(retry success) = (%v, %v), calls %d", result, err, retryCalls)
	}
	assertWorkerOutboxState(t, ctx, pool, retryFixture.id, &current, current.Add(-repositoryRetryDelay), 2, nil)

	permanentFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "device.revoked.v1", "device")
	insertWorkerOutboxFixture(t, ctx, pool, permanentFixture)
	permanentProcessor := newProcessorIntegration(t, pool, repository, HandlerFunc(
		func(context.Context, EventKind, string) error { return ErrPermanentHandler },
	), clock)
	result, err = permanentProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessDiscarded {
		t.Fatalf("ProcessNext(permanent) = (%v, %v), want discarded, nil", result, err)
	}
	permanentError := "permanent handler failure"
	assertWorkerOutboxState(
		t, ctx, pool, permanentFixture.id, &current, permanentFixture.availableAt, 1, &permanentError,
	)

	exhaustedFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "device.revoked.v1", "device")
	exhaustedFixture.attemptCount = maxDeliveryAttempts - 1
	insertWorkerOutboxFixture(t, ctx, pool, exhaustedFixture)
	exhaustedProcessor := newProcessorIntegration(t, pool, repository, HandlerFunc(
		func(context.Context, EventKind, string) error { return errors.New("still unavailable") },
	), clock)
	result, err = exhaustedProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessDiscarded {
		t.Fatalf("ProcessNext(exhausted) = (%v, %v), want discarded, nil", result, err)
	}
	exhaustedError := "delivery attempts exhausted"
	assertWorkerOutboxState(
		t, ctx, pool, exhaustedFixture.id, &current,
		exhaustedFixture.availableAt, maxDeliveryAttempts, &exhaustedError,
	)

	crashFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "device.revoked.v1", "device")
	insertWorkerOutboxFixture(t, ctx, pool, crashFixture)
	crashCalls := 0
	crashHandler := HandlerFunc(func(context.Context, EventKind, string) error {
		crashCalls++
		return nil
	})
	failingStore := &failingFinishStore{repository: repository, fail: true}
	crashProcessor := newProcessorIntegration(t, pool, failingStore, crashHandler, clock)
	if result, err := crashProcessor.ProcessNext(ctx); result != ProcessNoEvent ||
		!errors.Is(err, ErrProcessingUnavailable) {
		t.Fatalf("ProcessNext(crash after side effect) = (%v, %v), want unavailable", result, err)
	}
	assertWorkerOutboxState(t, ctx, pool, crashFixture.id, nil, crashFixture.availableAt, 0, nil)
	recoveryProcessor := newProcessorIntegration(t, pool, repository, crashHandler, clock)
	result, err = recoveryProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessCompleted || crashCalls != 2 {
		t.Fatalf("ProcessNext(redelivery) = (%v, %v), calls %d", result, err, crashCalls)
	}
	assertWorkerOutboxState(t, ctx, pool, crashFixture.id, &current, crashFixture.availableAt, 1, nil)

	ambiguousFixture := newWorkerOutboxFixture(t, current.Add(-time.Minute), "agent.revoked.v1", "agent")
	insertWorkerOutboxFixture(t, ctx, pool, ambiguousFixture)
	ambiguousCalls := 0
	ambiguousHandler := HandlerFunc(func(context.Context, EventKind, string) error {
		ambiguousCalls++
		return nil
	})
	ambiguousProcessor := newProcessorIntegration(
		t, &ambiguousWorkerCommitStarter{pool: pool}, repository, ambiguousHandler, clock,
	)
	if result, err := ambiguousProcessor.ProcessNext(ctx); result != ProcessNoEvent ||
		!errors.Is(err, ErrProcessOutcomeUnknown) {
		t.Fatalf("ProcessNext(ambiguous commit) = (%v, %v), want outcome unknown", result, err)
	}
	assertWorkerOutboxState(t, ctx, pool, ambiguousFixture.id, &current, ambiguousFixture.availableAt, 1, nil)
	reconcileProcessor := newProcessorIntegration(t, pool, repository, ambiguousHandler, clock)
	result, err = reconcileProcessor.ProcessNext(ctx)
	if err != nil || result != ProcessNoEvent || ambiguousCalls != 1 {
		t.Fatalf("ProcessNext(after ambiguous commit) = (%v, %v), calls %d", result, err, ambiguousCalls)
	}
}

func newProcessorIntegration(
	t *testing.T,
	transactions transactionStarter,
	store eventStore,
	handler Handler,
	now func() time.Time,
) *Processor {
	t.Helper()
	processor, err := NewProcessor(transactions, store, handler, now)
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	return processor
}
