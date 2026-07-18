package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidEvent       = errors.New("invalid outbox event")
	ErrEventAlreadyExists = errors.New("outbox event already exists")
	ErrEnqueueUnavailable = errors.New("outbox enqueue unavailable")
)

type EventKind uint8

const (
	EventDeviceRevoked EventKind = iota + 1
	EventAgentRevoked
)

type Event struct {
	id            ids.ID
	eventType     string
	aggregateType string
	aggregateID   ids.ID
	availableAt   time.Time
}

func NewEvent(kind EventKind, aggregateID ids.ID, availableAt time.Time) (Event, error) {
	var eventType, aggregateType string
	switch kind {
	case EventDeviceRevoked:
		eventType = "device.revoked.v1"
		aggregateType = "device"
	case EventAgentRevoked:
		eventType = "agent.revoked.v1"
		aggregateType = "agent"
	default:
		return Event{}, fmt.Errorf("%w: kind", ErrInvalidEvent)
	}
	if aggregateID.String() == "" {
		return Event{}, fmt.Errorf("%w: aggregate id", ErrInvalidEvent)
	}
	if availableAt.IsZero() {
		return Event{}, fmt.Errorf("%w: available time", ErrInvalidEvent)
	}
	eventID, err := ids.New()
	if err != nil {
		return Event{}, fmt.Errorf("%w: generate event id: %w", ErrEnqueueUnavailable, err)
	}

	return Event{
		id:            eventID,
		eventType:     eventType,
		aggregateType: aggregateType,
		aggregateID:   aggregateID,
		availableAt:   availableAt.UTC(),
	}, nil
}

// Enqueue inserts metadata-only event identity inside the caller's existing transaction.
func Enqueue(ctx context.Context, tx pgx.Tx, event Event) error {
	if ctx == nil {
		return ErrEnqueueUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.id.String() == "" || event.eventType == "" || event.aggregateType == "" ||
		event.aggregateID.String() == "" || event.availableAt.IsZero() {
		return ErrInvalidEvent
	}
	if tx == nil {
		return ErrEnqueueUnavailable
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO outbox.events (id, event_type, aggregate_type, aggregate_id, payload, available_at)
		VALUES ($1, $2, $3, $4, '{}'::jsonb, $5)`,
		event.id.String(), event.eventType, event.aggregateType, event.aggregateID.String(), event.availableAt)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}

	var databaseErr *pgconn.PgError
	if errors.As(err, &databaseErr) && databaseErr.Code == "23505" &&
		databaseErr.ConstraintName == "events_pkey" {
		return ErrEventAlreadyExists
	}
	return fmt.Errorf("enqueue outbox event: %w", err)
}
