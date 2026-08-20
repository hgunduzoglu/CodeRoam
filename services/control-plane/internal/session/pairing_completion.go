package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/workspace"
	"github.com/jackc/pgx/v5"
)

// PairingCompletion is the stable, non-secret result of atomically registering
// both confirmed endpoints and consuming their one-use pairing attempt.
type PairingCompletion struct {
	PairingID  string
	DeviceID   string
	AgentID    string
	ConsumedAt time.Time
}

type pairingCompletionCandidate struct {
	locked          lockedClaimedPairingAttempt
	consumedAt      time.Time
	alreadyConsumed bool
}

type pairingCompletionStore interface {
	lockPairingCompletion(context.Context, pgx.Tx, string, []byte) (pairingCompletionCandidate, error)
	consumePairingAttempt(context.Context, pgx.Tx, pairingCompletionCandidate) error
}

type pairedDeviceRegistrar interface {
	RegisterPaired(
		context.Context,
		pgx.Tx,
		auth.UserID,
		string,
		string,
		device.Platform,
		cryptox.X25519PublicKey,
		time.Time,
	) error
}

type pairedAgentRegistrar interface {
	RegisterPairedAgent(
		context.Context,
		pgx.Tx,
		auth.UserID,
		string,
		string,
		cryptox.X25519PublicKey,
		string,
		time.Time,
	) error
}

// lockPairingCompletion restores one matching two-sided confirmation under
// FOR UPDATE. A consumed row remains readable after expiry for exact outcome
// reconciliation, but an unconsumed row must still be live at the lock time.
func (repository *Repository) lockPairingCompletion(
	ctx context.Context,
	tx pgx.Tx,
	encodedID string,
	credential []byte,
) (pairingCompletionCandidate, error) {
	if ctx == nil || repository == nil || repository.now == nil ||
		repository.operationMax <= 0 || tx == nil {
		return pairingCompletionCandidate{}, ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return pairingCompletionCandidate{}, err
	}

	locked, err := repository.lockClaimedPairingAttempt(ctx, tx, encodedID)
	if err != nil {
		return pairingCompletionCandidate{}, err
	}
	candidateHash, credentialErr := HashPairingBootstrapCredential(encodedID, credential)
	matched := subtle.ConstantTimeCompare(
		candidateHash[:], locked.attempt.bootstrapCredentialHash[:],
	)
	clear(candidateHash[:])
	if err := ctx.Err(); err != nil {
		return pairingCompletionCandidate{}, err
	}
	if credentialErr != nil || matched != 1 {
		return pairingCompletionCandidate{}, ErrPairingAttemptUnavailable
	}
	if !locked.hasMobileBinding || !locked.hasAgentBinding || subtle.ConstantTimeCompare(
		locked.mobileBinding[:], locked.agentBinding[:],
	) != 1 {
		return pairingCompletionCandidate{}, ErrPairingAttemptUnavailable
	}
	if locked.attempt.state == pairingAttemptStateConsumed {
		return pairingCompletionCandidate{
			locked: locked, consumedAt: locked.consumedAt, alreadyConsumed: true,
		}, nil
	}
	if locked.attempt.state != pairingAttemptStateConfirming {
		return pairingCompletionCandidate{}, ErrPairingAttemptUnavailable
	}
	consumedAt := locked.attempt.lockedAt.UTC().Truncate(time.Microsecond)
	if consumedAt.Before(locked.attempt.updatedAt) || !locked.attempt.expiresAt.After(consumedAt) {
		return pairingCompletionCandidate{}, ErrPairingAttemptUnavailable
	}
	return pairingCompletionCandidate{locked: locked, consumedAt: consumedAt}, nil
}

// consumePairingAttempt performs only the session-owned final state transition.
// The caller keeps the row lock and owns the transaction containing endpoint registration.
func (repository *Repository) consumePairingAttempt(
	ctx context.Context,
	tx pgx.Tx,
	candidate pairingCompletionCandidate,
) error {
	if ctx == nil || repository == nil || repository.operationMax <= 0 || tx == nil {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	locked := candidate.locked
	if candidate.alreadyConsumed || candidate.consumedAt.IsZero() ||
		locked.attempt.state != pairingAttemptStateConfirming ||
		!locked.hasMobileBinding || !locked.hasAgentBinding ||
		subtle.ConstantTimeCompare(locked.mobileBinding[:], locked.agentBinding[:]) != 1 ||
		candidate.consumedAt.Before(locked.attempt.updatedAt) ||
		!locked.attempt.expiresAt.After(candidate.consumedAt) {
		return ErrPairingAttemptUnavailable
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()
	result, err := tx.Exec(operationCtx, `
		UPDATE session.pairing_attempts
		SET state = 'consumed', consumed_at = $1, updated_at = $1
		WHERE id = $2
		  AND state = 'confirming' AND updated_at = $3 AND expires_at > $1
		  AND failed_attempt_count = $4 AND failed_attempt_count < $5
		  AND claimed_user_id = $6
		  AND mobile_channel_binding = $7 AND agent_channel_binding = $7
		  AND mobile_confirmed_at IS NOT NULL AND agent_confirmed_at IS NOT NULL
		  AND consumed_at IS NULL`,
		candidate.consumedAt, locked.attempt.id.String(), locked.attempt.updatedAt,
		locked.attempt.failedAttemptCount, maxPairingAttemptFailures,
		locked.claim.ownerID.String(), locked.mobileBinding[:],
	)
	if err != nil {
		return pairingAttemptPersistenceError("consume", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPairingAttemptUnavailable
	}
	return nil
}

type pairingCompletionService struct {
	transactions transactionStarter
	attempts     pairingCompletionStore
	devices      pairedDeviceRegistrar
	agents       pairedAgentRegistrar
	operationMax time.Duration
}

func newPairingCompletionService(
	transactions transactionStarter,
	attempts pairingCompletionStore,
	devices pairedDeviceRegistrar,
	agents pairedAgentRegistrar,
) (*pairingCompletionService, error) {
	if transactions == nil || attempts == nil || devices == nil || agents == nil {
		return nil, errors.New("pairing completion service repositories are required")
	}
	return &pairingCompletionService{
		transactions: transactions,
		attempts:     attempts,
		devices:      devices,
		agents:       agents,
		operationMax: serviceOperationTimeout,
	}, nil
}

// complete registers both module-owned identities and consumes the attempt in
// one transaction. The agent must re-present the exact attempt-bound bootstrap
// credential. A commit error is unknown and is reconciled with the same inputs.
func (service *pairingCompletionService) complete(
	ctx context.Context,
	encodedID string,
	credential []byte,
) (completion PairingCompletion, err error) {
	if ctx == nil || service == nil || service.transactions == nil || service.attempts == nil ||
		service.devices == nil || service.agents == nil || service.operationMax <= 0 {
		return PairingCompletion{}, ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return PairingCompletion{}, err
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, service.operationMax)
	defer cancelOperation()
	tx, beginErr := service.transactions.Begin(operationCtx)
	if tx == nil {
		return PairingCompletion{}, pairingAttemptServiceError("begin completion", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(
			context.WithoutCancel(ctx), serviceCleanupTimeout,
		)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = pairingAttemptServiceError("rollback completion", rollbackErr)
		if err == nil {
			completion = PairingCompletion{}
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return PairingCompletion{}, pairingAttemptServiceError("begin completion", beginErr)
	}

	candidate, lockErr := service.attempts.lockPairingCompletion(
		operationCtx, tx, encodedID, credential,
	)
	if lockErr != nil {
		if errors.Is(lockErr, ErrPairingAttemptUnavailable) {
			return PairingCompletion{}, ErrPairingAttemptUnavailable
		}
		return PairingCompletion{}, pairingAttemptServiceError("lock completion", lockErr)
	}
	locked := candidate.locked
	completion = PairingCompletion{
		PairingID:  locked.attempt.id.String(),
		DeviceID:   locked.claim.deviceID.String(),
		AgentID:    locked.attempt.agentID.String(),
		ConsumedAt: candidate.consumedAt,
	}
	if !candidate.alreadyConsumed {
		if registerErr := service.devices.RegisterPaired(
			operationCtx, tx, locked.claim.ownerID, completion.DeviceID,
			locked.claim.deviceDisplayName, locked.claim.devicePlatform,
			locked.claim.devicePublicKey, candidate.consumedAt,
		); registerErr != nil {
			return PairingCompletion{}, pairingRegistrationError("register device", registerErr)
		}
		if registerErr := service.agents.RegisterPairedAgent(
			operationCtx, tx, locked.claim.ownerID, completion.AgentID,
			locked.attempt.agentDisplayName, locked.attempt.agentPublicKey,
			locked.attempt.agentVersion, candidate.consumedAt,
		); registerErr != nil {
			return PairingCompletion{}, pairingRegistrationError("register agent", registerErr)
		}
		if consumeErr := service.attempts.consumePairingAttempt(
			operationCtx, tx, candidate,
		); consumeErr != nil {
			if errors.Is(consumeErr, ErrPairingAttemptUnavailable) {
				return PairingCompletion{}, ErrPairingAttemptUnavailable
			}
			return PairingCompletion{}, pairingAttemptServiceError("consume", consumeErr)
		}
	}
	if commitErr := tx.Commit(operationCtx); commitErr != nil {
		return PairingCompletion{}, fmt.Errorf("%w: %w", ErrPairingAttemptCommitOutcomeUnknown, commitErr)
	}
	return completion, nil
}

func pairingRegistrationError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if errors.Is(err, device.ErrDeviceAccessDenied) || errors.Is(err, device.ErrInvalidDevice) ||
		errors.Is(err, workspace.ErrAgentAccessDenied) || errors.Is(err, workspace.ErrInvalidAgent) {
		return ErrPairingAttemptUnavailable
	}
	return fmt.Errorf("%w: %s: %w", ErrPairingAttemptPersistenceUnavailable, operation, err)
}
