package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/jackc/pgx/v5"
)

func TestNewEvent(t *testing.T) {
	availableAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.FixedZone("test", 3*60*60))
	aggregateID, err := ids.Parse("1123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ids.Parse() error = %v", err)
	}
	event, err := NewEvent(EventDeviceRevoked, aggregateID, availableAt)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	if _, err := ids.Parse(event.id.String()); err != nil {
		t.Fatalf("generated event id = %q: %v", event.id.String(), err)
	}
	if event.eventType != "device.revoked.v1" || event.aggregateType != "device" ||
		event.aggregateID != aggregateID {
		t.Fatal("NewEvent() did not preserve validated metadata")
	}
	if !event.availableAt.Equal(availableAt) || event.availableAt.Location() != time.UTC {
		t.Fatalf("availableAt = %v", event.availableAt)
	}
	agentEvent, err := NewEvent(EventAgentRevoked, aggregateID, availableAt)
	if err != nil {
		t.Fatalf("NewEvent(agent revoked) error = %v", err)
	}
	if agentEvent.eventType != "agent.revoked.v1" || agentEvent.aggregateType != "agent" ||
		agentEvent.aggregateID != aggregateID {
		t.Fatal("NewEvent(agent revoked) did not preserve validated metadata")
	}

	tests := map[string]struct {
		kind        EventKind
		aggregateID ids.ID
		availableAt time.Time
	}{
		"zero kind": {
			aggregateID: aggregateID,
			availableAt: availableAt,
		},
		"unknown kind": {
			kind:        EventKind(255),
			aggregateID: aggregateID,
			availableAt: availableAt,
		},
		"zero aggregate id": {
			kind:        EventDeviceRevoked,
			availableAt: availableAt,
		},
		"zero available time": {
			kind:        EventDeviceRevoked,
			aggregateID: aggregateID,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewEvent(test.kind, test.aggregateID, test.availableAt)
			if !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("NewEvent() error = %v, want ErrInvalidEvent", err)
			}
		})
	}
}

func TestEnqueueRejectsInvalidBoundaries(t *testing.T) {
	availableAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	aggregateID, err := ids.Parse("1123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ids.Parse() error = %v", err)
	}
	event, err := NewEvent(EventDeviceRevoked, aggregateID, availableAt)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if err := Enqueue(context.Background(), nil, Event{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Enqueue(zero event) error = %v, want ErrInvalidEvent", err)
	}
	if err := Enqueue(context.Background(), nil, event); !errors.Is(err, ErrEnqueueUnavailable) {
		t.Fatalf("Enqueue(nil transaction) error = %v, want ErrEnqueueUnavailable", err)
	}
	if err := Enqueue(nil, nil, event); !errors.Is(err, ErrEnqueueUnavailable) {
		t.Fatalf("Enqueue(nil context) error = %v, want ErrEnqueueUnavailable", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var transaction pgx.Tx
	if err := Enqueue(ctx, transaction, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("Enqueue(canceled context) error = %v, want context.Canceled", err)
	}
}
