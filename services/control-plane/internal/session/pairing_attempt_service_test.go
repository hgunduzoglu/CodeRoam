package session

import (
	"context"
	"errors"
	"testing"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/jackc/pgx/v5"
)

type pairingAttemptStoreStub struct {
	matched      bool
	err          error
	after        func()
	calls        int
	tx           pgx.Tx
	id           string
	claimErr     error
	claimCalls   int
	claimTx      pgx.Tx
	claimID      string
	claim        PairingAttemptClaim
	confirmErr   error
	confirmCalls int
	confirmTx    pgx.Tx
	confirmID    string
	confirmation MobilePairingConfirmation
}

func (stub *pairingAttemptStoreStub) authenticateOpenPairingAttempt(
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

func (stub *pairingAttemptStoreStub) claimOpenPairingAttempt(
	_ context.Context,
	tx pgx.Tx,
	id string,
	claim PairingAttemptClaim,
) error {
	stub.claimCalls++
	stub.claimTx = tx
	stub.claimID = id
	stub.claim = claim
	return stub.claimErr
}

func (stub *pairingAttemptStoreStub) confirmMobilePairingAttempt(
	_ context.Context,
	tx pgx.Tx,
	id string,
	confirmation MobilePairingConfirmation,
) error {
	stub.confirmCalls++
	stub.confirmTx = tx
	stub.confirmID = id
	stub.confirmation = confirmation
	return stub.confirmErr
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
			store := &pairingAttemptStoreStub{matched: test.matched}
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
			store := &pairingAttemptStoreStub{err: test.storeErr}
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
	store := &pairingAttemptStoreStub{matched: true}
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

func TestPairingAttemptServiceClaimsInOneTransaction(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptStoreStub{}
	service := newPairingAttemptServiceForTest(t, starter, store)
	claim, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}

	if err := service.claim(context.Background(), serviceTestSessionID, claim); err != nil {
		t.Fatalf("claim() error = %v", err)
	}
	if starter.calls != 1 || store.claimCalls != 1 || store.claimTx != tx ||
		store.claimID != serviceTestSessionID || !store.claim.valid() ||
		tx.commitCalls != 1 || tx.rollbackCalls != 1 {
		t.Fatalf(
			"calls = begin %d, claim %d, commit %d, rollback %d",
			starter.calls, store.claimCalls, tx.commitCalls, tx.rollbackCalls,
		)
	}
}

func TestPairingAttemptServiceClaimFailsClosed(t *testing.T) {
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
		"ambiguous commit": {
			commitErr: commitErr, want: ErrPairingAttemptCommitOutcomeUnknown, commit: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &sessionServiceTxStub{commitErr: test.commitErr}
			starter := &sessionServiceStarterStub{tx: tx}
			store := &pairingAttemptStoreStub{claimErr: test.storeErr}
			service := newPairingAttemptServiceForTest(t, starter, store)
			claim, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
			if err != nil {
				t.Fatalf("NewPairingAttemptClaim() error = %v", err)
			}

			err = service.claim(context.Background(), serviceTestSessionID, claim)
			if !errors.Is(err, test.want) {
				t.Fatalf("claim() error = %v, want %v", err, test.want)
			}
			if test.commitErr != nil && !errors.Is(err, test.commitErr) {
				t.Fatalf("claim() error = %v, want commit cause", err)
			}
			if tx.commitCalls != test.commit || tx.rollbackCalls != 1 {
				t.Fatalf("transaction calls = commit %d, rollback %d", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestPairingAttemptServiceClaimRejectsInvalidBoundaryBeforePersistence(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptStoreStub{}
	service := newPairingAttemptServiceForTest(t, starter, store)
	valid, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	invalid := valid
	invalid.deviceFingerprint = cryptox.X25519Fingerprint{}

	if err := service.claim(context.Background(), serviceTestSessionID, invalid); !errors.Is(
		err, ErrInvalidPairingAttemptClaim,
	) {
		t.Fatalf("claim(invalid) error = %v", err)
	}
	var nilService *pairingAttemptService
	if err := nilService.claim(context.Background(), serviceTestSessionID, valid); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("nil service claim() error = %v", err)
	}
	if starter.calls != 0 || store.claimCalls != 0 || tx.commitCalls != 0 {
		t.Fatal("invalid claim reached persistence")
	}
}

func TestPairingAttemptServiceConfirmsMobileInOneTransaction(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptStoreStub{}
	service := newPairingAttemptServiceForTest(t, starter, store)
	confirmation, err := NewMobilePairingConfirmation(validMobilePairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
	}

	if err := service.confirmMobile(
		context.Background(), serviceTestSessionID, confirmation,
	); err != nil {
		t.Fatalf("confirmMobile() error = %v", err)
	}
	if starter.calls != 1 || store.confirmCalls != 1 || store.confirmTx != tx ||
		store.confirmID != serviceTestSessionID || !store.confirmation.valid() ||
		tx.commitCalls != 1 || tx.rollbackCalls != 1 {
		t.Fatalf(
			"calls = begin %d, confirm %d, commit %d, rollback %d",
			starter.calls, store.confirmCalls, tx.commitCalls, tx.rollbackCalls,
		)
	}
}

func TestPairingAttemptServiceMobileConfirmationFailsClosed(t *testing.T) {
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
		"ambiguous commit": {
			commitErr: commitErr, want: ErrPairingAttemptCommitOutcomeUnknown, commit: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &sessionServiceTxStub{commitErr: test.commitErr}
			starter := &sessionServiceStarterStub{tx: tx}
			store := &pairingAttemptStoreStub{confirmErr: test.storeErr}
			service := newPairingAttemptServiceForTest(t, starter, store)
			confirmation, err := NewMobilePairingConfirmation(validMobilePairingConfirmationSpec(t))
			if err != nil {
				t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
			}

			err = service.confirmMobile(context.Background(), serviceTestSessionID, confirmation)
			if !errors.Is(err, test.want) {
				t.Fatalf("confirmMobile() error = %v, want %v", err, test.want)
			}
			if test.commitErr != nil && !errors.Is(err, test.commitErr) {
				t.Fatalf("confirmMobile() error = %v, want commit cause", err)
			}
			if tx.commitCalls != test.commit || tx.rollbackCalls != 1 {
				t.Fatalf("transaction calls = commit %d, rollback %d", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestPairingAttemptServiceMobileConfirmationRejectsInvalidBoundary(t *testing.T) {
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptStoreStub{}
	service := newPairingAttemptServiceForTest(t, starter, store)
	valid, err := NewMobilePairingConfirmation(validMobilePairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
	}
	invalid := valid
	invalid.channelBinding = [pairingChannelBindingLen]byte{}

	if err := service.confirmMobile(
		context.Background(), serviceTestSessionID, invalid,
	); !errors.Is(err, ErrInvalidPairingAttemptConfirmation) {
		t.Fatalf("confirmMobile(invalid) error = %v", err)
	}
	var nilService *pairingAttemptService
	if err := nilService.confirmMobile(
		context.Background(), serviceTestSessionID, valid,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil service confirmMobile() error = %v", err)
	}
	if starter.calls != 0 || store.confirmCalls != 0 || tx.commitCalls != 0 {
		t.Fatal("invalid mobile confirmation reached persistence")
	}
}

func newPairingAttemptServiceForTest(
	t *testing.T,
	starter transactionStarter,
	store pairingAttemptStore,
) *pairingAttemptService {
	t.Helper()
	service, err := newPairingAttemptService(starter, store)
	if err != nil {
		t.Fatalf("newPairingAttemptService() error = %v", err)
	}
	return service
}
