package session

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type pairingAttemptCredentialStoreStub struct {
	matched bool
	err     error
	after   func()
	calls   int
	tx      pgx.Tx
	id      string
}

func (stub *pairingAttemptCredentialStoreStub) authenticateOpenPairingAttempt(
	_ context.Context,
	tx pgx.Tx,
	id string,
	_ []byte,
) (PairingAttempt, bool, error) {
	stub.calls++
	stub.tx = tx
	stub.id = id
	if stub.after != nil {
		stub.after()
	}
	return PairingAttempt{}, stub.matched, stub.err
}

func TestPairingAttemptServiceCommitsAuthenticationOutcomes(t *testing.T) {
	tests := map[string]struct {
		matched          bool
		cancelAfterCheck bool
		want             error
	}{
		"accepted":                    {matched: true},
		"rejected":                    {want: ErrPairingAttemptUnavailable},
		"rejected after cancellation": {cancelAfterCheck: true, want: ErrPairingAttemptUnavailable},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tx := &sessionServiceTxStub{}
			starter := &sessionServiceStarterStub{tx: tx}
			store := &pairingAttemptCredentialStoreStub{matched: test.matched}
			if test.cancelAfterCheck {
				store.after = cancel
			}
			service := newPairingAttemptServiceForTest(t, starter, store)

			err := service.authenticateBootstrapCredential(
				ctx, serviceTestSessionID, validPairingBootstrapCredential(),
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("authenticateBootstrapCredential() error = %v, want %v", err, test.want)
			}
			if starter.calls != 1 || store.calls != 1 || store.tx != tx || store.id != serviceTestSessionID ||
				tx.commitCalls != 1 || tx.rollbackCalls != 1 {
				t.Fatalf(
					"calls = begin %d, authenticate %d, commit %d, rollback %d",
					starter.calls, store.calls, tx.commitCalls, tx.rollbackCalls,
				)
			}
		})
	}
}

func TestPairingAttemptServiceFailsClosed(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	commitErr := errors.New("commit acknowledgement lost")
	tests := map[string]struct {
		storeErr  error
		commitErr error
		want      error
		commit    int
	}{
		"unavailable attempt": {storeErr: ErrPairingAttemptUnavailable, want: ErrPairingAttemptUnavailable},
		"persistence failure": {storeErr: databaseErr, want: ErrPairingAttemptPersistenceUnavailable},
		"ambiguous rejection commit": {
			commitErr: commitErr, want: ErrPairingAttemptCommitOutcomeUnknown, commit: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &sessionServiceTxStub{commitErr: test.commitErr}
			starter := &sessionServiceStarterStub{tx: tx}
			store := &pairingAttemptCredentialStoreStub{err: test.storeErr}
			service := newPairingAttemptServiceForTest(t, starter, store)

			err := service.authenticateBootstrapCredential(
				context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("authenticateBootstrapCredential() error = %v, want %v", err, test.want)
			}
			if test.commitErr != nil && !errors.Is(err, test.commitErr) {
				t.Fatalf("authenticateBootstrapCredential() error = %v, want commit cause", err)
			}
			if tx.commitCalls != test.commit || tx.rollbackCalls != 1 {
				t.Fatalf("transaction calls = commit %d, rollback %d", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestPairingAttemptServiceRejectsInvalidBoundaries(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptCredentialStoreStub{matched: true}
	service := newPairingAttemptServiceForTest(t, starter, store)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.authenticateBootstrapCredential(
		nil, serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("authenticateBootstrapCredential(nil context) error = %v", err)
	}
	if err := service.authenticateBootstrapCredential(
		canceledCtx, serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("authenticateBootstrapCredential(canceled context) error = %v", err)
	}
	service.operationMax = 0
	if err := service.authenticateBootstrapCredential(
		context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("authenticateBootstrapCredential(invalid timeout) error = %v", err)
	}
	var nilService *pairingAttemptService
	if err := nilService.authenticateBootstrapCredential(
		context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil pairingAttemptService authenticateBootstrapCredential() error = %v", err)
	}
	if starter.calls != 0 || store.calls != 0 {
		t.Fatal("invalid pairing-attempt service boundary reached persistence")
	}

	if _, err := newPairingAttemptService(nil, store); err == nil {
		t.Fatal("newPairingAttemptService(nil transactions) error = nil")
	}
	if _, err := newPairingAttemptService(starter, nil); err == nil {
		t.Fatal("newPairingAttemptService(nil attempts) error = nil")
	}
}

func newPairingAttemptServiceForTest(
	t *testing.T,
	starter transactionStarter,
	store pairingAttemptCredentialStore,
) *pairingAttemptService {
	t.Helper()
	service, err := newPairingAttemptService(starter, store)
	if err != nil {
		t.Fatalf("newPairingAttemptService() error = %v", err)
	}
	return service
}
