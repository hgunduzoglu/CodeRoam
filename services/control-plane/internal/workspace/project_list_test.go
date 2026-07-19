package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestRepositoryListProjectsRejectsInvalidBoundaries(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	starter := &workspaceTransactionStarterStub{err: errors.New("database unavailable")}
	repository, err := NewRepository(starter, func() time.Time {
		return time.Date(2026, time.July, 19, 23, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	tests := map[string]struct {
		repository *Repository
		ctx        context.Context
		actor      auth.Actor
		limit      int
		want       error
	}{
		"nil repository":  {ctx: context.Background(), actor: owner, limit: 10, want: ErrWorkspacePersistenceUnavailable},
		"nil context":     {repository: repository, actor: owner, limit: 10, want: ErrWorkspacePersistenceUnavailable},
		"canceled":        {repository: repository, ctx: canceledCtx, actor: owner, limit: 10, want: context.Canceled},
		"zero actor":      {repository: repository, ctx: context.Background(), limit: 10, want: ErrProjectAccessDenied},
		"zero limit":      {repository: repository, ctx: context.Background(), actor: owner, want: ErrInvalidProjectList},
		"oversized limit": {repository: repository, ctx: context.Background(), actor: owner, limit: 101, want: ErrInvalidProjectList},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := test.repository.ListProjects(test.ctx, test.actor, test.limit); !errors.Is(err, test.want) {
				t.Fatalf("ListProjects() error = %v, want %v", err, test.want)
			}
		})
	}
	if starter.beginCalls != 0 {
		t.Fatalf("invalid boundaries began %d transactions", starter.beginCalls)
	}
	if _, err := repository.ListProjects(context.Background(), owner, 10); !errors.Is(
		err, ErrWorkspacePersistenceUnavailable,
	) {
		t.Fatalf("ListProjects(begin failure) error = %v", err)
	}
	starter.err = nil
	if _, err := repository.ListProjects(context.Background(), owner, 10); !errors.Is(
		err, ErrWorkspacePersistenceUnavailable,
	) {
		t.Fatalf("ListProjects(nil transaction) error = %v", err)
	}
}

func TestProjectSummaryValid(t *testing.T) {
	valid := ProjectSummary{
		ID: "1123456789abcdef0123456789abcdef", EnvironmentID: "2123456789abcdef0123456789abcdef",
		AgentID: "3123456789abcdef0123456789abcdef", Name: "CodeRoam",
		EnvironmentName: "Development", CreatedAt: time.Date(2026, time.July, 19, 23, 0, 0, 0, time.UTC),
	}
	if !valid.Valid() {
		t.Fatal("valid project summary was rejected")
	}
	tests := map[string]ProjectSummary{
		"invalid project id":       func() ProjectSummary { value := valid; value.ID = "invalid"; return value }(),
		"invalid environment id":   func() ProjectSummary { value := valid; value.EnvironmentID = "invalid"; return value }(),
		"invalid agent id":         func() ProjectSummary { value := valid; value.AgentID = "invalid"; return value }(),
		"control name":             func() ProjectSummary { value := valid; value.Name = "secret\nname"; return value }(),
		"unnormalized environment": func() ProjectSummary { value := valid; value.EnvironmentName = " Development "; return value }(),
		"zero creation":            func() ProjectSummary { value := valid; value.CreatedAt = time.Time{}; return value }(),
	}
	for name, summary := range tests {
		t.Run(name, func(t *testing.T) {
			if summary.Valid() {
				t.Fatal("invalid project summary was accepted")
			}
		})
	}
}
