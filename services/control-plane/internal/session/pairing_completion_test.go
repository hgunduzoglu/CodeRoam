package session

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
)

type pairingCompletionStoreStub struct {
	candidate    pairingCompletionCandidate
	lockErr      error
	consumeErr   error
	lockCalls    int
	consumeCalls int
	tx           pgx.Tx
	id           string
	order        *[]string
}

func (stub *pairingCompletionStoreStub) lockPairingCompletion(
	_ context.Context,
	tx pgx.Tx,
	id string,
	_ []byte,
) (pairingCompletionCandidate, error) {
	stub.lockCalls++
	stub.tx = tx
	stub.id = id
	if stub.order != nil {
		*stub.order = append(*stub.order, "lock")
	}
	return stub.candidate, stub.lockErr
}

func (stub *pairingCompletionStoreStub) consumePairingAttempt(
	_ context.Context,
	tx pgx.Tx,
	_ pairingCompletionCandidate,
) error {
	stub.consumeCalls++
	stub.tx = tx
	if stub.order != nil {
		*stub.order = append(*stub.order, "consume")
	}
	return stub.consumeErr
}

type pairingDeviceRegistrarStub struct {
	err   error
	calls int
	tx    pgx.Tx
	order *[]string
}

func (stub *pairingDeviceRegistrarStub) RegisterPaired(
	_ context.Context,
	tx pgx.Tx,
	_ auth.UserID,
	_, _ string,
	_ device.Platform,
	_ cryptox.X25519PublicKey,
	_ time.Time,
) error {
	stub.calls++
	stub.tx = tx
	if stub.order != nil {
		*stub.order = append(*stub.order, "device")
	}
	return stub.err
}

type pairingAgentRegistrarStub struct {
	err   error
	calls int
	tx    pgx.Tx
	order *[]string
}

func (stub *pairingAgentRegistrarStub) RegisterPairedAgent(
	_ context.Context,
	tx pgx.Tx,
	_ auth.UserID,
	_, _ string,
	_ cryptox.X25519PublicKey,
	_ string,
	_ time.Time,
) error {
	stub.calls++
	stub.tx = tx
	if stub.order != nil {
		*stub.order = append(*stub.order, "agent")
	}
	return stub.err
}

func TestPairingCompletionServiceUsesOneOrderedTransaction(t *testing.T) {
	order := make([]string, 0, 4)
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingCompletionStoreStub{candidate: validPairingCompletionCandidate(t), order: &order}
	devices := &pairingDeviceRegistrarStub{order: &order}
	agents := &pairingAgentRegistrarStub{order: &order}
	service := newPairingCompletionServiceForTest(t, starter, store, devices, agents)

	completed, err := service.complete(
		context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
	)
	if err != nil {
		t.Fatalf("complete() error = %v", err)
	}
	if completed.PairingID != serviceTestSessionID ||
		completed.DeviceID != serviceTestDeviceID ||
		completed.AgentID != serviceTestAgentID || completed.ConsumedAt.IsZero() {
		t.Fatal("complete() did not return the exact stable completion metadata")
	}
	if starter.calls != 1 || store.lockCalls != 1 || devices.calls != 1 ||
		agents.calls != 1 || store.consumeCalls != 1 || tx.commitCalls != 1 || tx.rollbackCalls != 1 {
		t.Fatal("complete() did not execute every operation exactly once")
	}
	if store.tx != tx || devices.tx != tx || agents.tx != tx {
		t.Fatal("complete() did not pass one transaction through every owning module")
	}
	wantOrder := []string{"lock", "device", "agent", "consume"}
	for index := range wantOrder {
		if order[index] != wantOrder[index] {
			t.Fatalf("completion order = %v, want %v", order, wantOrder)
		}
	}
}

func TestPairingCompletionServiceReconcilesConsumedAttemptWithoutReRegistration(t *testing.T) {
	tx := &sessionServiceTxStub{}
	candidate := validPairingCompletionCandidate(t)
	candidate.alreadyConsumed = true
	candidate.locked.attempt.state = pairingAttemptStateConsumed
	candidate.locked.consumedAt = candidate.consumedAt
	store := &pairingCompletionStoreStub{candidate: candidate}
	devices := &pairingDeviceRegistrarStub{}
	agents := &pairingAgentRegistrarStub{}
	service := newPairingCompletionServiceForTest(
		t, &sessionServiceStarterStub{tx: tx}, store, devices, agents,
	)

	completed, err := service.complete(
		context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
	)
	if err != nil {
		t.Fatalf("complete(consumed retry) error = %v", err)
	}
	if completed.ConsumedAt != candidate.consumedAt || devices.calls != 0 ||
		agents.calls != 0 || store.consumeCalls != 0 || tx.commitCalls != 1 {
		t.Fatal("consumed completion retry mutated endpoint registrations")
	}
}

func TestPairingCompletionServiceFailsClosedAndRollsBack(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	commitErr := errors.New("commit acknowledgement lost")
	tests := map[string]struct {
		lockErr    error
		deviceErr  error
		agentErr   error
		consumeErr error
		commitErr  error
		want       error
		calls      [3]int
		commit     int
	}{
		"confirmation unavailable": {
			lockErr: ErrPairingAttemptUnavailable, want: ErrPairingAttemptUnavailable,
		},
		"device revoked": {
			deviceErr: device.ErrDeviceAccessDenied, want: ErrPairingAttemptUnavailable,
			calls: [3]int{1, 0, 0},
		},
		"agent collision": {
			agentErr: workspace.ErrAgentAccessDenied, want: ErrPairingAttemptUnavailable,
			calls: [3]int{1, 1, 0},
		},
		"consume race": {
			consumeErr: ErrPairingAttemptUnavailable, want: ErrPairingAttemptUnavailable,
			calls: [3]int{1, 1, 1},
		},
		"persistence failure": {
			consumeErr: databaseErr, want: ErrPairingAttemptPersistenceUnavailable,
			calls: [3]int{1, 1, 1},
		},
		"ambiguous commit": {
			commitErr: commitErr, want: ErrPairingAttemptCommitOutcomeUnknown,
			calls: [3]int{1, 1, 1}, commit: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &sessionServiceTxStub{commitErr: test.commitErr}
			store := &pairingCompletionStoreStub{
				candidate: validPairingCompletionCandidate(t), lockErr: test.lockErr,
				consumeErr: test.consumeErr,
			}
			devices := &pairingDeviceRegistrarStub{err: test.deviceErr}
			agents := &pairingAgentRegistrarStub{err: test.agentErr}
			service := newPairingCompletionServiceForTest(
				t, &sessionServiceStarterStub{tx: tx}, store, devices, agents,
			)

			completed, err := service.complete(
				context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("complete() error = %v, want %v", err, test.want)
			}
			if completed != (PairingCompletion{}) {
				t.Fatal("failed completion returned registration metadata")
			}
			gotCalls := [3]int{devices.calls, agents.calls, store.consumeCalls}
			if gotCalls != test.calls || tx.commitCalls != test.commit || tx.rollbackCalls != 1 {
				t.Fatalf(
					"completion calls = %v, commit %d, rollback %d",
					gotCalls, tx.commitCalls, tx.rollbackCalls,
				)
			}
		})
	}
}

func TestPairingCompletionServiceRejectsInvalidDependenciesAndContext(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingCompletionStoreStub{candidate: validPairingCompletionCandidate(t)}
	devices := &pairingDeviceRegistrarStub{}
	agents := &pairingAgentRegistrarStub{}
	dependencies := []struct {
		transactions transactionStarter
		attempts     pairingCompletionStore
		devices      pairedDeviceRegistrar
		agents       pairedAgentRegistrar
	}{
		{attempts: store, devices: devices, agents: agents},
		{transactions: starter, devices: devices, agents: agents},
		{transactions: starter, attempts: store, agents: agents},
		{transactions: starter, attempts: store, devices: devices},
	}
	for _, dependencies := range dependencies {
		if _, err := newPairingCompletionService(
			dependencies.transactions, dependencies.attempts,
			dependencies.devices, dependencies.agents,
		); err == nil {
			t.Fatal("newPairingCompletionService(invalid dependencies) error = nil")
		}
	}

	service := newPairingCompletionServiceForTest(t, starter, store, devices, agents)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.complete(
		canceledCtx, serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("complete(canceled context) error = %v", err)
	}
	var nilService *pairingCompletionService
	if _, err := nilService.complete(
		context.Background(), serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("nil service complete() error = %v", err)
	}
	if starter.calls != 0 || store.lockCalls != 0 {
		t.Fatal("invalid completion boundary reached persistence")
	}
}

func TestPairingCompletionRepositoryRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	candidate := validPairingCompletionCandidate(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := repository.lockPairingCompletion(
		nil, nil, serviceTestSessionID, validPairingBootstrapCredential(),
	); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("lockPairingCompletion(nil boundary) error = %v", err)
	}
	if _, err := repository.lockPairingCompletion(
		canceledCtx, &sessionServiceTxStub{}, serviceTestSessionID,
		validPairingBootstrapCredential(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("lockPairingCompletion(canceled context) error = %v", err)
	}
	if err := repository.consumePairingAttempt(
		context.Background(), nil, candidate,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("consumePairingAttempt(nil transaction) error = %v", err)
	}
	candidate.alreadyConsumed = true
	if err := repository.consumePairingAttempt(
		context.Background(), &sessionServiceTxStub{}, candidate,
	); !errors.Is(err, ErrPairingAttemptUnavailable) {
		t.Fatalf("consumePairingAttempt(consumed candidate) error = %v", err)
	}
}

func validPairingCompletionCandidate(t *testing.T) pairingCompletionCandidate {
	t.Helper()
	attemptSpec := validPairingAttemptSpec(t)
	attemptSpec.ID = serviceTestSessionID
	attemptSpec.AgentID = serviceTestAgentID
	attempt, err := NewPairingAttempt(attemptSpec)
	if err != nil {
		t.Fatalf("NewPairingAttempt() error = %v", err)
	}
	claimSpec := validPairingAttemptClaimSpec(t)
	claimSpec.DeviceID = serviceTestDeviceID
	claimSpec.ExpectedAgentPublicKey = attempt.agentPublicKey
	claim, err := NewPairingAttemptClaim(claimSpec)
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	claimedAt := attempt.createdAt.Add(time.Minute)
	confirmedAt := claimedAt.Add(time.Minute)
	consumedAt := confirmedAt.Add(time.Minute)
	attempt.state = pairingAttemptStateConfirming
	attempt.updatedAt = confirmedAt
	attempt.lockedAt = consumedAt
	binding := bytes.Repeat([]byte{0x51}, pairingChannelBindingLen)
	locked := lockedClaimedPairingAttempt{
		attempt: attempt, claim: claim, claimedAt: claimedAt,
		mobileConfirmedAt: confirmedAt, agentConfirmedAt: confirmedAt,
		hasMobileBinding: true, hasAgentBinding: true,
	}
	copy(locked.mobileBinding[:], binding)
	copy(locked.agentBinding[:], binding)
	return pairingCompletionCandidate{locked: locked, consumedAt: consumedAt}
}

func newPairingCompletionServiceForTest(
	t *testing.T,
	starter transactionStarter,
	store pairingCompletionStore,
	devices pairedDeviceRegistrar,
	agents pairedAgentRegistrar,
) *pairingCompletionService {
	t.Helper()
	service, err := newPairingCompletionService(starter, store, devices, agents)
	if err != nil {
		t.Fatalf("newPairingCompletionService() error = %v", err)
	}
	return service
}
