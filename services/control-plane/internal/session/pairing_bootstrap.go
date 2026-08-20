package session

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"github.com/jackc/pgx/v5"
)

const pairingTicketClockSkew = 5 * time.Second

var (
	ErrPairingBootstrapUnavailable          = errors.New("pairing bootstrap unavailable")
	ErrPairingBootstrapCommitOutcomeUnknown = errors.New("pairing bootstrap commit outcome unknown")
)

type pairingAttemptCreator interface {
	CreatePairingAttempt(context.Context, pgx.Tx, PairingAttempt) error
}

type pairingTicketIssuer interface {
	SignPairingTicket(*relayv1.ConnectionTicketClaims) ([]byte, error)
}

// PairingBootstrapSpec contains only agent-created public metadata. PairingSecret must never be
// added here: it remains local to the agent and mobile endpoint for the XXpsk3 handshake.
type PairingBootstrapSpec struct {
	PairingID        string
	AgentID          string
	AgentPublicKey   cryptox.X25519PublicKey
	AgentFingerprint string
	AgentDisplayName string
	AgentVersion     string
	ProtocolVersion  int
	ExpiresAt        time.Time
}

// PairingBootstrap contains short-lived capabilities. Callers must return them once, must not log
// or persist BootstrapCredential, and must clear their copies when the attempt ends.
type PairingBootstrap struct {
	PairingID           string
	AgentID             string
	RelayRegion         string
	ExpiresAt           time.Time
	BootstrapCredential []byte
	AgentTicket         []byte
}

// PairingBootstrapService creates one bounded, hash-only pairing attempt and its agent ticket.
type PairingBootstrapService struct {
	transactions transactionStarter
	attempts     pairingAttemptCreator
	tickets      pairingTicketIssuer
	relayRegion  string
	now          func() time.Time
	random       io.Reader
	newID        func() (ids.ID, error)
	operationMax time.Duration
}

// NewPairingBootstrapService validates the persistence, signing, region, and clock boundaries.
func NewPairingBootstrapService(
	transactions transactionStarter,
	attempts pairingAttemptCreator,
	tickets pairingTicketIssuer,
	relayRegion string,
	now func() time.Time,
) (*PairingBootstrapService, error) {
	if transactions == nil || attempts == nil || tickets == nil {
		return nil, errors.New("pairing bootstrap service dependencies are required")
	}
	if !validRelayRegion(relayRegion) {
		return nil, errors.New("pairing bootstrap relay region is invalid")
	}
	if now == nil {
		return nil, errors.New("pairing bootstrap clock is required")
	}
	return &PairingBootstrapService{
		transactions: transactions,
		attempts:     attempts,
		tickets:      tickets,
		relayRegion:  relayRegion,
		now:          now,
		random:       rand.Reader,
		newID:        ids.New,
		operationMax: serviceOperationTimeout,
	}, nil
}

// Start persists the credential hash and returns the raw credential and signed ticket only after
// a successful commit. A commit error deliberately returns no capabilities because the raw
// credential cannot be reconstructed from its persisted hash.
func (service *PairingBootstrapService) Start(
	ctx context.Context,
	spec PairingBootstrapSpec,
) (bootstrap PairingBootstrap, err error) {
	if ctx == nil || service == nil || service.transactions == nil || service.attempts == nil ||
		service.tickets == nil || service.now == nil || service.random == nil || service.newID == nil ||
		service.operationMax <= 0 {
		return PairingBootstrap{}, ErrPairingBootstrapUnavailable
	}
	if err := ctx.Err(); err != nil {
		return PairingBootstrap{}, err
	}

	requestedFingerprint, fingerprintErr := cryptox.ParseX25519Fingerprint(spec.AgentFingerprint)
	computedFingerprint, keyErr := cryptox.FingerprintX25519PublicKey(spec.AgentPublicKey)
	if fingerprintErr != nil || keyErr != nil || !requestedFingerprint.Equal(computedFingerprint) {
		return PairingBootstrap{}, fmt.Errorf("%w: agent fingerprint", ErrInvalidPairingAttempt)
	}

	createdAt := service.now().UTC().Truncate(time.Microsecond)
	if createdAt.IsZero() {
		return PairingBootstrap{}, ErrPairingBootstrapUnavailable
	}

	credential := make([]byte, pairingBootstrapCredentialLen)
	var ticket []byte
	succeeded := false
	defer func() {
		if succeeded {
			return
		}
		clear(credential)
		clear(ticket)
		bootstrap = PairingBootstrap{}
	}()
	if err := readPairingBootstrapRandom(service.random, credential); err != nil {
		return PairingBootstrap{}, err
	}
	credentialHash, err := HashPairingBootstrapCredential(spec.PairingID, credential)
	if err != nil {
		return PairingBootstrap{}, fmt.Errorf("%w: pairing id", ErrInvalidPairingAttempt)
	}
	attempt, err := NewPairingAttempt(PairingAttemptSpec{
		ID: spec.PairingID, AgentID: spec.AgentID, AgentPublicKey: spec.AgentPublicKey,
		AgentDisplayName: spec.AgentDisplayName, AgentVersion: spec.AgentVersion,
		ProtocolVersion: spec.ProtocolVersion, RelayRegion: service.relayRegion,
		BootstrapCredentialHash: credentialHash[:], CreatedAt: createdAt, ExpiresAt: spec.ExpiresAt,
	})
	if err != nil {
		return PairingBootstrap{}, err
	}

	nonce := make([]byte, pairingTicketNonceSize)
	defer clear(nonce)
	if err := readPairingBootstrapRandom(service.random, nonce); err != nil {
		return PairingBootstrap{}, err
	}
	ticketID, err := service.newID()
	if err != nil {
		return PairingBootstrap{}, fmt.Errorf("%w: generate ticket id: %w", ErrPairingBootstrapUnavailable, err)
	}
	ticketIssuedAt := createdAt.Truncate(time.Second)
	ticketExpiresAt := ticketIssuedAt.Add(maxPairingTicketLifetime - pairingTicketClockSkew)
	attemptTicketExpiry := attempt.expiresAt.Truncate(time.Second).Add(-pairingTicketClockSkew)
	if attemptTicketExpiry.Before(ticketExpiresAt) {
		ticketExpiresAt = attemptTicketExpiry
	}
	if !ticketExpiresAt.After(ticketIssuedAt) {
		return PairingBootstrap{}, fmt.Errorf("%w: expiry leaves no usable ticket lifetime", ErrInvalidPairingAttempt)
	}
	ticket, err = service.tickets.SignPairingTicket(&relayv1.ConnectionTicketClaims{
		TicketId: ticketID.String(), RouteId: spec.PairingID,
		Role: relayv1.EndpointRole_ENDPOINT_ROLE_AGENT, EndpointId: spec.AgentID,
		RelayRegion: service.relayRegion, IssuedAtUnixSeconds: ticketIssuedAt.Unix(),
		ExpiresAtUnixSeconds: ticketExpiresAt.Unix(),
		Nonce:                nonce, Purpose: relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING,
		ProtocolVersion:      pairingTicketProtocolVersion,
		NotBeforeUnixSeconds: ticketIssuedAt.Add(-pairingTicketClockSkew).Unix(),
	})
	if err != nil {
		return PairingBootstrap{}, fmt.Errorf("%w: sign agent ticket: %w", ErrPairingBootstrapUnavailable, err)
	}
	if len(ticket) == 0 || len(ticket) > maxPairingTicketEncodedSize {
		return PairingBootstrap{}, fmt.Errorf("%w: signer returned invalid ticket size", ErrPairingBootstrapUnavailable)
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, service.operationMax)
	defer cancelOperation()
	tx, beginErr := service.transactions.Begin(operationCtx)
	if tx == nil {
		return PairingBootstrap{}, pairingBootstrapServiceError("begin", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), serviceCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = pairingBootstrapServiceError("rollback", rollbackErr)
		if err == nil {
			err = rollbackErr
			succeeded = false
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return PairingBootstrap{}, pairingBootstrapServiceError("begin", beginErr)
	}
	if err := service.attempts.CreatePairingAttempt(operationCtx, tx, attempt); err != nil {
		if errors.Is(err, ErrPairingAttemptAlreadyExists) || errors.Is(err, ErrInvalidPairingAttempt) {
			return PairingBootstrap{}, err
		}
		return PairingBootstrap{}, pairingBootstrapServiceError("persist", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return PairingBootstrap{}, fmt.Errorf("%w: %w", ErrPairingBootstrapCommitOutcomeUnknown, err)
	}

	succeeded = true
	return PairingBootstrap{
		PairingID: spec.PairingID, AgentID: spec.AgentID, RelayRegion: service.relayRegion,
		ExpiresAt: attempt.expiresAt, BootstrapCredential: credential, AgentTicket: ticket,
	}, nil
}

func readPairingBootstrapRandom(source io.Reader, destination []byte) error {
	if _, err := io.ReadFull(source, destination); err != nil {
		return fmt.Errorf("%w: generate capability: %w", ErrPairingBootstrapUnavailable, err)
	}
	if allZero(destination) {
		return fmt.Errorf("%w: generated all-zero capability", ErrPairingBootstrapUnavailable)
	}
	return nil
}

func pairingBootstrapServiceError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if err == nil {
		return fmt.Errorf("%w: %s returned no transaction", ErrPairingBootstrapUnavailable, operation)
	}
	return fmt.Errorf("%w: %s: %w", ErrPairingBootstrapUnavailable, operation, err)
}
