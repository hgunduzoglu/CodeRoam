package ticket

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"google.golang.org/protobuf/proto"
)

const (
	pairingProtocolVersion = 1
	pairingNonceSize       = 32
	maxPairingLifetime     = time.Minute
	pairingClockSkew       = 5 * time.Second
	maxTicketSize          = 2 * 1024
	maxClaimsSize          = 1024
)

// ErrInvalidPairingTicket is returned for every untrusted ticket failure.
var ErrInvalidPairingTicket = errors.New("relay: invalid pairing ticket")

// VerificationKey is one explicitly configured Ed25519 ticket key.
type VerificationKey struct {
	ID        string
	PublicKey ed25519.PublicKey
}

// PreviousVerificationKey limits the previous key to tickets issued through a cutover instant.
type PreviousVerificationKey struct {
	VerificationKey
	IssuedThrough time.Time
}

type verificationPolicy struct {
	publicKey     ed25519.PublicKey
	issuedThrough time.Time
}

// PairingTicketVerifier accepts the current key and, during rotation, one previous key.
type PairingTicketVerifier struct {
	relayRegion string
	keys        map[string]verificationPolicy
}

// NewPairingTicketVerifier validates and copies the pinned verification keys.
func NewPairingTicketVerifier(
	relayRegion string,
	current VerificationKey,
	previous *PreviousVerificationKey,
) (*PairingTicketVerifier, error) {
	if !boundedNonempty(relayRegion, 64) || !validVerificationKey(current) {
		return nil, ErrInvalidPairingTicket
	}
	keys := map[string]verificationPolicy{
		current.ID: {
			publicKey: append(ed25519.PublicKey(nil), current.PublicKey...),
		},
	}
	if previous != nil {
		if !validVerificationKey(previous.VerificationKey) ||
			previous.ID == current.ID ||
			previous.IssuedThrough.IsZero() {
			return nil, ErrInvalidPairingTicket
		}
		keys[previous.ID] = verificationPolicy{
			publicKey:     append(ed25519.PublicKey(nil), previous.PublicKey...),
			issuedThrough: previous.IssuedThrough,
		}
	}
	return &PairingTicketVerifier{relayRegion: relayRegion, keys: keys}, nil
}

// VerifyPairingTicket authenticates and validates one bounded pairing-only ticket.
func (v *PairingTicketVerifier) VerifyPairingTicket(
	encoded []byte,
	now time.Time,
) (*relayv1.ConnectionTicketClaims, error) {
	if v == nil || now.IsZero() || len(encoded) == 0 || len(encoded) > maxTicketSize {
		return nil, ErrInvalidPairingTicket
	}

	var envelope relayv1.SignedConnectionTicket
	if err := proto.Unmarshal(encoded, &envelope); err != nil {
		return nil, ErrInvalidPairingTicket
	}
	if len(envelope.GetClaims()) == 0 ||
		len(envelope.GetClaims()) > maxClaimsSize ||
		len(envelope.GetSignature()) != ed25519.SignatureSize {
		return nil, ErrInvalidPairingTicket
	}
	canonicalEnvelope, err := proto.MarshalOptions{Deterministic: true}.Marshal(&envelope)
	if err != nil || !bytes.Equal(canonicalEnvelope, encoded) {
		return nil, ErrInvalidPairingTicket
	}

	var claims relayv1.ConnectionTicketClaims
	if err := proto.Unmarshal(envelope.GetClaims(), &claims); err != nil {
		return nil, ErrInvalidPairingTicket
	}
	canonicalClaims, err := proto.MarshalOptions{Deterministic: true}.Marshal(&claims)
	if err != nil || !bytes.Equal(canonicalClaims, envelope.GetClaims()) {
		return nil, ErrInvalidPairingTicket
	}
	policy, ok := v.keys[claims.GetKeyId()]
	if !ok || !ed25519.Verify(policy.publicKey, envelope.GetClaims(), envelope.GetSignature()) {
		return nil, ErrInvalidPairingTicket
	}
	issuedAt := time.Unix(claims.GetIssuedAtUnixSeconds(), 0)
	if !policy.issuedThrough.IsZero() && issuedAt.After(policy.issuedThrough) {
		return nil, ErrInvalidPairingTicket
	}
	if err := v.validateClaims(&claims, now); err != nil {
		return nil, err
	}
	return proto.Clone(&claims).(*relayv1.ConnectionTicketClaims), nil
}

func (v *PairingTicketVerifier) validateClaims(
	claims *relayv1.ConnectionTicketClaims,
	now time.Time,
) error {
	if claims.GetPurpose() != relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING ||
		claims.GetProtocolVersion() != pairingProtocolVersion ||
		claims.GetRelayRegion() != v.relayRegion {
		return ErrInvalidPairingTicket
	}
	if claims.GetRole() != relayv1.EndpointRole_ENDPOINT_ROLE_CLIENT &&
		claims.GetRole() != relayv1.EndpointRole_ENDPOINT_ROLE_AGENT {
		return ErrInvalidPairingTicket
	}
	if !validOpaqueID(claims.GetTicketId()) ||
		!validOpaqueID(claims.GetRouteId()) ||
		!validOpaqueID(claims.GetEndpointId()) ||
		!boundedNonempty(claims.GetKeyId(), 64) ||
		len(claims.GetNonce()) != pairingNonceSize {
		return ErrInvalidPairingTicket
	}

	notBefore := time.Unix(claims.GetNotBeforeUnixSeconds(), 0)
	issuedAt := time.Unix(claims.GetIssuedAtUnixSeconds(), 0)
	expiresAt := time.Unix(claims.GetExpiresAtUnixSeconds(), 0)
	if claims.GetNotBeforeUnixSeconds() <= 0 ||
		claims.GetIssuedAtUnixSeconds() <= 0 ||
		claims.GetExpiresAtUnixSeconds() <= 0 ||
		notBefore.After(issuedAt) ||
		issuedAt.After(expiresAt) ||
		expiresAt.Sub(notBefore) > maxPairingLifetime ||
		issuedAt.After(now.Add(pairingClockSkew)) ||
		now.Before(notBefore.Add(-pairingClockSkew)) ||
		now.After(expiresAt.Add(pairingClockSkew)) {
		return ErrInvalidPairingTicket
	}
	return nil
}

func validVerificationKey(key VerificationKey) bool {
	return boundedNonempty(key.ID, 64) && len(key.PublicKey) == ed25519.PublicKeySize
}

func boundedNonempty(value string, limit int) bool {
	return value != "" && len(value) <= limit
}

func validOpaqueID(value string) bool {
	_, err := ids.Parse(value)
	return err == nil
}
