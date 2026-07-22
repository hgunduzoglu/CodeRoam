package outbox

import (
	"errors"
	"testing"
	"time"
)

func TestParseEvent(t *testing.T) {
	eventID := "0123456789abcdef0123456789abcdef"
	aggregateID := "1123456789abcdef0123456789abcdef"
	availableAt := time.Date(2026, time.July, 19, 18, 0, 0, 123, time.FixedZone("test", 2*60*60))
	event, err := parseEvent(
		eventID, "device.revoked.v1", "device", aggregateID, true, availableAt, 2,
	)
	if err != nil {
		t.Fatalf("parseEvent() error = %v", err)
	}
	if event.id.String() != eventID || event.kind != EventDeviceRevoked ||
		event.aggregateID.String() != aggregateID || !event.availableAt.Equal(availableAt) ||
		event.availableAt.Location() != time.UTC || event.attemptCount != 2 {
		t.Fatal("parseEvent() did not preserve normalized metadata")
	}

	tests := map[string]struct {
		eventID       string
		eventType     string
		aggregateType string
		aggregateID   string
		payloadEmpty  bool
		availableAt   time.Time
		attemptCount  int
		want          error
	}{
		"device revocation":      {eventID: eventID, eventType: "device.revoked.v1", aggregateType: "device", aggregateID: aggregateID, payloadEmpty: true, availableAt: availableAt},
		"agent revocation":       {eventID: eventID, eventType: "agent.revoked.v1", aggregateType: "agent", aggregateID: aggregateID, payloadEmpty: true, availableAt: availableAt},
		"invalid event ID":       {eventID: "invalid", eventType: "device.revoked.v1", aggregateType: "device", aggregateID: aggregateID, payloadEmpty: true, availableAt: availableAt, want: ErrInvalidEvent},
		"invalid aggregate ID":   {eventID: eventID, eventType: "device.revoked.v1", aggregateType: "device", aggregateID: "invalid", payloadEmpty: true, availableAt: availableAt, want: ErrInvalidEvent},
		"unknown event type":     {eventID: eventID, eventType: "unknown.v1", aggregateType: "device", aggregateID: aggregateID, payloadEmpty: true, availableAt: availableAt, want: ErrInvalidEvent},
		"mismatched aggregate":   {eventID: eventID, eventType: "device.revoked.v1", aggregateType: "agent", aggregateID: aggregateID, payloadEmpty: true, availableAt: availableAt, want: ErrInvalidEvent},
		"nonempty payload":       {eventID: eventID, eventType: "device.revoked.v1", aggregateType: "device", aggregateID: aggregateID, availableAt: availableAt, want: ErrInvalidEvent},
		"zero available time":    {eventID: eventID, eventType: "device.revoked.v1", aggregateType: "device", aggregateID: aggregateID, payloadEmpty: true, want: ErrInvalidEvent},
		"negative attempt count": {eventID: eventID, eventType: "device.revoked.v1", aggregateType: "device", aggregateID: aggregateID, payloadEmpty: true, availableAt: availableAt, attemptCount: -1, want: ErrInvalidEvent},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseEvent(
				test.eventID, test.eventType, test.aggregateType, test.aggregateID,
				test.payloadEmpty, test.availableAt, test.attemptCount,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("parseEvent() error = %v, want %v", err, test.want)
			}
		})
	}
}
