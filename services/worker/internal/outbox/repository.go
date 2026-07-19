package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	repositoryOperationTimeout = 5 * time.Second
	repositoryRetryDelay       = time.Second
	maxPostgreSQLInteger       = 2147483647
)

var ErrRepositoryUnavailable = errors.New("outbox repository unavailable")

type finishOutcome uint8

const (
	finishCompleted finishOutcome = iota + 1
	finishRetry
	finishDiscardInvalid
	finishDiscardPermanent
	finishDiscardExhausted
)

type claimedEvent struct {
	locator string
	event   Event
	invalid bool
}

type Repository struct {
	operationMax time.Duration
	retryDelay   time.Duration
}

func NewRepository() *Repository {
	return &Repository{
		operationMax: repositoryOperationTimeout,
		retryDelay:   repositoryRetryDelay,
	}
}

func (repository *Repository) Claim(
	ctx context.Context,
	tx pgx.Tx,
	availableAt time.Time,
) (claimedEvent, bool, error) {
	if ctx == nil || repository == nil || repository.operationMax <= 0 {
		return claimedEvent{}, false, ErrRepositoryUnavailable
	}
	if err := ctx.Err(); err != nil {
		return claimedEvent{}, false, err
	}
	if tx == nil || availableAt.IsZero() {
		return claimedEvent{}, false, ErrRepositoryUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var locator string
	var encodedID, eventType, aggregateType, encodedAggregateID *string
	var payloadEmpty bool
	var storedAvailableAt time.Time
	var attemptCount int
	err := tx.QueryRow(operationCtx, `
		SELECT ctid::text,
		       CASE WHEN octet_length(id) = 32 THEN id END,
		       CASE WHEN octet_length(event_type) <= 64 THEN event_type END,
		       CASE WHEN octet_length(aggregate_type) <= 16 THEN aggregate_type END,
		       CASE WHEN octet_length(aggregate_id) = 32 THEN aggregate_id END,
		       payload = '{}'::jsonb,
		       available_at,
		       attempt_count
		FROM outbox.events
		WHERE processed_at IS NULL AND available_at <= $1
		ORDER BY available_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT 1`, availableAt.UTC()).Scan(
		&locator, &encodedID, &eventType, &aggregateType, &encodedAggregateID,
		&payloadEmpty, &storedAvailableAt, &attemptCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return claimedEvent{}, false, nil
	}
	if err != nil {
		return claimedEvent{}, false, repositoryError("claim", err)
	}
	claim := claimedEvent{locator: locator}
	if encodedID == nil || eventType == nil || aggregateType == nil || encodedAggregateID == nil {
		claim.invalid = true
		return claim, true, nil
	}
	event, err := parseEvent(
		*encodedID, *eventType, *aggregateType, *encodedAggregateID,
		payloadEmpty, storedAvailableAt, attemptCount,
	)
	if err != nil || event.availableAt.After(availableAt) {
		claim.invalid = true
		return claim, true, nil
	}
	claim.event = event
	return claim, true, nil
}

func (repository *Repository) Finish(
	ctx context.Context,
	tx pgx.Tx,
	claim claimedEvent,
	outcome finishOutcome,
	finishedAt time.Time,
) error {
	if ctx == nil || repository == nil || repository.operationMax <= 0 || repository.retryDelay <= 0 {
		return ErrRepositoryUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx == nil || claim.locator == "" || finishedAt.IsZero() {
		return ErrRepositoryUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, repository.operationMax)
	defer cancelOperation()

	var commandTag pgconn.CommandTag
	var err error
	switch outcome {
	case finishCompleted:
		commandTag, err = tx.Exec(operationCtx, `
			UPDATE outbox.events
			SET processed_at = $1,
			    attempt_count = CASE WHEN attempt_count < $2 THEN attempt_count + 1 ELSE attempt_count END,
			    last_error = NULL
			WHERE ctid = $3::tid AND processed_at IS NULL`,
			finishedAt.UTC(), maxPostgreSQLInteger, claim.locator,
		)
	case finishRetry:
		commandTag, err = tx.Exec(operationCtx, `
			UPDATE outbox.events
			SET available_at = $1,
			    attempt_count = CASE WHEN attempt_count < $2 THEN attempt_count + 1 ELSE attempt_count END,
			    last_error = 'retryable handler failure'
			WHERE ctid = $3::tid AND processed_at IS NULL`,
			finishedAt.UTC().Add(repository.retryDelay), maxPostgreSQLInteger, claim.locator,
		)
	case finishDiscardInvalid, finishDiscardPermanent, finishDiscardExhausted:
		lastError := "invalid event metadata"
		if outcome == finishDiscardPermanent {
			lastError = "permanent handler failure"
		}
		if outcome == finishDiscardExhausted {
			lastError = "delivery attempts exhausted"
		}
		commandTag, err = tx.Exec(operationCtx, `
			UPDATE outbox.events
			SET processed_at = $1,
			    attempt_count = CASE WHEN attempt_count < $2 THEN attempt_count + 1 ELSE attempt_count END,
			    last_error = $3
			WHERE ctid = $4::tid AND processed_at IS NULL`,
			finishedAt.UTC(), maxPostgreSQLInteger, lastError, claim.locator,
		)
	default:
		return ErrRepositoryUnavailable
	}
	if err != nil {
		return repositoryError("finish", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("%w: finish affected %d rows", ErrRepositoryUnavailable, commandTag.RowsAffected())
	}
	return nil
}

func repositoryError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("%w: %s: %w", ErrRepositoryUnavailable, operation, err)
}
