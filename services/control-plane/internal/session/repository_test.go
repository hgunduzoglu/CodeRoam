package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
)

func TestRepositoryCreateRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	metadata, err := NewSession(
		owner,
		"2123456789abcdef0123456789abcdef",
		"3123456789abcdef0123456789abcdef",
		"4123456789abcdef0123456789abcdef",
		"5123456789abcdef0123456789abcdef",
		"eu-central-1",
		time.Date(2026, time.July, 19, 13, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	var nilRepository *Repository
	if err := nilRepository.Create(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("nil Repository Create() error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	if err := repository.Create(nil, nil, metadata); !errors.Is(err, ErrSessionPersistenceUnavailable) {
		t.Fatalf("Create(nil context) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Create(canceledCtx, nil, metadata); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.Create(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("Create(nil transaction) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	repository.operationMax = 0
	if err := repository.Create(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("Create(invalid operation maximum) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
}

func TestRepositoryCreateOrGetRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	metadata, err := NewSession(
		owner,
		"2123456789abcdef0123456789abcdef",
		"3123456789abcdef0123456789abcdef",
		"4123456789abcdef0123456789abcdef",
		"5123456789abcdef0123456789abcdef",
		"eu-central-1",
		time.Date(2026, time.July, 19, 13, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	var nilRepository *Repository
	if _, err := nilRepository.CreateOrGet(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("nil Repository CreateOrGet() error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	if _, err := repository.CreateOrGet(nil, nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(nil context) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repository.CreateOrGet(canceledCtx, nil, metadata); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateOrGet(canceled context) error = %v, want context.Canceled", err)
	}
	if _, err := repository.CreateOrGet(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(nil transaction) error = %v, want ErrSessionPersistenceUnavailable", err)
	}
	if _, err := repository.CreateOrGet(context.Background(), nil, Session{}); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(zero session, nil tx) error = %v, want persistence unavailable", err)
	}
	repository.operationMax = 0
	if _, err := repository.CreateOrGet(context.Background(), nil, metadata); !errors.Is(
		err, ErrSessionPersistenceUnavailable,
	) {
		t.Fatalf("CreateOrGet(invalid operation maximum) error = %v, want persistence unavailable", err)
	}
}

func TestPairingAttemptRepositoryRejectsInvalidBoundaries(t *testing.T) {
	repository := NewRepository()
	attempt, err := NewPairingAttempt(validPairingAttemptSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttempt() error = %v", err)
	}
	checkedAt := attempt.createdAt.Add(time.Minute)

	var nilRepository *Repository
	if err := nilRepository.CreatePairingAttempt(context.Background(), nil, attempt); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("nil Repository CreatePairingAttempt() error = %v", err)
	}
	if err := repository.CreatePairingAttempt(nil, nil, attempt); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("CreatePairingAttempt(nil context) error = %v", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.CreatePairingAttempt(canceledCtx, nil, attempt); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreatePairingAttempt(canceled context) error = %v", err)
	}
	if err := repository.CreatePairingAttempt(context.Background(), nil, attempt); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("CreatePairingAttempt(nil transaction) error = %v", err)
	}
	if _, err := nilRepository.LockOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil Repository LockOpenPairingAttempt() error = %v", err)
	}
	if _, err := repository.LockOpenPairingAttempt(nil, nil, attempt.id.String()); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("LockOpenPairingAttempt(nil context) error = %v", err)
	}
	if _, err := repository.LockOpenPairingAttempt(
		canceledCtx, nil, attempt.id.String(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("LockOpenPairingAttempt(canceled context) error = %v", err)
	}
	if _, err := repository.LockOpenPairingAttempt(
		context.Background(), nil, "invalid",
	); !errors.Is(err, ErrInvalidPairingAttempt) {
		t.Fatalf("LockOpenPairingAttempt(invalid id) error = %v", err)
	}
	repository.now = func() time.Time { return time.Time{} }
	if _, err := repository.LockOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("LockOpenPairingAttempt(zero clock) error = %v", err)
	}
	repository.now = func() time.Time { return checkedAt }
	if _, err := repository.LockOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("LockOpenPairingAttempt(nil transaction) error = %v", err)
	}
	if _, _, err := nilRepository.authenticateOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil Repository AuthenticateOpenPairingAttempt() error = %v", err)
	}
	if _, _, err := repository.authenticateOpenPairingAttempt(
		nil, nil, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("AuthenticateOpenPairingAttempt(nil context) error = %v", err)
	}
	if _, _, err := repository.authenticateOpenPairingAttempt(
		canceledCtx, nil, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("AuthenticateOpenPairingAttempt(canceled context) error = %v", err)
	}
	if _, _, err := repository.authenticateOpenPairingAttempt(
		context.Background(), nil, "invalid", validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptUnavailable) {
		t.Fatalf("AuthenticateOpenPairingAttempt(invalid id) error = %v", err)
	}
	if _, _, err := repository.authenticateOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("AuthenticateOpenPairingAttempt(nil transaction) error = %v", err)
	}
	claim, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	if err := nilRepository.claimOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(), claim,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil Repository claimOpenPairingAttempt() error = %v", err)
	}
	if err := repository.claimOpenPairingAttempt(
		nil, nil, attempt.id.String(), claim,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("claimOpenPairingAttempt(nil context) error = %v", err)
	}
	if err := repository.claimOpenPairingAttempt(
		canceledCtx, nil, attempt.id.String(), claim,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("claimOpenPairingAttempt(canceled context) error = %v", err)
	}
	if err := repository.claimOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(), claim,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("claimOpenPairingAttempt(nil transaction) error = %v", err)
	}
	invalidClaim := claim
	invalidClaim.deviceFingerprint = cryptox.X25519Fingerprint{}
	if err := repository.claimOpenPairingAttempt(
		context.Background(), &sessionServiceTxStub{}, attempt.id.String(), invalidClaim,
	); !errors.Is(err, ErrInvalidPairingAttemptClaim) {
		t.Fatalf("claimOpenPairingAttempt(invalid claim) error = %v", err)
	}
	confirmation, err := NewMobilePairingConfirmation(validMobilePairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
	}
	if err := nilRepository.confirmMobilePairingAttempt(
		context.Background(), nil, attempt.id.String(), confirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil Repository confirmMobilePairingAttempt() error = %v", err)
	}
	if err := repository.confirmMobilePairingAttempt(
		nil, nil, attempt.id.String(), confirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("confirmMobilePairingAttempt(nil context) error = %v", err)
	}
	if err := repository.confirmMobilePairingAttempt(
		canceledCtx, nil, attempt.id.String(), confirmation,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("confirmMobilePairingAttempt(canceled context) error = %v", err)
	}
	if err := repository.confirmMobilePairingAttempt(
		context.Background(), nil, attempt.id.String(), confirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("confirmMobilePairingAttempt(nil transaction) error = %v", err)
	}
	invalidConfirmation := confirmation
	invalidConfirmation.channelBinding = [pairingChannelBindingLen]byte{}
	if err := repository.confirmMobilePairingAttempt(
		context.Background(), &sessionServiceTxStub{}, attempt.id.String(), invalidConfirmation,
	); !errors.Is(err, ErrInvalidPairingAttemptConfirmation) {
		t.Fatalf("confirmMobilePairingAttempt(invalid confirmation) error = %v", err)
	}
	agentConfirmation, err := NewAgentPairingConfirmation(validAgentPairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewAgentPairingConfirmation() error = %v", err)
	}
	if _, err := nilRepository.confirmAgentPairingAttempt(
		context.Background(), nil, attempt.id.String(),
		validPairingBootstrapCredential(), agentConfirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("nil Repository confirmAgentPairingAttempt() error = %v", err)
	}
	if _, err := repository.confirmAgentPairingAttempt(
		nil, nil, attempt.id.String(), validPairingBootstrapCredential(), agentConfirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("confirmAgentPairingAttempt(nil context) error = %v", err)
	}
	if _, err := repository.confirmAgentPairingAttempt(
		canceledCtx, nil, attempt.id.String(), validPairingBootstrapCredential(), agentConfirmation,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("confirmAgentPairingAttempt(canceled context) error = %v", err)
	}
	if _, err := repository.confirmAgentPairingAttempt(
		context.Background(), nil, attempt.id.String(),
		validPairingBootstrapCredential(), agentConfirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("confirmAgentPairingAttempt(nil transaction) error = %v", err)
	}
	invalidAgentConfirmation := agentConfirmation
	invalidAgentConfirmation.channelBinding = [pairingChannelBindingLen]byte{}
	if _, err := repository.confirmAgentPairingAttempt(
		context.Background(), &sessionServiceTxStub{}, attempt.id.String(),
		validPairingBootstrapCredential(), invalidAgentConfirmation,
	); !errors.Is(err, ErrInvalidPairingAttemptConfirmation) {
		t.Fatalf("confirmAgentPairingAttempt(invalid confirmation) error = %v", err)
	}
	repository.operationMax = 0
	if err := repository.CreatePairingAttempt(context.Background(), nil, attempt); !errors.Is(
		err, ErrPairingAttemptPersistenceUnavailable,
	) {
		t.Fatalf("CreatePairingAttempt(invalid operation maximum) error = %v", err)
	}
	if _, _, err := repository.authenticateOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(), validPairingBootstrapCredential(),
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("AuthenticateOpenPairingAttempt(invalid operation maximum) error = %v", err)
	}
	if err := repository.claimOpenPairingAttempt(
		context.Background(), nil, attempt.id.String(), claim,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("claimOpenPairingAttempt(invalid operation maximum) error = %v", err)
	}
	if err := repository.confirmMobilePairingAttempt(
		context.Background(), nil, attempt.id.String(), confirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("confirmMobilePairingAttempt(invalid operation maximum) error = %v", err)
	}
	if _, err := repository.confirmAgentPairingAttempt(
		context.Background(), nil, attempt.id.String(),
		validPairingBootstrapCredential(), agentConfirmation,
	); !errors.Is(err, ErrPairingAttemptPersistenceUnavailable) {
		t.Fatalf("confirmAgentPairingAttempt(invalid operation maximum) error = %v", err)
	}
}
