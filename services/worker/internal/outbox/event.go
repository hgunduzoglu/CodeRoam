package outbox

import (
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
)

var ErrInvalidEvent = errors.New("invalid outbox event")

type EventKind uint8

const (
	EventDeviceRevoked EventKind = iota + 1
	EventAgentRevoked
)

type Event struct {
	id           ids.ID
	kind         EventKind
	aggregateID  ids.ID
	availableAt  time.Time
	attemptCount int
}

func parseEvent(
	encodedID string,
	eventType string,
	aggregateType string,
	encodedAggregateID string,
	payloadEmpty bool,
	availableAt time.Time,
	attemptCount int,
) (Event, error) {
	id, err := ids.Parse(encodedID)
	if err != nil {
		return Event{}, fmt.Errorf("%w: id", ErrInvalidEvent)
	}
	aggregateID, err := ids.Parse(encodedAggregateID)
	if err != nil {
		return Event{}, fmt.Errorf("%w: aggregate id", ErrInvalidEvent)
	}
	var kind EventKind
	switch {
	case eventType == "device.revoked.v1" && aggregateType == "device":
		kind = EventDeviceRevoked
	case eventType == "agent.revoked.v1" && aggregateType == "agent":
		kind = EventAgentRevoked
	default:
		return Event{}, fmt.Errorf("%w: kind", ErrInvalidEvent)
	}
	if !payloadEmpty {
		return Event{}, fmt.Errorf("%w: payload", ErrInvalidEvent)
	}
	if availableAt.IsZero() {
		return Event{}, fmt.Errorf("%w: available time", ErrInvalidEvent)
	}
	if attemptCount < 0 {
		return Event{}, fmt.Errorf("%w: attempt count", ErrInvalidEvent)
	}
	return Event{
		id:           id,
		kind:         kind,
		aggregateID:  aggregateID,
		availableAt:  availableAt.UTC(),
		attemptCount: attemptCount,
	}, nil
}
