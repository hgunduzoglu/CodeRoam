package session

import (
	"crypto/ed25519"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"google.golang.org/protobuf/proto"
)

const (
	pairingTicketProtocolVersion = 1
	pairingTicketNonceSize       = 32
	maxPairingTicketEncodedSize  = 2 * 1024
	maxPairingTicketLifetime     = time.Minute
)

// ErrInvalidPairingTicket indicates that claims or signing material violate M3 policy.
var ErrInvalidPairingTicket = errors.New("session: invalid pairing ticket")

// PairingTicketSigner owns one control-plane Ed25519 signing key.
type PairingTicketSigner struct {
	keyID      string
	privateKey ed25519.PrivateKey
}

// NewPairingTicketSigner validates and copies control-plane signing material.
func NewPairingTicketSigner(
	keyID string,
	privateKey ed25519.PrivateKey,
) (*PairingTicketSigner, error) {
	if !validPairingTicketKeyID(keyID) || len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPairingTicket
	}
	seed := privateKey.Seed()
	defer clear(seed)
	expectedPrivateKey := ed25519.NewKeyFromSeed(seed)
	defer clear(expectedPrivateKey)
	if allZero(seed) || subtle.ConstantTimeCompare(privateKey, expectedPrivateKey) != 1 {
		return nil, ErrInvalidPairingTicket
	}

	return &PairingTicketSigner{
		keyID:      keyID,
		privateKey: append(ed25519.PrivateKey(nil), privateKey...),
	}, nil
}

// SignPairingTicket validates pairing-only claims and signs their exact deterministic encoding.
func (s *PairingTicketSigner) SignPairingTicket(
	claims *relayv1.ConnectionTicketClaims,
) ([]byte, error) {
	if s == nil || claims == nil {
		return nil, ErrInvalidPairingTicket
	}

	copied := proto.Clone(claims).(*relayv1.ConnectionTicketClaims)
	if copied.GetKeyId() != "" {
		return nil, fmt.Errorf("%w: caller supplied key ID", ErrInvalidPairingTicket)
	}
	copied.KeyId = s.keyID
	if err := validatePairingTicketClaims(copied); err != nil {
		return nil, err
	}

	claimsBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(copied)
	if err != nil {
		return nil, fmt.Errorf("%w: encode claims: %v", ErrInvalidPairingTicket, err)
	}
	envelope := &relayv1.SignedConnectionTicket{
		Claims:    claimsBytes,
		Signature: ed25519.Sign(s.privateKey, claimsBytes),
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("%w: encode envelope: %v", ErrInvalidPairingTicket, err)
	}
	if len(encoded) == 0 || len(encoded) > maxPairingTicketEncodedSize {
		return nil, fmt.Errorf("%w: invalid encoded size", ErrInvalidPairingTicket)
	}
	return encoded, nil
}

func validatePairingTicketClaims(claims *relayv1.ConnectionTicketClaims) error {
	if claims.GetPurpose() != relayv1.TicketPurpose_TICKET_PURPOSE_PAIRING ||
		claims.GetProtocolVersion() != pairingTicketProtocolVersion {
		return fmt.Errorf("%w: wrong purpose or protocol version", ErrInvalidPairingTicket)
	}
	if claims.GetRole() != relayv1.EndpointRole_ENDPOINT_ROLE_CLIENT &&
		claims.GetRole() != relayv1.EndpointRole_ENDPOINT_ROLE_AGENT {
		return fmt.Errorf("%w: invalid endpoint role", ErrInvalidPairingTicket)
	}
	if !validOpaqueID(claims.GetTicketId()) ||
		!validOpaqueID(claims.GetRouteId()) ||
		!validOpaqueID(claims.GetEndpointId()) ||
		!validRelayRegion(claims.GetRelayRegion()) ||
		!validPairingTicketKeyID(claims.GetKeyId()) {
		return fmt.Errorf("%w: missing or oversized identifier", ErrInvalidPairingTicket)
	}
	if len(claims.GetNonce()) != pairingTicketNonceSize || allZero(claims.GetNonce()) {
		return fmt.Errorf("%w: invalid nonce", ErrInvalidPairingTicket)
	}

	notBefore := time.Unix(claims.GetNotBeforeUnixSeconds(), 0)
	issuedAt := time.Unix(claims.GetIssuedAtUnixSeconds(), 0)
	expiresAt := time.Unix(claims.GetExpiresAtUnixSeconds(), 0)
	if claims.GetNotBeforeUnixSeconds() <= 0 ||
		claims.GetIssuedAtUnixSeconds() <= 0 ||
		claims.GetExpiresAtUnixSeconds() <= 0 ||
		notBefore.After(issuedAt) ||
		issuedAt.After(expiresAt) ||
		expiresAt.Sub(notBefore) > maxPairingTicketLifetime {
		return fmt.Errorf("%w: invalid time window", ErrInvalidPairingTicket)
	}
	return nil
}

func validPairingTicketKeyID(value string) bool {
	return validRelayRegion(value)
}

func validOpaqueID(value string) bool {
	_, err := ids.Parse(value)
	return err == nil
}
