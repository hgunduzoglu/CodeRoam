package session

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"google.golang.org/protobuf/proto"
)

func TestPairingTicketSignerProducesDeterministicVerifiableEnvelope(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	signer, err := NewPairingTicketSigner("m3-current", privateKey)
	if err != nil {
		t.Fatalf("NewPairingTicketSigner() error = %v", err)
	}
	claims := validPairingTicketClaims(time.Unix(1_800_000_000, 0))

	first, err := signer.SignPairingTicket(claims)
	if err != nil {
		t.Fatalf("SignPairingTicket() error = %v", err)
	}
	second, err := signer.SignPairingTicket(claims)
	if err != nil {
		t.Fatalf("SignPairingTicket() second error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("SignPairingTicket() is not deterministic")
	}
	if claims.GetKeyId() != "" {
		t.Fatal("SignPairingTicket() mutated caller claims")
	}

	var envelope relayv1.SignedConnectionTicket
	if err := proto.Unmarshal(first, &envelope); err != nil {
		t.Fatalf("decode signed ticket: %v", err)
	}
	if !ed25519.Verify(privateKey.Public().(ed25519.PublicKey), envelope.GetClaims(), envelope.GetSignature()) {
		t.Fatal("signed ticket signature is invalid")
	}
	var signedClaims relayv1.ConnectionTicketClaims
	if err := proto.Unmarshal(envelope.GetClaims(), &signedClaims); err != nil {
		t.Fatalf("decode signed claims: %v", err)
	}
	if signedClaims.GetKeyId() != "m3-current" {
		t.Fatalf("signed key ID = %q, want %q", signedClaims.GetKeyId(), "m3-current")
	}
}

func TestPairingTicketSignerRejectsInvalidClaims(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	signer, err := NewPairingTicketSigner("m3-current", privateKey)
	if err != nil {
		t.Fatalf("NewPairingTicketSigner() error = %v", err)
	}
	now := time.Unix(1_800_000_000, 0)

	tests := map[string]func(*relayv1.ConnectionTicketClaims){
		"caller key ID": func(claims *relayv1.ConnectionTicketClaims) {
			claims.KeyId = "attacker-selected"
		},
		"session purpose": func(claims *relayv1.ConnectionTicketClaims) {
			claims.Purpose = relayv1.TicketPurpose_TICKET_PURPOSE_SESSION
		},
		"unspecified role": func(claims *relayv1.ConnectionTicketClaims) {
			claims.Role = relayv1.EndpointRole_ENDPOINT_ROLE_UNSPECIFIED
		},
		"malformed route ID": func(claims *relayv1.ConnectionTicketClaims) {
			claims.RouteId = "route-1"
		},
		"short nonce": func(claims *relayv1.ConnectionTicketClaims) {
			claims.Nonce = []byte("short")
		},
		"long lifetime": func(claims *relayv1.ConnectionTicketClaims) {
			claims.ExpiresAtUnixSeconds = claims.IssuedAtUnixSeconds + 61
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			claims := validPairingTicketClaims(now)
			mutate(claims)
			if _, err := signer.SignPairingTicket(claims); !errors.Is(err, ErrInvalidPairingTicket) {
				t.Fatalf("SignPairingTicket() error = %v, want %v", err, ErrInvalidPairingTicket)
			}
		})
	}
}

func validPairingTicketClaims(now time.Time) *relayv1.ConnectionTicketClaims {
	return &relayv1.ConnectionTicketClaims{
		TicketId:             "11111111111111111111111111111111",
		RouteId:              "22222222222222222222222222222222",
		Role:                 relayv1.EndpointRole_ENDPOINT_ROLE_CLIENT,
		EndpointId:           "33333333333333333333333333333333",
		RelayRegion:          "eu-test-1",
		IssuedAtUnixSeconds:  now.Unix(),
		ExpiresAtUnixSeconds: now.Add(time.Minute).Unix(),
		Nonce:                bytes.Repeat([]byte{0x24}, pairingTicketNonceSize),
		Purpose:              relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING,
		ProtocolVersion:      pairingTicketProtocolVersion,
		NotBeforeUnixSeconds: now.Unix(),
	}
}
