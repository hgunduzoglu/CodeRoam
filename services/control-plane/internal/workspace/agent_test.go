package workspace

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

type agentVerifierStub struct {
	userID auth.UserID
}

func (verifier agentVerifierStub) Verify(context.Context, string) (auth.UserID, error) {
	return verifier.userID, nil
}

type agentUserFinderStub struct {
	user auth.User
}

func (finder agentUserFinderStub) FindByID(context.Context, auth.UserID) (auth.User, error) {
	return finder.user, nil
}

func TestNewAgent(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	publicKey := newWorkspaceTestPublicKey(t, 0x42)
	createdAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.FixedZone("test", 3*60*60))

	agent, err := NewAgent(
		owner,
		"1123456789abcdef0123456789abcdef",
		"  Home workstation  ",
		publicKey,
		"  0.1.0-dev  ",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent.id.String() != "1123456789abcdef0123456789abcdef" || agent.name != "Home workstation" ||
		agent.version != "0.1.0-dev" || !agent.publicKey.Equal(publicKey) {
		t.Fatal("NewAgent() did not preserve validated identity metadata")
	}
	if !agent.createdAt.Equal(createdAt) || agent.createdAt.Location() != time.UTC {
		t.Fatalf("createdAt = %v", agent.createdAt)
	}
	if !agent.CanAuthorize(owner) {
		t.Fatal("new agent did not authorize its owner")
	}

	tests := map[string]struct {
		actor     auth.Actor
		id        string
		name      string
		publicKey cryptox.X25519PublicKey
		version   string
		createdAt time.Time
		want      error
	}{
		"zero actor": {
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrAgentAccessDenied,
		},
		"empty id": {
			actor:     owner,
			name:      "Workstation",
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"uppercase id": {
			actor:     owner,
			id:        "1123456789ABCDEF0123456789ABCDEF",
			name:      "Workstation",
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"non hexadecimal id": {
			actor:     owner,
			id:        "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			name:      "Workstation",
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"empty name": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"invalid name encoding": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      string([]byte{0xff}),
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"name with control character": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation\nName",
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"oversized name": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      strings.Repeat("a", maxAgentNameRunes+1),
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"oversized blank name": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      strings.Repeat(" ", maxAgentNameBytes+1),
			publicKey: publicKey,
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"zero public key": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			version:   "0.1.0",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"empty version": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			publicKey: publicKey,
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"invalid version encoding": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			publicKey: publicKey,
			version:   string([]byte{0xff}),
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"version with control character": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			publicKey: publicKey,
			version:   "0.1.0\nforged",
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"oversized version": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			publicKey: publicKey,
			version:   strings.Repeat("a", maxAgentVersionBytes+1),
			createdAt: createdAt,
			want:      ErrInvalidAgent,
		},
		"zero creation time": {
			actor:     owner,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Workstation",
			publicKey: publicKey,
			version:   "0.1.0",
			want:      ErrInvalidAgent,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			agent, err := NewAgent(
				test.actor,
				test.id,
				test.name,
				test.publicKey,
				test.version,
				test.createdAt,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewAgent() error = %v, want %v", err, test.want)
			}
			if agent.CanAuthorize(owner) {
				t.Fatal("invalid agent authorized an actor")
			}
		})
	}
}

func TestAgentRevocationFailsClosed(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newWorkspaceTestActor(t, "2123456789abcdef0123456789abcdef", "foreign@example.com")
	createdAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	agent, err := NewAgent(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Workstation",
		newWorkspaceTestPublicKey(t, 0x42),
		"0.1.0",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	if agent.CanAuthorize(auth.Actor{}) || agent.CanAuthorize(foreignActor) {
		t.Fatal("agent authorized a zero or foreign actor")
	}
	if err := agent.Revoke(foreignActor, createdAt.Add(time.Hour)); !errors.Is(err, ErrAgentAccessDenied) {
		t.Fatalf("Revoke(foreign) error = %v, want ErrAgentAccessDenied", err)
	}
	if !agent.CanAuthorize(owner) || agent.revocation.revokedAt != nil {
		t.Fatal("foreign revocation changed agent state")
	}
	if err := agent.Revoke(owner, createdAt.Add(-time.Second)); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("Revoke(before creation) error = %v, want ErrInvalidAgent", err)
	}
	if !agent.CanAuthorize(owner) || agent.revocation.revokedAt != nil {
		t.Fatal("invalid revocation time changed agent state")
	}

	firstRevokedAt := createdAt.Add(time.Hour)
	if err := agent.Revoke(owner, firstRevokedAt); err != nil {
		t.Fatalf("Revoke(owner) error = %v", err)
	}
	if agent.CanAuthorize(owner) || agent.revocation.revokedAt == nil ||
		!agent.revocation.revokedAt.Equal(firstRevokedAt) {
		t.Fatal("revoked agent remained authorized or lost its revocation time")
	}
	if err := agent.Revoke(owner, firstRevokedAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke(owner repeated) error = %v", err)
	}
	if !agent.revocation.revokedAt.Equal(firstRevokedAt) {
		t.Fatal("repeated revocation replaced the original time")
	}

	var nilAgent *Agent
	if err := nilAgent.Revoke(owner, firstRevokedAt); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("nil Agent Revoke() error = %v, want ErrInvalidAgent", err)
	}
}

func TestAgentCopiesShareRevocationState(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	createdAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	agent, err := NewAgent(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Workstation",
		newWorkspaceTestPublicKey(t, 0x42),
		"0.1.0",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	retainedCopy := agent
	if err := agent.Revoke(owner, createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if retainedCopy.CanAuthorize(owner) {
		t.Fatal("agent copy retained authorization after revocation")
	}
}

func TestAgentAuthorizationIsRaceSafeDuringRevocation(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	createdAt := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	agent, err := NewAgent(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Workstation",
		newWorkspaceTestPublicKey(t, 0x42),
		"0.1.0",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	retainedCopy := agent
	start := make(chan struct{})
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for range 100 {
				_ = retainedCopy.CanAuthorize(owner)
			}
		}()
	}
	close(start)
	if err := agent.Revoke(owner, createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	readers.Wait()
	if retainedCopy.CanAuthorize(owner) {
		t.Fatal("agent copy authorized after concurrent revocation")
	}
}

func newWorkspaceTestActor(t *testing.T, encodedID, email string) auth.Actor {
	t.Helper()
	user, err := auth.NewUser(
		encodedID,
		email,
		"Test User",
		time.Date(2026, time.July, 18, 9, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	userID, err := auth.ParseUserID(encodedID)
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	service, err := auth.NewService(
		agentUserFinderStub{user: user},
		agentVerifierStub{userID: userID},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	actor, err := service.Authenticate(context.Background(), "test-evidence")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	return actor
}

func newWorkspaceTestPublicKey(t *testing.T, value byte) cryptox.X25519PublicKey {
	t.Helper()
	key, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{value}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	return key
}
