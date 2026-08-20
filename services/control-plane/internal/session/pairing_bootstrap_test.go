package session

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

const pairingBootstrapTestTicketID = "5123456789abcdef0123456789abcdef"

type pairingAttemptCreatorStub struct {
	err     error
	calls   int
	tx      pgx.Tx
	attempt PairingAttempt
}

func (stub *pairingAttemptCreatorStub) CreatePairingAttempt(
	_ context.Context,
	tx pgx.Tx,
	attempt PairingAttempt,
) error {
	stub.calls++
	stub.tx = tx
	stub.attempt = attempt
	return stub.err
}

type pairingTicketIssuerStub struct {
	ticket []byte
	err    error
	calls  int
	claims *relayv1.ConnectionTicketClaims
}

func (stub *pairingTicketIssuerStub) SignPairingTicket(
	claims *relayv1.ConnectionTicketClaims,
) ([]byte, error) {
	stub.calls++
	stub.claims = proto.Clone(claims).(*relayv1.ConnectionTicketClaims)
	return append([]byte(nil), stub.ticket...), stub.err
}

func TestPairingBootstrapServiceCreatesHashOnlyAttemptAndAgentTicket(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 30, 0, 987654321, time.UTC)
	credential := bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen)
	nonce := bytes.Repeat([]byte{0x24}, pairingTicketNonceSize)
	randomInput := append(append([]byte(nil), credential...), nonce...)
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptCreatorStub{}
	tickets := &pairingTicketIssuerStub{ticket: []byte("signed-agent-ticket")}
	service := newPairingBootstrapServiceForTest(t, starter, store, tickets, now)
	service.random = bytes.NewReader(randomInput)
	service.newID = func() (ids.ID, error) { return ids.Parse(pairingBootstrapTestTicketID) }
	spec := validPairingBootstrapSpec(t, now)

	bootstrap, err := service.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer clear(bootstrap.BootstrapCredential)
	defer clear(bootstrap.AgentTicket)
	if bootstrap.PairingID != spec.PairingID || bootstrap.AgentID != spec.AgentID ||
		bootstrap.RelayRegion != "eu-test-1" || !bootstrap.ExpiresAt.Equal(spec.ExpiresAt) ||
		!bytes.Equal(bootstrap.BootstrapCredential, credential) ||
		!bytes.Equal(bootstrap.AgentTicket, tickets.ticket) {
		t.Fatal("Start() did not return the exact committed bootstrap result")
	}
	if starter.calls != 1 || store.calls != 1 || store.tx != tx ||
		tickets.calls != 1 || tx.commitCalls != 1 || tx.rollbackCalls != 1 {
		t.Fatalf(
			"calls = begin %d, store %d, sign %d, commit %d, rollback %d",
			starter.calls, store.calls, tickets.calls, tx.commitCalls, tx.rollbackCalls,
		)
	}

	wantHash, err := HashPairingBootstrapCredential(spec.PairingID, credential)
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential() error = %v", err)
	}
	if store.attempt.id.String() != spec.PairingID || store.attempt.agentID.String() != spec.AgentID ||
		store.attempt.bootstrapCredentialHash != wantHash ||
		!store.attempt.createdAt.Equal(now.Truncate(time.Microsecond)) ||
		!store.attempt.expiresAt.Equal(spec.ExpiresAt) {
		t.Fatal("Start() did not persist the exact hash-only pairing attempt")
	}
	claims := tickets.claims
	issuedAt := now.Truncate(time.Second)
	if claims.GetTicketId() != pairingBootstrapTestTicketID || claims.GetRouteId() != spec.PairingID ||
		claims.GetEndpointId() != spec.AgentID || claims.GetRelayRegion() != "eu-test-1" ||
		claims.GetRole() != relayv1.EndpointRole_ENDPOINT_ROLE_AGENT ||
		claims.GetPurpose() != relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING ||
		claims.GetProtocolVersion() != pairingTicketProtocolVersion ||
		claims.GetIssuedAtUnixSeconds() != issuedAt.Unix() ||
		claims.GetNotBeforeUnixSeconds() != issuedAt.Add(-pairingTicketClockSkew).Unix() ||
		claims.GetExpiresAtUnixSeconds() != issuedAt.Add(maxPairingTicketLifetime-pairingTicketClockSkew).Unix() ||
		!bytes.Equal(claims.GetNonce(), nonce) {
		t.Fatal("Start() did not bind the bounded agent ticket to the pairing attempt")
	}
}

func TestPairingBootstrapServiceTicketDoesNotOutliveAttempt(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 30, 0, 987654321, time.UTC)
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptCreatorStub{}
	tickets := &pairingTicketIssuerStub{ticket: []byte("signed-agent-ticket")}
	service := newPairingBootstrapServiceForTest(t, starter, store, tickets, now)
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen+pairingTicketNonceSize))
	service.newID = func() (ids.ID, error) { return ids.Parse(pairingBootstrapTestTicketID) }
	spec := validPairingBootstrapSpec(t, now)
	spec.ExpiresAt = now.Add(30 * time.Second)

	bootstrap, err := service.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer clear(bootstrap.BootstrapCredential)
	defer clear(bootstrap.AgentTicket)
	claimExpiry := time.Unix(tickets.claims.GetExpiresAtUnixSeconds(), 0)
	if effectiveExpiry, want := claimExpiry.Add(pairingTicketClockSkew), spec.ExpiresAt.Truncate(time.Second); !effectiveExpiry.Equal(want) {
		t.Fatalf("effective relay expiry = %s, want attempt expiry %s", effectiveExpiry, want)
	}
}

func TestPairingBootstrapServiceFailsClosedWithoutReturningCapabilities(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 30, 0, 0, time.UTC)
	databaseErr := errors.New("database unavailable")
	commitErr := errors.New("commit acknowledgement lost")
	signingErr := errors.New("signer unavailable")
	idErr := errors.New("random id unavailable")
	tests := map[string]struct {
		mutate  func(*PairingBootstrapService, *pairingAttemptCreatorStub, *pairingTicketIssuerStub, *sessionServiceTxStub, *PairingBootstrapSpec)
		want    error
		persist int
		commit  int
	}{
		"mismatched fingerprint": {
			mutate: func(_ *PairingBootstrapService, _ *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, spec *PairingBootstrapSpec) {
				spec.AgentFingerprint = "sha256:" + string(bytes.Repeat([]byte{'0'}, 64))
			},
			want: ErrInvalidPairingAttempt,
		},
		"short random input": {
			mutate: func(service *PairingBootstrapService, _ *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				service.random = bytes.NewReader([]byte{0x01})
			},
			want: ErrPairingBootstrapUnavailable,
		},
		"all-zero credential": {
			mutate: func(service *PairingBootstrapService, _ *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				service.random = bytes.NewReader(make([]byte, pairingBootstrapCredentialLen+pairingTicketNonceSize))
			},
			want: ErrPairingBootstrapUnavailable,
		},
		"expiry leaves no ticket lifetime": {
			mutate: func(_ *PairingBootstrapService, _ *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, spec *PairingBootstrapSpec) {
				spec.ExpiresAt = now.Add(pairingTicketClockSkew)
			},
			want: ErrInvalidPairingAttempt,
		},
		"ticket id failure": {
			mutate: func(service *PairingBootstrapService, _ *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				service.newID = func() (ids.ID, error) { return ids.ID{}, idErr }
			},
			want: ErrPairingBootstrapUnavailable,
		},
		"signing failure": {
			mutate: func(_ *PairingBootstrapService, _ *pairingAttemptCreatorStub, tickets *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				tickets.err = signingErr
			},
			want: ErrPairingBootstrapUnavailable,
		},
		"empty signed ticket": {
			mutate: func(_ *PairingBootstrapService, _ *pairingAttemptCreatorStub, tickets *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				tickets.ticket = nil
			},
			want: ErrPairingBootstrapUnavailable,
		},
		"oversized signed ticket": {
			mutate: func(_ *PairingBootstrapService, _ *pairingAttemptCreatorStub, tickets *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				tickets.ticket = make([]byte, maxPairingTicketEncodedSize+1)
			},
			want: ErrPairingBootstrapUnavailable,
		},
		"duplicate pairing id": {
			mutate: func(_ *PairingBootstrapService, store *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				store.err = ErrPairingAttemptAlreadyExists
			},
			want: ErrPairingAttemptAlreadyExists, persist: 1,
		},
		"persistence failure": {
			mutate: func(_ *PairingBootstrapService, store *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, _ *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				store.err = databaseErr
			},
			want: ErrPairingBootstrapUnavailable, persist: 1,
		},
		"ambiguous commit": {
			mutate: func(_ *PairingBootstrapService, _ *pairingAttemptCreatorStub, _ *pairingTicketIssuerStub, tx *sessionServiceTxStub, _ *PairingBootstrapSpec) {
				tx.commitErr = commitErr
			},
			want: ErrPairingBootstrapCommitOutcomeUnknown, persist: 1, commit: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := &sessionServiceTxStub{}
			starter := &sessionServiceStarterStub{tx: tx}
			store := &pairingAttemptCreatorStub{}
			tickets := &pairingTicketIssuerStub{ticket: []byte("signed-agent-ticket")}
			service := newPairingBootstrapServiceForTest(t, starter, store, tickets, now)
			service.random = bytes.NewReader(bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen+pairingTicketNonceSize))
			service.newID = func() (ids.ID, error) { return ids.Parse(pairingBootstrapTestTicketID) }
			spec := validPairingBootstrapSpec(t, now)
			test.mutate(service, store, tickets, tx, &spec)

			bootstrap, err := service.Start(context.Background(), spec)
			if !errors.Is(err, test.want) {
				t.Fatalf("Start() error = %v, want %v", err, test.want)
			}
			assertEmptyPairingBootstrap(t, bootstrap)
			if store.calls != test.persist || tx.commitCalls != test.commit {
				t.Fatalf("calls = persist %d, commit %d", store.calls, tx.commitCalls)
			}
		})
	}
}

func TestPairingBootstrapServiceRejectsInvalidBoundaries(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 30, 0, 0, time.UTC)
	tx := &sessionServiceTxStub{}
	starter := &sessionServiceStarterStub{tx: tx}
	store := &pairingAttemptCreatorStub{}
	tickets := &pairingTicketIssuerStub{}
	service := newPairingBootstrapServiceForTest(t, starter, store, tickets, now)
	spec := validPairingBootstrapSpec(t, now)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if result, err := service.Start(nil, spec); !errors.Is(err, ErrPairingBootstrapUnavailable) {
		t.Fatalf("Start(nil context) error = %v", err)
	} else {
		assertEmptyPairingBootstrap(t, result)
	}
	if result, err := service.Start(canceledCtx, spec); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start(canceled context) error = %v", err)
	} else {
		assertEmptyPairingBootstrap(t, result)
	}
	var nilService *PairingBootstrapService
	if result, err := nilService.Start(context.Background(), spec); !errors.Is(err, ErrPairingBootstrapUnavailable) {
		t.Fatalf("nil service Start() error = %v", err)
	} else {
		assertEmptyPairingBootstrap(t, result)
	}
	if starter.calls != 0 || store.calls != 0 || tickets.calls != 0 {
		t.Fatal("invalid bootstrap boundary reached signing or persistence")
	}

	if _, err := NewPairingBootstrapService(nil, store, tickets, "eu-test-1", time.Now); err == nil {
		t.Fatal("NewPairingBootstrapService(nil transactions) error = nil")
	}
	if _, err := NewPairingBootstrapService(starter, nil, tickets, "eu-test-1", time.Now); err == nil {
		t.Fatal("NewPairingBootstrapService(nil attempts) error = nil")
	}
	if _, err := NewPairingBootstrapService(starter, store, nil, "eu-test-1", time.Now); err == nil {
		t.Fatal("NewPairingBootstrapService(nil tickets) error = nil")
	}
	if _, err := NewPairingBootstrapService(starter, store, tickets, "invalid--region", time.Now); err == nil {
		t.Fatal("NewPairingBootstrapService(invalid region) error = nil")
	}
	if _, err := NewPairingBootstrapService(starter, store, tickets, "eu-test-1", nil); err == nil {
		t.Fatal("NewPairingBootstrapService(nil clock) error = nil")
	}
}

func validPairingBootstrapSpec(t *testing.T, now time.Time) PairingBootstrapSpec {
	t.Helper()
	publicKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(publicKey)
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey() error = %v", err)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		t.Fatalf("fingerprint.String() error = %v", err)
	}
	return PairingBootstrapSpec{
		PairingID: serviceTestSessionID, AgentID: serviceTestAgentID,
		AgentPublicKey: publicKey, AgentFingerprint: encodedFingerprint,
		AgentDisplayName: "M3 agent", AgentVersion: "0.1.0",
		ProtocolVersion: pairingAttemptProtocolVersion,
		ExpiresAt:       now.UTC().Truncate(time.Microsecond).Add(maxPairingAttemptLifetime),
	}
}

func newPairingBootstrapServiceForTest(
	t *testing.T,
	starter transactionStarter,
	store pairingAttemptCreator,
	tickets pairingTicketIssuer,
	now time.Time,
) *PairingBootstrapService {
	t.Helper()
	service, err := NewPairingBootstrapService(starter, store, tickets, "eu-test-1", func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewPairingBootstrapService() error = %v", err)
	}
	return service
}

func assertEmptyPairingBootstrap(t *testing.T, bootstrap PairingBootstrap) {
	t.Helper()
	if bootstrap.PairingID != "" || bootstrap.AgentID != "" || bootstrap.RelayRegion != "" ||
		!bootstrap.ExpiresAt.IsZero() || len(bootstrap.BootstrapCredential) != 0 ||
		len(bootstrap.AgentTicket) != 0 {
		t.Fatal("failed bootstrap returned a partial capability")
	}
}
