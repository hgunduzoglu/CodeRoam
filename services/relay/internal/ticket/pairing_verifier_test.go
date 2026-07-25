package ticket

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"google.golang.org/protobuf/proto"
)

func TestPairingTicketVerifierAcceptsCurrentAndPreviousKeys(t *testing.T) {
	currentPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x41}, ed25519.SeedSize))
	previousPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	verifier, err := NewPairingTicketVerifier(
		"eu-test-1",
		VerificationKey{ID: "current", PublicKey: currentPrivate.Public().(ed25519.PublicKey)},
		&PreviousVerificationKey{
			VerificationKey: VerificationKey{
				ID:        "previous",
				PublicKey: previousPrivate.Public().(ed25519.PublicKey),
			},
			IssuedThrough: time.Unix(1_800_000_000, 0),
		},
	)
	if err != nil {
		t.Fatalf("NewPairingTicketVerifier() error = %v", err)
	}
	now := time.Unix(1_800_000_000, 0)

	for keyID, privateKey := range map[string]ed25519.PrivateKey{
		"current":  currentPrivate,
		"previous": previousPrivate,
	} {
		t.Run(keyID, func(t *testing.T) {
			encoded := signedPairingTicket(t, validPairingClaims(now, keyID), privateKey)
			claims, err := verifier.VerifyPairingTicket(encoded, now)
			if err != nil {
				t.Fatalf("VerifyPairingTicket() error = %v", err)
			}
			if claims.GetKeyId() != keyID {
				t.Fatalf("verified key ID = %q, want %q", claims.GetKeyId(), keyID)
			}
		})
	}
}

func TestPairingTicketVerifierBoundsPreviousKeyIssuance(t *testing.T) {
	currentPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x41}, ed25519.SeedSize))
	previousPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	cutover := time.Unix(1_800_000_000, 0)
	verifier, err := NewPairingTicketVerifier(
		"eu-test-1",
		VerificationKey{ID: "current", PublicKey: currentPrivate.Public().(ed25519.PublicKey)},
		&PreviousVerificationKey{
			VerificationKey: VerificationKey{
				ID:        "previous",
				PublicKey: previousPrivate.Public().(ed25519.PublicKey),
			},
			IssuedThrough: cutover,
		},
	)
	if err != nil {
		t.Fatalf("NewPairingTicketVerifier() error = %v", err)
	}

	previousBefore := validPairingClaims(cutover.Add(-time.Second), "previous")
	previousBefore.ExpiresAtUnixSeconds = cutover.Add(30 * time.Second).Unix()
	if _, err := verifier.VerifyPairingTicket(
		signedPairingTicket(t, previousBefore, previousPrivate),
		cutover,
	); err != nil {
		t.Fatalf("previous ticket issued before cutoff error = %v", err)
	}

	afterCutover := cutover.Add(time.Second)
	previousAfter := validPairingClaims(afterCutover, "previous")
	if _, err := verifier.VerifyPairingTicket(
		signedPairingTicket(t, previousAfter, previousPrivate),
		afterCutover,
	); !errors.Is(err, ErrInvalidPairingTicket) {
		t.Fatalf("previous ticket issued after cutoff error = %v, want %v", err, ErrInvalidPairingTicket)
	}
	currentAfter := validPairingClaims(afterCutover, "current")
	if _, err := verifier.VerifyPairingTicket(
		signedPairingTicket(t, currentAfter, currentPrivate),
		afterCutover,
	); err != nil {
		t.Fatalf("current ticket issued after cutoff error = %v", err)
	}
}

func TestPairingTicketVerifierRejectsMalformedOrMutatedTickets(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x41}, ed25519.SeedSize))
	verifier, err := NewPairingTicketVerifier(
		"eu-test-1",
		VerificationKey{ID: "current", PublicKey: privateKey.Public().(ed25519.PublicKey)},
		nil,
	)
	if err != nil {
		t.Fatalf("NewPairingTicketVerifier() error = %v", err)
	}
	now := time.Unix(1_800_000_000, 0)
	valid := signedPairingTicket(t, validPairingClaims(now, "current"), privateKey)

	mutated := append([]byte(nil), valid...)
	mutated[len(mutated)-1] ^= 0x01
	tests := map[string][]byte{
		"empty":     nil,
		"truncated": valid[:len(valid)-1],
		"mutated":   mutated,
		"oversized": make([]byte, maxTicketSize+1),
	}
	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.VerifyPairingTicket(encoded, now); !errors.Is(err, ErrInvalidPairingTicket) {
				t.Fatalf("VerifyPairingTicket() error = %v, want %v", err, ErrInvalidPairingTicket)
			}
		})
	}
}

func TestPairingTicketVerifierRejectsInvalidClaims(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x41}, ed25519.SeedSize))
	verifier, err := NewPairingTicketVerifier(
		"eu-test-1",
		VerificationKey{ID: "current", PublicKey: privateKey.Public().(ed25519.PublicKey)},
		nil,
	)
	if err != nil {
		t.Fatalf("NewPairingTicketVerifier() error = %v", err)
	}
	now := time.Unix(1_800_000_000, 0)
	tests := map[string]func(*relayv1.ConnectionTicketClaims){
		"unknown key": func(claims *relayv1.ConnectionTicketClaims) {
			claims.KeyId = "unknown"
		},
		"session purpose": func(claims *relayv1.ConnectionTicketClaims) {
			claims.Purpose = relayv1.TicketPurpose_TICKET_PURPOSE_SESSION
		},
		"wrong region": func(claims *relayv1.ConnectionTicketClaims) {
			claims.RelayRegion = "other-region"
		},
		"unspecified role": func(claims *relayv1.ConnectionTicketClaims) {
			claims.Role = relayv1.EndpointRole_ENDPOINT_ROLE_UNSPECIFIED
		},
		"malformed endpoint ID": func(claims *relayv1.ConnectionTicketClaims) {
			claims.EndpointId = "device-1"
		},
		"short nonce": func(claims *relayv1.ConnectionTicketClaims) {
			claims.Nonce = []byte("short")
		},
		"not active": func(claims *relayv1.ConnectionTicketClaims) {
			claims.NotBeforeUnixSeconds = now.Add(6 * time.Second).Unix()
			claims.IssuedAtUnixSeconds = now.Add(6 * time.Second).Unix()
			claims.ExpiresAtUnixSeconds = now.Add(30 * time.Second).Unix()
		},
		"expired": func(claims *relayv1.ConnectionTicketClaims) {
			claims.NotBeforeUnixSeconds = now.Add(-time.Minute).Unix()
			claims.IssuedAtUnixSeconds = now.Add(-time.Minute).Unix()
			claims.ExpiresAtUnixSeconds = now.Add(-6 * time.Second).Unix()
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			claims := validPairingClaims(now, "current")
			mutate(claims)
			encoded := signedPairingTicket(t, claims, privateKey)
			if _, err := verifier.VerifyPairingTicket(encoded, now); !errors.Is(err, ErrInvalidPairingTicket) {
				t.Fatalf("VerifyPairingTicket() error = %v, want %v", err, ErrInvalidPairingTicket)
			}
		})
	}
}

func TestNewPairingTicketVerifierRejectsInvalidRotation(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x41}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	current := VerificationKey{ID: "same", PublicKey: publicKey}
	previous := PreviousVerificationKey{
		VerificationKey: VerificationKey{ID: "same", PublicKey: publicKey},
		IssuedThrough:   time.Unix(1_800_000_000, 0),
	}

	if _, err := NewPairingTicketVerifier("eu-test-1", current, &previous); !errors.Is(err, ErrInvalidPairingTicket) {
		t.Fatalf("NewPairingTicketVerifier() error = %v, want %v", err, ErrInvalidPairingTicket)
	}
}

func signedPairingTicket(
	t *testing.T,
	claims *relayv1.ConnectionTicketClaims,
	privateKey ed25519.PrivateKey,
) []byte {
	t.Helper()
	claimsBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(claims)
	if err != nil {
		t.Fatalf("encode claims: %v", err)
	}
	envelope := &relayv1.SignedConnectionTicket{
		Claims:    claimsBytes,
		Signature: ed25519.Sign(privateKey, claimsBytes),
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(envelope)
	if err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
	return encoded
}

func validPairingClaims(now time.Time, keyID string) *relayv1.ConnectionTicketClaims {
	return &relayv1.ConnectionTicketClaims{
		TicketId:             "11111111111111111111111111111111",
		RouteId:              "22222222222222222222222222222222",
		Role:                 relayv1.EndpointRole_ENDPOINT_ROLE_CLIENT,
		EndpointId:           "33333333333333333333333333333333",
		RelayRegion:          "eu-test-1",
		IssuedAtUnixSeconds:  now.Unix(),
		ExpiresAtUnixSeconds: now.Add(time.Minute).Unix(),
		Nonce:                bytes.Repeat([]byte{0x24}, pairingNonceSize),
		Purpose:              relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING,
		ProtocolVersion:      pairingProtocolVersion,
		NotBeforeUnixSeconds: now.Unix(),
		KeyId:                keyID,
	}
}
