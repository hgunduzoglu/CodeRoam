package workspace

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestNewProject(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	environment := newWorkspaceTestEnvironment(t, owner)
	createdAt := time.Date(2026, time.July, 18, 18, 0, 0, 0, time.FixedZone("test", 3*60*60))

	project, err := NewProject(
		owner,
		"4123456789abcdef0123456789abcdef",
		environment,
		"  CodeRoam  ",
		"/srv/coderoam/project",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if project.id.String() != "4123456789abcdef0123456789abcdef" ||
		project.environmentID.String() != environment.id.String() || project.name != "CodeRoam" ||
		project.rootPath != "/srv/coderoam/project" {
		t.Fatal("NewProject() did not preserve validated metadata")
	}
	if !project.createdAt.Equal(createdAt) || project.createdAt.Location() != time.UTC {
		t.Fatalf("createdAt = %v", project.createdAt)
	}
	if !project.OwnedBy(owner) {
		t.Fatal("new project did not recognize its owner")
	}

	foreignActor := newWorkspaceTestActor(t, "3123456789abcdef0123456789abcdef", "foreign@example.com")
	foreignEnvironment := newWorkspaceTestEnvironment(t, foreignActor)
	tests := map[string]struct {
		actor       auth.Actor
		id          string
		environment Environment
		name        string
		rootPath    string
		createdAt   time.Time
		want        error
	}{
		"zero actor": {
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrProjectAccessDenied,
		},
		"zero environment": {
			actor:     owner,
			id:        "4123456789abcdef0123456789abcdef",
			name:      "CodeRoam",
			rootPath:  "/srv/coderoam/project",
			createdAt: createdAt,
			want:      ErrProjectAccessDenied,
		},
		"foreign environment": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: foreignEnvironment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrProjectAccessDenied,
		},
		"empty id": {
			actor:       owner,
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"uppercase id": {
			actor:       owner,
			id:          "4123456789ABCDEF0123456789ABCDEF",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"empty name": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"invalid name encoding": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        string([]byte{0xff}),
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"name with control character": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam\nforged",
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"oversized name": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        strings.Repeat("a", maxProjectNameRunes+1),
			rootPath:    "/srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"empty root path": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"invalid root path encoding": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    string([]byte{'/', 0xff}),
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"root path with control character": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam\nproject",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"relative root path": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "srv/coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"filesystem root": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"noncanonical traversal root": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/../secrets",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"noncanonical duplicate separator": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv//coderoam/project",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"noncanonical trailing separator": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project/",
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"oversized root path": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/" + strings.Repeat("a", maxProjectRootBytes),
			createdAt:   createdAt,
			want:        ErrInvalidProject,
		},
		"zero creation time": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project",
			want:        ErrInvalidProject,
		},
		"creation before environment": {
			actor:       owner,
			id:          "4123456789abcdef0123456789abcdef",
			environment: environment,
			name:        "CodeRoam",
			rootPath:    "/srv/coderoam/project",
			createdAt:   environment.createdAt.Add(-time.Nanosecond),
			want:        ErrInvalidProject,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			project, err := NewProject(
				test.actor,
				test.id,
				test.environment,
				test.name,
				test.rootPath,
				test.createdAt,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewProject() error = %v, want %v", err, test.want)
			}
			if project.OwnedBy(owner) {
				t.Fatal("invalid project recognized an owner")
			}
		})
	}
}

func TestProjectOwnershipIsSeparateFromAgentTrust(t *testing.T) {
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
	project, err := NewProject(
		owner,
		"4123456789abcdef0123456789abcdef",
		environment,
		"CodeRoam",
		"/srv/coderoam/project",
		environment.createdAt,
	)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if project.OwnedBy(auth.Actor{}) || project.OwnedBy(foreignActor) {
		t.Fatal("project recognized a zero or foreign actor")
	}
	if err := agent.Revoke(owner, project.createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if agent.CanAuthorize(owner) {
		t.Fatal("revoked agent remained active")
	}
	if !project.OwnedBy(owner) {
		t.Fatal("agent revocation incorrectly changed stable project ownership")
	}
}

func newWorkspaceTestEnvironment(t *testing.T, owner auth.Actor) Environment {
	t.Helper()
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
	return environment
}
