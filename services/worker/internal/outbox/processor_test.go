package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type processorTxStub struct {
	pgx.Tx
	commitErr     error
	rollbackErr   error
	commitCalls   int
	rollbackCalls int
}

func (tx *processorTxStub) Commit(context.Context) error {
	tx.commitCalls++
	return tx.commitErr
}

func (tx *processorTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	if tx.commitCalls > 0 && tx.commitErr == nil {
		return pgx.ErrTxClosed
	}
	return tx.rollbackErr
}

type processorStarterStub struct {
	tx    pgx.Tx
	err   error
	calls int
}

func (starter *processorStarterStub) Begin(context.Context) (pgx.Tx, error) {
	starter.calls++
	return starter.tx, starter.err
}

type processorStoreStub struct {
	claim       claimedEvent
	claimOK     bool
	claimErr    error
	finishErr   error
	claimCalls  int
	finishCalls int
	claimTx     pgx.Tx
	finishTx    pgx.Tx
	outcome     finishOutcome
}

func (store *processorStoreStub) Claim(
	_ context.Context,
	tx pgx.Tx,
	_ time.Time,
) (claimedEvent, bool, error) {
	store.claimCalls++
	store.claimTx = tx
	return store.claim, store.claimOK, store.claimErr
}

func (store *processorStoreStub) Finish(
	_ context.Context,
	tx pgx.Tx,
	_ claimedEvent,
	outcome finishOutcome,
	_ time.Time,
) error {
	store.finishCalls++
	store.finishTx = tx
	store.outcome = outcome
	return store.finishErr
}

type processorHandlerStub struct {
	err         error
	calls       int
	kind        EventKind
	aggregateID string
}

func (handler *processorHandlerStub) Handle(
	_ context.Context,
	kind EventKind,
	aggregateID string,
) error {
	handler.calls++
	handler.kind = kind
	handler.aggregateID = aggregateID
	return handler.err
}

func TestNewProcessorRequiresDependencies(t *testing.T) {
	starter := &processorStarterStub{}
	store := &processorStoreStub{}
	handler := &processorHandlerStub{}
	now := time.Now
	tests := map[string]struct {
		transactions transactionStarter
		store        eventStore
		handler      Handler
		now          func() time.Time
	}{
		"missing transactions": {store: store, handler: handler, now: now},
		"missing store":        {transactions: starter, handler: handler, now: now},
		"missing handler":      {transactions: starter, store: store, now: now},
		"missing clock":        {transactions: starter, store: store, handler: handler},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewProcessor(test.transactions, test.store, test.handler, test.now); err == nil {
				t.Fatal("NewProcessor() error = nil")
			}
		})
	}
}

func TestProcessorProcessNextOutcomes(t *testing.T) {
	now := time.Date(2026, time.July, 19, 19, 30, 0, 0, time.UTC)
	validClaim := newProcessorTestClaim(t, 0)
	tests := map[string]struct {
		claim       claimedEvent
		claimOK     bool
		claimErr    error
		handlerErr  error
		commitErr   error
		wantResult  ProcessResult
		wantError   error
		wantOutcome finishOutcome
		handlerCall int
		finishCall  int
		commitCall  int
	}{
		"no event": {wantResult: ProcessNoEvent},
		"completed": {
			claim: validClaim, claimOK: true, wantResult: ProcessCompleted,
			wantOutcome: finishCompleted, handlerCall: 1, finishCall: 1, commitCall: 1,
		},
		"invalid discarded": {
			claim: claimedEvent{locator: "(0,2)", invalid: true}, claimOK: true,
			wantResult: ProcessDiscarded, wantOutcome: finishDiscardInvalid, finishCall: 1, commitCall: 1,
		},
		"retry scheduled": {
			claim: validClaim, claimOK: true, handlerErr: errors.New("temporary failure"),
			wantResult: ProcessRetryScheduled, wantOutcome: finishRetry,
			handlerCall: 1, finishCall: 1, commitCall: 1,
		},
		"permanent discarded": {
			claim: validClaim, claimOK: true, handlerErr: ErrPermanentHandler,
			wantResult: ProcessDiscarded, wantOutcome: finishDiscardPermanent,
			handlerCall: 1, finishCall: 1, commitCall: 1,
		},
		"attempts exhausted before handler": {
			claim: newProcessorTestClaim(t, maxDeliveryAttempts), claimOK: true,
			wantResult: ProcessDiscarded, wantOutcome: finishDiscardExhausted,
			finishCall: 1, commitCall: 1,
		},
		"last failed attempt discarded": {
			claim: newProcessorTestClaim(t, maxDeliveryAttempts-1), claimOK: true,
			handlerErr: errors.New("temporary failure"), wantResult: ProcessDiscarded,
			wantOutcome: finishDiscardExhausted, handlerCall: 1, finishCall: 1, commitCall: 1,
		},
		"claim failure": {
			claimErr: errors.New("database unavailable"), wantResult: ProcessNoEvent,
			wantError: ErrProcessingUnavailable,
		},
		"commit outcome unknown": {
			claim: validClaim, claimOK: true, commitErr: errors.New("connection lost"),
			wantResult: ProcessNoEvent, wantError: ErrProcessOutcomeUnknown,
			wantOutcome: finishCompleted, handlerCall: 1, finishCall: 1, commitCall: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &processorTxStub{commitErr: test.commitErr}
			starter := &processorStarterStub{tx: tx}
			store := &processorStoreStub{claim: test.claim, claimOK: test.claimOK, claimErr: test.claimErr}
			handler := &processorHandlerStub{err: test.handlerErr}
			processor := newProcessorForTest(t, starter, store, handler, func() time.Time { return now })

			result, err := processor.ProcessNext(context.Background())
			if result != test.wantResult || !errors.Is(err, test.wantError) {
				t.Fatalf("ProcessNext() = (%v, %v), want (%v, %v)", result, err, test.wantResult, test.wantError)
			}
			if handler.calls != test.handlerCall || store.finishCalls != test.finishCall ||
				tx.commitCalls != test.commitCall || tx.rollbackCalls != 1 {
				t.Fatalf(
					"calls = handler %d, finish %d, commit %d, rollback %d",
					handler.calls, store.finishCalls, tx.commitCalls, tx.rollbackCalls,
				)
			}
			if store.finishCalls > 0 && (store.outcome != test.wantOutcome || store.claimTx != tx || store.finishTx != tx) {
				t.Fatal("ProcessNext() did not finish the claim with the expected outcome in one transaction")
			}
			if handler.calls > 0 && (handler.kind != EventDeviceRevoked || handler.aggregateID != "1123456789abcdef0123456789abcdef") {
				t.Fatal("ProcessNext() passed unexpected bounded metadata to handler")
			}
		})
	}
}

func TestProcessorProcessNextBoundsHandler(t *testing.T) {
	tx := &processorTxStub{}
	starter := &processorStarterStub{tx: tx}
	store := &processorStoreStub{claim: newProcessorTestClaim(t, 0), claimOK: true}
	handler := HandlerFunc(func(ctx context.Context, _ EventKind, _ string) error {
		<-ctx.Done()
		return nil
	})
	processor := newProcessorForTest(t, starter, store, handler, time.Now)
	processor.handlerMax = time.Millisecond
	processor.operationMax = time.Second

	result, err := processor.ProcessNext(context.Background())
	if err != nil || result != ProcessRetryScheduled {
		t.Fatalf("ProcessNext(timeout) = (%v, %v), want retry, nil", result, err)
	}
	if store.outcome != finishRetry || tx.commitCalls != 1 {
		t.Fatal("handler timeout was not durably scheduled for retry")
	}
}

func TestProcessorProcessNextRejectsInvalidBoundaries(t *testing.T) {
	tx := &processorTxStub{}
	starter := &processorStarterStub{tx: tx}
	store := &processorStoreStub{}
	handler := &processorHandlerStub{}
	processor := newProcessorForTest(t, starter, store, handler, time.Now)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	var nilProcessor *Processor
	if _, err := nilProcessor.ProcessNext(context.Background()); !errors.Is(err, ErrProcessingUnavailable) {
		t.Fatalf("nil Processor ProcessNext() error = %v, want ErrProcessingUnavailable", err)
	}
	if _, err := processor.ProcessNext(nil); !errors.Is(err, ErrProcessingUnavailable) {
		t.Fatalf("ProcessNext(nil context) error = %v, want ErrProcessingUnavailable", err)
	}
	if _, err := processor.ProcessNext(canceledCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ProcessNext(canceled context) error = %v, want context.Canceled", err)
	}
	processor.handlerMax = processor.operationMax
	if _, err := processor.ProcessNext(context.Background()); !errors.Is(err, ErrProcessingUnavailable) {
		t.Fatalf("ProcessNext(invalid timeouts) error = %v, want ErrProcessingUnavailable", err)
	}
	if starter.calls != 0 {
		t.Fatalf("invalid boundaries began %d transactions, want 0", starter.calls)
	}
}

func newProcessorForTest(
	t *testing.T,
	starter transactionStarter,
	store eventStore,
	handler Handler,
	now func() time.Time,
) *Processor {
	t.Helper()
	processor, err := NewProcessor(starter, store, handler, now)
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	return processor
}

func newProcessorTestClaim(t *testing.T, attemptCount int) claimedEvent {
	t.Helper()
	event, err := parseEvent(
		"0123456789abcdef0123456789abcdef",
		"device.revoked.v1",
		"device",
		"1123456789abcdef0123456789abcdef",
		true,
		time.Date(2026, time.July, 19, 19, 0, 0, 0, time.UTC),
		attemptCount,
	)
	if err != nil {
		t.Fatalf("parseEvent() error = %v", err)
	}
	return claimedEvent{locator: "(0,1)", event: event}
}
