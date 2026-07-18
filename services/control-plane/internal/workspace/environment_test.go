package workspace

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestNewEnvironment(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	agent := newWorkspaceTestAgent(t, owner)
	createdAt := time.Date(2026, time.July, 18, 15, 0, 0, 0, time.FixedZone("test", 3*60*60))

	environment, err := NewEnvironment(
		owner,
		"2123456789abcdef0123456789abcdef",
		agent,
		"  Production workspace  ",
		"  self-hosted  ",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewEnvironment() error = %v", err)
	}
	if environment.id.String() != "2123456789abcdef0123456789abcdef" ||
		environment.agentID.String() != agent.id.String() || environment.name != "Production workspace" ||
		environment.provider != "self-hosted" {
		t.Fatal("NewEnvironment() did not preserve validated metadata")
	}
	if !environment.createdAt.Equal(createdAt) || environment.createdAt.Location() != time.UTC {
		t.Fatalf("createdAt = %v", environment.createdAt)
	}
	if !environment.OwnedBy(owner) {
		t.Fatal("new environment did not recognize its owner")
	}

	foreignActor := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	foreignAgent := newWorkspaceTestAgent(t, foreignActor)
	revokedAgent := newWorkspaceTestAgent(t, owner)
	if err := revokedAgent.Revoke(owner, revokedAgent.createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	tests := map[string]struct {
		actor     auth.Actor
		id        string
		agent     Agent
		name      string
		provider  string
		createdAt time.Time
		want      error
	}{
		"zero actor": {
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrEnvironmentAccessDenied,
		},
		"zero agent": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrEnvironmentAccessDenied,
		},
		"foreign agent": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     foreignAgent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrEnvironmentAccessDenied,
		},
		"revoked agent": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     revokedAgent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrEnvironmentAccessDenied,
		},
		"empty id": {
			actor:     owner,
			agent:     agent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"uppercase id": {
			actor:     owner,
			id:        "2123456789ABCDEF0123456789ABCDEF",
			agent:     agent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"non hexadecimal id": {
			actor:     owner,
			id:        "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			agent:     agent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"empty name": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"invalid name encoding": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      string([]byte{0xff}),
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"name with control character": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace\nName",
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"oversized name": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      strings.Repeat("a", maxEnvironmentNameRunes+1),
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"oversized blank name": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      strings.Repeat(" ", maxEnvironmentNameBytes+1),
			provider:  "self-hosted",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"empty provider": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"invalid provider encoding": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace",
			provider:  string([]byte{0xff}),
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"provider with control character": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace",
			provider:  "self-hosted\nforged",
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"oversized provider": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace",
			provider:  strings.Repeat("a", maxEnvironmentProviderBytes+1),
			createdAt: createdAt,
			want:      ErrInvalidEnvironment,
		},
		"zero creation time": {
			actor:    owner,
			id:       "2123456789abcdef0123456789abcdef",
			agent:    agent,
			name:     "Workspace",
			provider: "self-hosted",
			want:     ErrInvalidEnvironment,
		},
		"creation before agent": {
			actor:     owner,
			id:        "2123456789abcdef0123456789abcdef",
			agent:     agent,
			name:      "Workspace",
			provider:  "self-hosted",
			createdAt: agent.createdAt.Add(-time.Nanosecond),
			want:      ErrInvalidEnvironment,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			environment, err := NewEnvironment(
				test.actor,
				test.id,
				test.agent,
				test.name,
				test.provider,
				test.createdAt,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewEnvironment() error = %v, want %v", err, test.want)
			}
			if environment.OwnedBy(owner) {
				t.Fatal("invalid environment recognized an owner")
			}
		})
	}
}

func TestEnvironmentOwnershipIsSeparateFromAgentTrust(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	agent := newWorkspaceTestAgent(t, owner)
	environment, err := NewEnvironment(
		owner,
		"2123456789abcdef0123456789abcdef",
		agent,
		"Workspace",
		"self-hosted",
		agent.createdAt.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewEnvironment() error = %v", err)
	}
	if environment.OwnedBy(auth.Actor{}) || environment.OwnedBy(foreignActor) {
		t.Fatal("environment recognized a zero or foreign actor")
	}
	if err := agent.Revoke(owner, environment.createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if agent.CanAuthorize(owner) {
		t.Fatal("revoked agent remained active")
	}
	if !environment.OwnedBy(owner) {
		t.Fatal("agent revocation incorrectly changed stable environment ownership")
	}
}

func newWorkspaceTestAgent(t *testing.T, owner auth.Actor) Agent {
	t.Helper()
	agent, err := NewAgent(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Workstation",
		newWorkspaceTestPublicKey(t, 0x42),
		"0.1.0",
		time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	return agent
}
