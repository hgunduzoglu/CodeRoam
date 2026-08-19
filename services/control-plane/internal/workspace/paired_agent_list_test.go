package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestRepositoryListPairedAgentsRejectsInvalidBoundaries(t *testing.T) {
	owner := newWorkspaceTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	starter := &workspaceTransactionStarterStub{err: errors.New("database unavailable")}
	repository, err := NewRepository(starter, func() time.Time {
		return time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
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
		"zero actor":      {repository: repository, ctx: context.Background(), limit: 10, want: ErrAgentAccessDenied},
		"zero limit":      {repository: repository, ctx: context.Background(), actor: owner, want: ErrInvalidPairedAgentList},
		"oversized limit": {repository: repository, ctx: context.Background(), actor: owner, limit: 101, want: ErrInvalidPairedAgentList},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := test.repository.ListPairedAgents(
				test.ctx, test.actor, test.limit,
			); !errors.Is(err, test.want) {
				t.Fatalf("ListPairedAgents() error = %v, want %v", err, test.want)
			}
		})
	}
	if starter.beginCalls != 0 {
		t.Fatalf("invalid boundaries began %d transactions", starter.beginCalls)
	}
	if _, err := repository.ListPairedAgents(
		context.Background(), owner, 10,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("ListPairedAgents(begin failure) error = %v", err)
	}
	starter.err = nil
	if _, err := repository.ListPairedAgents(
		context.Background(), owner, 10,
	); !errors.Is(err, ErrWorkspacePersistenceUnavailable) {
		t.Fatalf("ListPairedAgents(nil transaction) error = %v", err)
	}
}

func TestPairedAgentSummaryValid(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 8, 0, 0, 0, time.UTC)
	lastSeenAt := createdAt.Add(time.Minute)
	revokedAt := createdAt.Add(2 * time.Minute)
	fingerprint, err := cryptox.FingerprintX25519PublicKey(newWorkspaceTestPublicKey(t, 0x42))
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey() error = %v", err)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		t.Fatalf("fingerprint.String() error = %v", err)
	}
	valid := PairedAgentSummary{
		ID: "1123456789abcdef0123456789abcdef", Name: "MacBook Agent", Version: "0.1.0",
		Fingerprint: encodedFingerprint, CreatedAt: createdAt,
		LastSeenAt: &lastSeenAt, RevokedAt: &revokedAt,
	}
	if !valid.Valid() {
		t.Fatal("valid paired agent summary was rejected")
	}
	tests := map[string]PairedAgentSummary{
		"invalid id":          func() PairedAgentSummary { value := valid; value.ID = "invalid"; return value }(),
		"control name":        func() PairedAgentSummary { value := valid; value.Name = "secret\nname"; return value }(),
		"empty version":       func() PairedAgentSummary { value := valid; value.Version = ""; return value }(),
		"invalid fingerprint": func() PairedAgentSummary { value := valid; value.Fingerprint = "invalid"; return value }(),
		"zero creation time":  func() PairedAgentSummary { value := valid; value.CreatedAt = time.Time{}; return value }(),
		"early last seen": func() PairedAgentSummary {
			value := valid
			early := createdAt.Add(-time.Second)
			value.LastSeenAt = &early
			return value
		}(),
		"early revocation": func() PairedAgentSummary {
			value := valid
			early := createdAt.Add(-time.Second)
			value.RevokedAt = &early
			return value
		}(),
	}
	for name, summary := range tests {
		t.Run(name, func(t *testing.T) {
			if summary.Valid() {
				t.Fatal("invalid paired agent summary was accepted")
			}
		})
	}
}
