package session

import (
	"errors"
	"fmt"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

const pairingChannelBindingLen = 32

var ErrInvalidPairingAttemptConfirmation = errors.New("invalid pairing attempt confirmation")

// MobilePairingConfirmationSpec contains the authenticated mobile endpoint's
// bounded observation of one completed pairing handshake.
type MobilePairingConfirmationSpec struct {
	Actor                       auth.Actor
	ProtocolVersion             int
	ChannelBinding              []byte
	ObservedAgentPublicKey      cryptox.X25519PublicKey
	ObservedAgentKeyFingerprint string
}

type MobilePairingConfirmation struct {
	ownerID                  auth.UserID
	protocolVersion          int
	channelBinding           [pairingChannelBindingLen]byte
	observedAgentPublicKey   cryptox.X25519PublicKey
	observedAgentFingerprint cryptox.X25519Fingerprint
}

// NewMobilePairingConfirmation copies the channel binding and requires the
// submitted peer fingerprint to match the observed agent public key.
func NewMobilePairingConfirmation(
	spec MobilePairingConfirmationSpec,
) (MobilePairingConfirmation, error) {
	ownerID, ok := spec.Actor.UserID()
	if !ok {
		return MobilePairingConfirmation{}, fmt.Errorf(
			"%w: owner", ErrInvalidPairingAttemptConfirmation,
		)
	}
	if spec.ProtocolVersion != pairingAttemptProtocolVersion {
		return MobilePairingConfirmation{}, fmt.Errorf(
			"%w: protocol version", ErrInvalidPairingAttemptConfirmation,
		)
	}
	if len(spec.ChannelBinding) != pairingChannelBindingLen || allZero(spec.ChannelBinding) {
		return MobilePairingConfirmation{}, fmt.Errorf(
			"%w: channel binding", ErrInvalidPairingAttemptConfirmation,
		)
	}
	publicKey, err := spec.ObservedAgentPublicKey.Bytes()
	if err != nil || allZero(publicKey) {
		return MobilePairingConfirmation{}, fmt.Errorf(
			"%w: observed agent public key", ErrInvalidPairingAttemptConfirmation,
		)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(spec.ObservedAgentPublicKey)
	if err != nil {
		return MobilePairingConfirmation{}, fmt.Errorf(
			"%w: observed agent fingerprint", ErrInvalidPairingAttemptConfirmation,
		)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil || encodedFingerprint != spec.ObservedAgentKeyFingerprint {
		return MobilePairingConfirmation{}, fmt.Errorf(
			"%w: observed agent fingerprint", ErrInvalidPairingAttemptConfirmation,
		)
	}

	confirmation := MobilePairingConfirmation{
		ownerID: ownerID, protocolVersion: spec.ProtocolVersion,
		observedAgentPublicKey:   spec.ObservedAgentPublicKey,
		observedAgentFingerprint: fingerprint,
	}
	copy(confirmation.channelBinding[:], spec.ChannelBinding)
	return confirmation, nil
}

func (confirmation MobilePairingConfirmation) valid() bool {
	ownerID, err := auth.ParseUserID(confirmation.ownerID.String())
	if err != nil || ownerID != confirmation.ownerID ||
		confirmation.protocolVersion != pairingAttemptProtocolVersion ||
		allZero(confirmation.channelBinding[:]) {
		return false
	}
	publicKey, err := confirmation.observedAgentPublicKey.Bytes()
	if err != nil || allZero(publicKey) {
		return false
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(confirmation.observedAgentPublicKey)
	return err == nil && fingerprint.Equal(confirmation.observedAgentFingerprint)
}
