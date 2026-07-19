package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

const maxRelayRegionBytes = 64

var (
	ErrInvalidSession      = errors.New("invalid session")
	ErrSessionAccessDenied = errors.New("session access denied")
)

type Session struct {
	id          ids.ID
	ownerID     auth.UserID
	deviceID    ids.ID
	agentID     ids.ID
	projectID   ids.ID
	relayRegion string
	startedAt   time.Time
}

func NewSession(
	actor auth.Actor,
	encodedID string,
	encodedDeviceID string,
	encodedAgentID string,
	encodedProjectID string,
	relayRegion string,
	startedAt time.Time,
) (Session, error) {
	ownerID, ok := actor.UserID()
	if !ok {
		return Session{}, ErrSessionAccessDenied
	}
	return newSession(
		ownerID, encodedID, encodedDeviceID, encodedAgentID, encodedProjectID, relayRegion, startedAt,
	)
}

func newSession(
	ownerID auth.UserID,
	encodedID string,
	encodedDeviceID string,
	encodedAgentID string,
	encodedProjectID string,
	relayRegion string,
	startedAt time.Time,
) (Session, error) {
	if ownerID.String() == "" {
		return Session{}, ErrSessionAccessDenied
	}
	sessionID, err := ids.Parse(encodedID)
	if err != nil {
		return Session{}, fmt.Errorf("%w: id", ErrInvalidSession)
	}
	deviceID, err := ids.Parse(encodedDeviceID)
	if err != nil {
		return Session{}, fmt.Errorf("%w: device id", ErrInvalidSession)
	}
	agentID, err := ids.Parse(encodedAgentID)
	if err != nil {
		return Session{}, fmt.Errorf("%w: agent id", ErrInvalidSession)
	}
	projectID, err := ids.Parse(encodedProjectID)
	if err != nil {
		return Session{}, fmt.Errorf("%w: project id", ErrInvalidSession)
	}
	if !validRelayRegion(relayRegion) {
		return Session{}, fmt.Errorf("%w: relay region", ErrInvalidSession)
	}
	if startedAt.IsZero() {
		return Session{}, fmt.Errorf("%w: start time", ErrInvalidSession)
	}

	return Session{
		id:          sessionID,
		ownerID:     ownerID,
		deviceID:    deviceID,
		agentID:     agentID,
		projectID:   projectID,
		relayRegion: relayRegion,
		startedAt:   startedAt.UTC(),
	}, nil
}

func (session Session) OwnedBy(actor auth.Actor) bool {
	actorID, ok := actor.UserID()
	return ok && session.ownerID.String() != "" && session.ownerID == actorID
}

func validRelayRegion(value string) bool {
	if len(value) == 0 || len(value) > maxRelayRegionBytes || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for index := range len(value) {
		character := value[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}
