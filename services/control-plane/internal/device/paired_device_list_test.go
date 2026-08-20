package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestRepositoryListPairedDevicesRejectsInvalidBoundaries(t *testing.T) {
	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	starter := &transactionStarterStub{err: errors.New("database unavailable")}
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
		"nil repository":  {ctx: context.Background(), actor: owner, limit: 10, want: ErrDevicePersistenceUnavailable},
		"nil context":     {repository: repository, actor: owner, limit: 10, want: ErrDevicePersistenceUnavailable},
		"canceled":        {repository: repository, ctx: canceledCtx, actor: owner, limit: 10, want: context.Canceled},
		"zero actor":      {repository: repository, ctx: context.Background(), limit: 10, want: ErrDeviceAccessDenied},
		"zero limit":      {repository: repository, ctx: context.Background(), actor: owner, want: ErrInvalidPairedDeviceList},
		"oversized limit": {repository: repository, ctx: context.Background(), actor: owner, limit: 101, want: ErrInvalidPairedDeviceList},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := test.repository.ListPairedDevices(
				test.ctx, test.actor, test.limit,
			); !errors.Is(err, test.want) {
				t.Fatalf("ListPairedDevices() error = %v, want %v", err, test.want)
			}
		})
	}
	if starter.beginCalls != 0 {
		t.Fatalf("invalid boundaries began %d transactions", starter.beginCalls)
	}
	if _, err := repository.ListPairedDevices(
		context.Background(), owner, 10,
	); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("ListPairedDevices(begin failure) error = %v", err)
	}
	starter.err = nil
	if _, err := repository.ListPairedDevices(
		context.Background(), owner, 10,
	); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("ListPairedDevices(nil transaction) error = %v", err)
	}
}

func TestPairedDeviceSummaryValid(t *testing.T) {
	pairedAt := time.Date(2026, time.August, 20, 8, 0, 0, 0, time.UTC)
	lastSeenAt := pairedAt.Add(time.Minute)
	revokedAt := pairedAt.Add(2 * time.Minute)
	fingerprint, err := cryptox.FingerprintX25519PublicKey(newTestPublicKey(t, 0x42))
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey() error = %v", err)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		t.Fatalf("fingerprint.String() error = %v", err)
	}
	valid := PairedDeviceSummary{
		ID: "1123456789abcdef0123456789abcdef", Name: "Husam's iPhone",
		Platform: PlatformIOS, Fingerprint: encodedFingerprint, PairedAt: pairedAt,
		LastSeenAt: &lastSeenAt, RevokedAt: &revokedAt,
	}
	if !valid.Valid() {
		t.Fatal("valid paired device summary was rejected")
	}
	tests := map[string]PairedDeviceSummary{
		"invalid id":          func() PairedDeviceSummary { value := valid; value.ID = "invalid"; return value }(),
		"control name":        func() PairedDeviceSummary { value := valid; value.Name = "secret\nname"; return value }(),
		"invalid platform":    func() PairedDeviceSummary { value := valid; value.Platform = "desktop"; return value }(),
		"invalid fingerprint": func() PairedDeviceSummary { value := valid; value.Fingerprint = "invalid"; return value }(),
		"zero pairing time":   func() PairedDeviceSummary { value := valid; value.PairedAt = time.Time{}; return value }(),
		"early last seen": func() PairedDeviceSummary {
			value := valid
			early := pairedAt.Add(-time.Second)
			value.LastSeenAt = &early
			return value
		}(),
		"early revocation": func() PairedDeviceSummary {
			value := valid
			early := pairedAt.Add(-time.Second)
			value.RevokedAt = &early
			return value
		}(),
	}
	for name, summary := range tests {
		t.Run(name, func(t *testing.T) {
			if summary.Valid() {
				t.Fatal("invalid paired device summary was accepted")
			}
		})
	}
}
