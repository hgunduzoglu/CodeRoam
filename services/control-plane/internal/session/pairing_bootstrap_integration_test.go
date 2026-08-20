package session

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"google.golang.org/protobuf/proto"
)

func TestPairingBootstrapServiceIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)
	applySessionIntegrationMigration(t, ctx, pool)

	repository := NewRepository()
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	signer, err := NewPairingTicketSigner("m3-bootstrap-test", privateKey)
	if err != nil {
		t.Fatalf("NewPairingTicketSigner() error = %v", err)
	}
	now := time.Date(2026, time.August, 20, 13, 0, 0, 123456789, time.UTC)
	credential := bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen)
	nonce := bytes.Repeat([]byte{0x24}, pairingTicketNonceSize)

	t.Run("commits hash-only attempt and verifiable ticket", func(t *testing.T) {
		spec := validPairingBootstrapSpec(t, now)
		spec.PairingID = newPairingBootstrapIntegrationID(t)
		spec.AgentID = newPairingBootstrapIntegrationID(t)
		deletePairingAttemptFixture(t, pool, spec.PairingID)
		t.Cleanup(func() { deletePairingAttemptFixture(t, pool, spec.PairingID) })
		service, err := NewPairingBootstrapService(pool, repository, signer, "eu-test-1", func() time.Time { return now })
		if err != nil {
			t.Fatalf("NewPairingBootstrapService() error = %v", err)
		}
		service.random = bytes.NewReader(append(append([]byte(nil), credential...), nonce...))

		bootstrap, err := service.Start(ctx, spec)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		defer clear(bootstrap.BootstrapCredential)
		defer clear(bootstrap.AgentTicket)
		assertStoredPairingBootstrap(t, ctx, pool, spec, now, bootstrap.BootstrapCredential)
		assertVerifiableAgentBootstrapTicket(t, privateKey.Public().(ed25519.PublicKey), bootstrap.AgentTicket, spec)

		repository.now = func() time.Time { return now.Add(time.Minute) }
		attempts, err := newPairingAttemptService(pool, repository)
		if err != nil {
			t.Fatalf("newPairingAttemptService() error = %v", err)
		}
		if err := attempts.authenticateBootstrapCredential(ctx, spec.PairingID, bootstrap.BootstrapCredential); err != nil {
			t.Fatalf("authenticateBootstrapCredential(returned credential) error = %v", err)
		}
	})

	t.Run("withholds capabilities after ambiguous commit", func(t *testing.T) {
		spec := validPairingBootstrapSpec(t, now.Add(time.Minute))
		spec.PairingID = newPairingBootstrapIntegrationID(t)
		spec.AgentID = newPairingBootstrapIntegrationID(t)
		deletePairingAttemptFixture(t, pool, spec.PairingID)
		t.Cleanup(func() { deletePairingAttemptFixture(t, pool, spec.PairingID) })
		commitErr := errors.New("commit acknowledgement lost")
		service, err := NewPairingBootstrapService(
			&commitAcknowledgementLostStarter{pool: pool, err: commitErr},
			repository,
			signer,
			"eu-test-1",
			func() time.Time { return now.Add(time.Minute) },
		)
		if err != nil {
			t.Fatalf("NewPairingBootstrapService() error = %v", err)
		}
		service.random = bytes.NewReader(append(append([]byte(nil), credential...), nonce...))

		bootstrap, err := service.Start(ctx, spec)
		if !errors.Is(err, ErrPairingBootstrapCommitOutcomeUnknown) || !errors.Is(err, commitErr) {
			t.Fatalf("Start(ambiguous commit) error = %v", err)
		}
		assertEmptyPairingBootstrap(t, bootstrap)
		assertStoredPairingBootstrap(t, ctx, pool, spec, now.Add(time.Minute), credential)
	})
}

func assertStoredPairingBootstrap(
	t *testing.T,
	ctx context.Context,
	reader sessionRowReader,
	spec PairingBootstrapSpec,
	createdAt time.Time,
	credential []byte,
) {
	t.Helper()
	wantHash, err := HashPairingBootstrapCredential(spec.PairingID, credential)
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential() error = %v", err)
	}
	var agentID, state, relayRegion string
	var storedHash []byte
	var storedCreatedAt, storedExpiresAt time.Time
	if err := reader.QueryRow(ctx, `
		SELECT agent_id, state, relay_region, bootstrap_credential_hash, created_at, expires_at
		FROM session.pairing_attempts WHERE id = $1`, spec.PairingID,
	).Scan(&agentID, &state, &relayRegion, &storedHash, &storedCreatedAt, &storedExpiresAt); err != nil {
		t.Fatalf("read stored pairing bootstrap: %v", err)
	}
	if agentID != spec.AgentID || state != string(pairingAttemptStateOpen) || relayRegion != "eu-test-1" ||
		!bytes.Equal(storedHash, wantHash[:]) || bytes.Equal(storedHash, credential) ||
		!storedCreatedAt.Equal(createdAt.UTC().Truncate(time.Microsecond)) ||
		!storedExpiresAt.Equal(spec.ExpiresAt.UTC().Truncate(time.Microsecond)) {
		t.Fatal("stored pairing bootstrap did not preserve the bounded hash-only attempt")
	}
}

func assertVerifiableAgentBootstrapTicket(
	t *testing.T,
	publicKey ed25519.PublicKey,
	encoded []byte,
	spec PairingBootstrapSpec,
) {
	t.Helper()
	var envelope relayv1.SignedConnectionTicket
	if err := proto.Unmarshal(encoded, &envelope); err != nil {
		t.Fatalf("decode signed agent ticket: %v", err)
	}
	if !ed25519.Verify(publicKey, envelope.GetClaims(), envelope.GetSignature()) {
		t.Fatal("agent bootstrap ticket signature is invalid")
	}
	var claims relayv1.ConnectionTicketClaims
	if err := proto.Unmarshal(envelope.GetClaims(), &claims); err != nil {
		t.Fatalf("decode agent ticket claims: %v", err)
	}
	if claims.GetRouteId() != spec.PairingID || claims.GetEndpointId() != spec.AgentID ||
		claims.GetRole() != relayv1.EndpointRole_ENDPOINT_ROLE_AGENT ||
		claims.GetPurpose() != relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING ||
		claims.GetProtocolVersion() != pairingTicketProtocolVersion ||
		claims.GetRelayRegion() != "eu-test-1" {
		t.Fatal("signed agent ticket is not bound to the pairing attempt")
	}
}

func newPairingBootstrapIntegrationID(t *testing.T) string {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New() error = %v", err)
	}
	return id.String()
}
