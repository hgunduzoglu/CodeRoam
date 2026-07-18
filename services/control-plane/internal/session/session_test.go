package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestNewSession(t *testing.T) {
	owner := newSessionTestActor(t, "0123456789abcdef0123456789abcdef")
	foreignActor := newSessionTestActor(t, "1123456789abcdef0123456789abcdef")
	startedAt := time.Date(2026, time.July, 19, 13, 0, 0, 123, time.FixedZone("test", 2*60*60))
	sessionID := "2123456789abcdef0123456789abcdef"
	deviceID := "3123456789abcdef0123456789abcdef"
	agentID := "4123456789abcdef0123456789abcdef"
	projectID := "5123456789abcdef0123456789abcdef"

	session, err := NewSession(owner, sessionID, deviceID, agentID, projectID, "eu-central-1", startedAt)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if session.id.String() != sessionID || session.deviceID.String() != deviceID ||
		session.agentID.String() != agentID || session.projectID.String() != projectID ||
		session.relayRegion != "eu-central-1" || !session.startedAt.Equal(startedAt) ||
		session.startedAt.Location() != time.UTC {
		t.Fatal("NewSession() did not preserve normalized metadata")
	}
	if !session.OwnedBy(owner) || session.OwnedBy(foreignActor) || session.OwnedBy(auth.Actor{}) {
		t.Fatal("Session.OwnedBy() did not enforce exact authenticated ownership")
	}

	tests := map[string]struct {
		actor     auth.Actor
		sessionID string
		deviceID  string
		agentID   string
		projectID string
		region    string
		startedAt time.Time
		want      error
	}{
		"valid metadata": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "eu-central-1", startedAt: startedAt, want: nil,
		},
		"missing actor": {
			actor: auth.Actor{}, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "eu-central-1", startedAt: startedAt, want: ErrSessionAccessDenied,
		},
		"invalid session id": {
			actor: owner, sessionID: "invalid", deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "eu-central-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"invalid device id": {
			actor: owner, sessionID: sessionID, deviceID: "invalid", agentID: agentID,
			projectID: projectID, region: "eu-central-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"invalid agent id": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: "invalid",
			projectID: projectID, region: "eu-central-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"invalid project id": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: "invalid", region: "eu-central-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"empty region": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "", startedAt: startedAt, want: ErrInvalidSession,
		},
		"oversized region": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: strings.Repeat("a", maxRelayRegionBytes+1),
			startedAt: startedAt, want: ErrInvalidSession,
		},
		"uppercase region": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "EU-central-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"leading separator": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "-eu-central-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"trailing separator": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "eu-central-1-", startedAt: startedAt, want: ErrInvalidSession,
		},
		"control character": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "eu\ncentral-1", startedAt: startedAt, want: ErrInvalidSession,
		},
		"zero start time": {
			actor: owner, sessionID: sessionID, deviceID: deviceID, agentID: agentID,
			projectID: projectID, region: "eu-central-1", want: ErrInvalidSession,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewSession(
				test.actor, test.sessionID, test.deviceID, test.agentID, test.projectID,
				test.region, test.startedAt,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewSession() error = %v, want %v", err, test.want)
			}
		})
	}
}

func newSessionTestActor(t *testing.T, encodedID string) auth.Actor {
	t.Helper()
	user, err := auth.NewUser(
		encodedID, "owner@example.com", "Session owner",
		time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("auth.NewUser() error = %v", err)
	}
	repository := &sessionUserFinderStub{user: user}
	verifier := &sessionIdentityVerifierStub{id: encodedID}
	service, err := auth.NewService(repository, verifier)
	if err != nil {
		t.Fatalf("auth.NewService() error = %v", err)
	}
	actor, err := service.Authenticate(t.Context(), "verified-evidence")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	return actor
}

type sessionUserFinderStub struct {
	user auth.User
}

func (finder *sessionUserFinderStub) FindByID(_ context.Context, _ auth.UserID) (auth.User, error) {
	return finder.user, nil
}

type sessionIdentityVerifierStub struct {
	id string
}

func (verifier *sessionIdentityVerifierStub) Verify(_ context.Context, _ string) (auth.UserID, error) {
	return auth.ParseUserID(verifier.id)
}
