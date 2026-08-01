package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrPairingAttemptCommitOutcomeUnknown = errors.New("pairing attempt commit outcome unknown")

type pairingAttemptCredentialStore interface {
	authenticateOpenPairingAttempt(
		context.Context,
		pgx.Tx,
		string,
		[]byte,
	) (PairingAttempt, bool, error)
}

type pairingAttemptService struct {
	transactions transactionStarter
	attempts     pairingAttemptCredentialStore
	operationMax time.Duration
}

func newPairingAttemptService(
	transactions transactionStarter,
	attempts pairingAttemptCredentialStore,
) (*pairingAttemptService, error) {
	if transactions == nil || attempts == nil {
		return nil, errors.New("pairing attempt service repositories are required")
	}
	return &pairingAttemptService{
		transactions: transactions,
		attempts:     attempts,
		operationMax: serviceOperationTimeout,
	}, nil
}

// authenticateBootstrapCredential commits a recorded rejection before returning the same
// unavailable result used by missing, expired, malformed, and exhausted attempts. It is not an
// authorization capability: successful verification remains package-private until the confirmation
// slice can mutate attempt state under this same row lock and transaction.
func (service *pairingAttemptService) authenticateBootstrapCredential(
	ctx context.Context,
	encodedID string,
	credential []byte,
) (err error) {
	if ctx == nil || service == nil || service.transactions == nil || service.attempts == nil ||
		service.operationMax <= 0 {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, service.operationMax)
	defer cancelOperation()
	tx, beginErr := service.transactions.Begin(operationCtx)
	if tx == nil {
		return pairingAttemptServiceError("begin authentication", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), serviceCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = pairingAttemptServiceError("rollback authentication", rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return pairingAttemptServiceError("begin authentication", beginErr)
	}

	_, matched, authenticateErr := service.attempts.authenticateOpenPairingAttempt(
		operationCtx, tx, encodedID, credential,
	)
	if authenticateErr != nil {
		if errors.Is(authenticateErr, ErrPairingAttemptUnavailable) {
			return ErrPairingAttemptUnavailable
		}
		return pairingAttemptServiceError("authenticate", authenticateErr)
	}

	commitCtx := context.Context(operationCtx)
	cancelCommit := func() {}
	if !matched {
		commitCtx, cancelCommit = context.WithTimeout(context.WithoutCancel(ctx), serviceCleanupTimeout)
	}
	defer cancelCommit()
	if commitErr := tx.Commit(commitCtx); commitErr != nil {
		return fmt.Errorf("%w: %w", ErrPairingAttemptCommitOutcomeUnknown, commitErr)
	}
	if !matched {
		return ErrPairingAttemptUnavailable
	}
	return nil
}

func pairingAttemptServiceError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if err == nil {
		return fmt.Errorf("%w: %s returned no transaction", ErrPairingAttemptPersistenceUnavailable, operation)
	}
	return fmt.Errorf("%w: %s: %w", ErrPairingAttemptPersistenceUnavailable, operation, err)
}
