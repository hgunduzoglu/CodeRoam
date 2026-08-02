package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrPairingAttemptCommitOutcomeUnknown = errors.New("pairing attempt commit outcome unknown")

type pairingAttemptStore interface {
	authenticateOpenPairingAttempt(
		context.Context,
		pgx.Tx,
		string,
		[]byte,
	) (PairingAttempt, bool, error)
	claimOpenPairingAttempt(context.Context, pgx.Tx, string, PairingAttemptClaim) error
	confirmMobilePairingAttempt(context.Context, pgx.Tx, string, MobilePairingConfirmation) error
}

type pairingAttemptService struct {
	transactions transactionStarter
	attempts     pairingAttemptStore
	operationMax time.Duration
}

func newPairingAttemptService(
	transactions transactionStarter,
	attempts pairingAttemptStore,
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

// claim binds one authenticated owner and mobile candidate in a single bounded
// transaction. A commit error is an unknown outcome and must be reconciled by
// retrying the exact same pairing ID and claim.
func (service *pairingAttemptService) claim(
	ctx context.Context,
	encodedID string,
	claim PairingAttemptClaim,
) (err error) {
	if ctx == nil || service == nil || service.transactions == nil || service.attempts == nil ||
		service.operationMax <= 0 {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !claim.valid() {
		return ErrInvalidPairingAttemptClaim
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, service.operationMax)
	defer cancelOperation()
	tx, beginErr := service.transactions.Begin(operationCtx)
	if tx == nil {
		return pairingAttemptServiceError("begin claim", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), serviceCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = pairingAttemptServiceError("rollback claim", rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return pairingAttemptServiceError("begin claim", beginErr)
	}

	if claimErr := service.attempts.claimOpenPairingAttempt(
		operationCtx, tx, encodedID, claim,
	); claimErr != nil {
		if errors.Is(claimErr, ErrPairingAttemptUnavailable) ||
			errors.Is(claimErr, ErrInvalidPairingAttemptClaim) {
			return claimErr
		}
		return pairingAttemptServiceError("claim", claimErr)
	}
	if commitErr := tx.Commit(operationCtx); commitErr != nil {
		return fmt.Errorf("%w: %w", ErrPairingAttemptCommitOutcomeUnknown, commitErr)
	}
	return nil
}

// confirmMobile records one authenticated mobile handshake observation inside a
// bounded transaction. A commit error must be reconciled with the exact same input.
func (service *pairingAttemptService) confirmMobile(
	ctx context.Context,
	encodedID string,
	confirmation MobilePairingConfirmation,
) (err error) {
	if ctx == nil || service == nil || service.transactions == nil || service.attempts == nil ||
		service.operationMax <= 0 {
		return ErrPairingAttemptPersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !confirmation.valid() {
		return ErrInvalidPairingAttemptConfirmation
	}

	operationCtx, cancelOperation := context.WithTimeout(ctx, service.operationMax)
	defer cancelOperation()
	tx, beginErr := service.transactions.Begin(operationCtx)
	if tx == nil {
		return pairingAttemptServiceError("begin mobile confirmation", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), serviceCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = pairingAttemptServiceError("rollback mobile confirmation", rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return pairingAttemptServiceError("begin mobile confirmation", beginErr)
	}

	if confirmationErr := service.attempts.confirmMobilePairingAttempt(
		operationCtx, tx, encodedID, confirmation,
	); confirmationErr != nil {
		if errors.Is(confirmationErr, ErrPairingAttemptUnavailable) ||
			errors.Is(confirmationErr, ErrInvalidPairingAttemptConfirmation) {
			return confirmationErr
		}
		return pairingAttemptServiceError("confirm mobile", confirmationErr)
	}
	if commitErr := tx.Commit(operationCtx); commitErr != nil {
		return fmt.Errorf("%w: %w", ErrPairingAttemptCommitOutcomeUnknown, commitErr)
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
