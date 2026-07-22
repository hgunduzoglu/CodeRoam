package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/outbox"
	"github.com/jackc/pgx/v5"
)

const (
	repositoryOperationTimeout = 5 * time.Second
	transactionCleanupTimeout  = 5 * time.Second
)

var ErrDevicePersistenceUnavailable = errors.New("device persistence unavailable")

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

type enqueueEvent func(context.Context, pgx.Tx, outbox.Event) error

type Repository struct {
	transactions transactionStarter
	now          func() time.Time
	enqueue      enqueueEvent
	operationMax time.Duration
}

func NewRepository(transactions transactionStarter, now func() time.Time) (*Repository, error) {
	if transactions == nil {
		return nil, errors.New("device repository transaction starter is required")
	}
	if now == nil {
		return nil, errors.New("device repository clock is required")
	}
	return &Repository{
		transactions: transactions,
		now:          now,
		enqueue:      outbox.Enqueue,
		operationMax: repositoryOperationTimeout,
	}, nil
}

func (repository *Repository) Authorize(
	ctx context.Context,
	tx pgx.Tx,
	actor auth.Actor,
	encodedDeviceID string,
) error {
	if ctx == nil || repository == nil || repository.now == nil || repository.operationMax <= 0 {
		return ErrDevicePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	actorID, ok := actor.UserID()
	if !ok {
		return ErrDeviceAccessDenied
	}
	deviceID, err := ids.Parse(encodedDeviceID)
	if err != nil {
		return fmt.Errorf("%w: id", ErrInvalidDevice)
	}
	checkedAt := repository.now().UTC()
	if checkedAt.IsZero() {
		return ErrDevicePersistenceUnavailable
	}
	if tx == nil {
		return ErrDevicePersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var name, storedPlatform string
	var publicKeyBytes []byte
	var pairedAt time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT name, platform, static_public_key, paired_at
		FROM device.devices
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
		  AND octet_length(name) BETWEEN 1 AND $3
		  AND platform IN ('ios', 'ipados', 'android')
		  AND octet_length(static_public_key) = 32
		  AND paired_at <= $4
		FOR SHARE`,
		deviceID.String(), actorID.String(), maxDeviceNameBytes, checkedAt).Scan(
		&name, &storedPlatform, &publicKeyBytes, &pairedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDeviceAccessDenied
	}
	if err != nil {
		return persistenceError("authorize device", err)
	}
	publicKey, err := cryptox.ParseX25519PublicKey(publicKeyBytes)
	if err != nil {
		return ErrDeviceAccessDenied
	}
	storedDevice, err := NewDevice(actor, deviceID.String(), name, Platform(storedPlatform), publicKey, pairedAt)
	if err != nil || !storedDevice.CanAuthorize(actor) {
		return ErrDeviceAccessDenied
	}
	return nil
}

func (repository *Repository) Revoke(ctx context.Context, actor auth.Actor, encodedDeviceID string) (err error) {
	if ctx == nil || repository == nil || repository.transactions == nil || repository.now == nil ||
		repository.enqueue == nil || repository.operationMax <= 0 {
		return ErrDevicePersistenceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	actorID, ok := actor.UserID()
	if !ok {
		return ErrDeviceAccessDenied
	}
	deviceID, err := ids.Parse(encodedDeviceID)
	if err != nil {
		return fmt.Errorf("%w: id", ErrInvalidDevice)
	}
	revokedAt := repository.now().UTC()
	if revokedAt.IsZero() {
		return ErrDevicePersistenceUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	tx, err := repository.transactions.Begin(operationCtx)
	if err != nil {
		return persistenceError("begin revocation", err)
	}
	if tx == nil {
		return ErrDevicePersistenceUnavailable
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), transactionCleanupTimeout)
		defer cancel()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = persistenceError("rollback device revocation", rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()

	var pairedAt time.Time
	var storedRevokedAt *time.Time
	err = tx.QueryRow(operationCtx, `
		SELECT paired_at, revoked_at
		FROM device.devices
		WHERE id = $1 AND user_id = $2
		FOR UPDATE`, deviceID.String(), actorID.String()).Scan(&pairedAt, &storedRevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDeviceAccessDenied
	}
	if err != nil {
		return persistenceError("lock device for revocation", err)
	}
	if storedRevokedAt != nil {
		if err := tx.Commit(operationCtx); err != nil {
			return persistenceError("finish idempotent revocation", err)
		}
		return nil
	}
	if pairedAt.IsZero() || revokedAt.Before(pairedAt) {
		return fmt.Errorf("%w: stored pairing time", ErrInvalidDevice)
	}

	event, err := outbox.NewEvent(outbox.EventDeviceRevoked, deviceID, revokedAt)
	if err != nil {
		return persistenceError("create revocation event", err)
	}
	result, err := tx.Exec(operationCtx, `
		UPDATE device.devices
		SET revoked_at = $1
		WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL`,
		revokedAt, deviceID.String(), actorID.String())
	if err != nil {
		return persistenceError("persist device revocation", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("%w: revocation update affected %d rows", ErrDevicePersistenceUnavailable, result.RowsAffected())
	}
	if err := repository.enqueue(operationCtx, tx, event); err != nil {
		return persistenceError("enqueue device revocation", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return persistenceError("commit device revocation", err)
	}
	return nil
}

func persistenceError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("%w: %s: %w", ErrDevicePersistenceUnavailable, operation, err)
}
