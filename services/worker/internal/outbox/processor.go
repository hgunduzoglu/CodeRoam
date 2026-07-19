package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	processorOperationTimeout = 10 * time.Second
	processorHandlerTimeout   = 5 * time.Second
	processorCleanupTimeout   = 5 * time.Second
	maxDeliveryAttempts       = 5
)

var (
	ErrProcessingUnavailable = errors.New("outbox processing unavailable")
	ErrProcessOutcomeUnknown = errors.New("outbox processing outcome unknown")
	ErrPermanentHandler      = errors.New("permanent outbox handler failure")
)

type ProcessResult uint8

const (
	ProcessNoEvent ProcessResult = iota
	ProcessCompleted
	ProcessRetryScheduled
	ProcessDiscarded
)

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

type eventStore interface {
	Claim(context.Context, pgx.Tx, time.Time) (claimedEvent, bool, error)
	Finish(context.Context, pgx.Tx, claimedEvent, finishOutcome, time.Time) error
}

type Handler interface {
	Handle(context.Context, EventKind, string) error
}

type HandlerFunc func(context.Context, EventKind, string) error

func (handler HandlerFunc) Handle(ctx context.Context, kind EventKind, aggregateID string) error {
	return handler(ctx, kind, aggregateID)
}

type Processor struct {
	transactions transactionStarter
	store        eventStore
	handler      Handler
	now          func() time.Time
	operationMax time.Duration
	handlerMax   time.Duration
}

func NewProcessor(
	transactions transactionStarter,
	store eventStore,
	handler Handler,
	now func() time.Time,
) (*Processor, error) {
	if transactions == nil || store == nil || handler == nil {
		return nil, errors.New("outbox processor dependencies are required")
	}
	if now == nil {
		return nil, errors.New("outbox processor clock is required")
	}
	return &Processor{
		transactions: transactions,
		store:        store,
		handler:      handler,
		now:          now,
		operationMax: processorOperationTimeout,
		handlerMax:   processorHandlerTimeout,
	}, nil
}

func (processor *Processor) ProcessNext(
	ctx context.Context,
) (result ProcessResult, err error) {
	if ctx == nil || processor == nil || processor.transactions == nil || processor.store == nil ||
		processor.handler == nil || processor.now == nil || processor.operationMax <= 0 ||
		processor.handlerMax <= 0 || processor.handlerMax >= processor.operationMax {
		return ProcessNoEvent, ErrProcessingUnavailable
	}
	if err := ctx.Err(); err != nil {
		return ProcessNoEvent, err
	}
	claimedAt := processor.now().UTC()
	if claimedAt.IsZero() {
		return ProcessNoEvent, ErrProcessingUnavailable
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, processor.operationMax)
	defer cancelOperation()
	tx, beginErr := processor.transactions.Begin(operationCtx)
	if tx == nil {
		return ProcessNoEvent, processingError("begin", beginErr)
	}
	defer func() {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), processorCleanupTimeout)
		defer cancelRollback()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = processingError("rollback", rollbackErr)
		if err == nil {
			result = ProcessNoEvent
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()
	if beginErr != nil {
		return ProcessNoEvent, processingError("begin", beginErr)
	}

	claim, ok, err := processor.store.Claim(operationCtx, tx, claimedAt)
	if err != nil {
		return ProcessNoEvent, processingError("claim", err)
	}
	if !ok {
		return ProcessNoEvent, nil
	}

	outcome := finishCompleted
	result = ProcessCompleted
	if claim.invalid {
		outcome = finishDiscardInvalid
		result = ProcessDiscarded
	} else if claim.event.attemptCount >= maxDeliveryAttempts {
		outcome = finishDiscardExhausted
		result = ProcessDiscarded
	} else {
		handlerCtx, cancelHandler := context.WithTimeout(operationCtx, processor.handlerMax)
		handlerErr := processor.handler.Handle(
			handlerCtx, claim.event.kind, claim.event.aggregateID.String(),
		)
		handlerContextErr := handlerCtx.Err()
		cancelHandler()
		if err := ctx.Err(); err != nil {
			return ProcessNoEvent, err
		}
		if handlerErr == nil && handlerContextErr != nil {
			handlerErr = handlerContextErr
		}
		if handlerErr != nil {
			switch {
			case errors.Is(handlerErr, ErrPermanentHandler):
				outcome = finishDiscardPermanent
				result = ProcessDiscarded
			case claim.event.attemptCount+1 >= maxDeliveryAttempts:
				outcome = finishDiscardExhausted
				result = ProcessDiscarded
			default:
				outcome = finishRetry
				result = ProcessRetryScheduled
			}
		}
	}
	finishedAt := processor.now().UTC()
	if finishedAt.IsZero() || finishedAt.Before(claimedAt) {
		return ProcessNoEvent, ErrProcessingUnavailable
	}
	if err := processor.store.Finish(operationCtx, tx, claim, outcome, finishedAt); err != nil {
		return ProcessNoEvent, processingError("finish", err)
	}
	if err := tx.Commit(operationCtx); err != nil {
		return ProcessNoEvent, ErrProcessOutcomeUnknown
	}
	return result, nil
}

func processingError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if err == nil {
		return fmt.Errorf("%w: %s returned no transaction", ErrProcessingUnavailable, operation)
	}
	return fmt.Errorf("%w: %s: %w", ErrProcessingUnavailable, operation, err)
}
