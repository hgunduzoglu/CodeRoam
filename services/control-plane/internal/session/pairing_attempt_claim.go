package session

import (
	"errors"
	"fmt"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
)

const (
	maxPairingDeviceNameRunes = 128
	maxPairingDeviceNameBytes = maxPairingDeviceNameRunes * 4
)

var ErrInvalidPairingAttemptClaim = errors.New("invalid pairing attempt claim")

// PairingAttemptClaimSpec contains the authenticated owner and public mobile identity
// candidate that will be bound to one open pairing attempt.
type PairingAttemptClaimSpec struct {
	Actor                  auth.Actor
	ExpectedAgentPublicKey cryptox.X25519PublicKey
	ProtocolVersion        int
	DeviceID               string
	DeviceDisplayName      string
	DevicePlatform         device.Platform
	DevicePublicKey        cryptox.X25519PublicKey
}

type PairingAttemptClaim struct {
	ownerID                  auth.UserID
	expectedAgentPublicKey   cryptox.X25519PublicKey
	expectedAgentFingerprint cryptox.X25519Fingerprint
	protocolVersion          int
	deviceID                 ids.ID
	deviceDisplayName        string
	devicePlatform           device.Platform
	devicePublicKey          cryptox.X25519PublicKey
	deviceFingerprint        cryptox.X25519Fingerprint
}

// NewPairingAttemptClaim validates one owner-bound public mobile candidate and
// derives its canonical fingerprint instead of accepting fingerprint text.
func NewPairingAttemptClaim(spec PairingAttemptClaimSpec) (PairingAttemptClaim, error) {
	ownerID, ok := spec.Actor.UserID()
	if !ok {
		return PairingAttemptClaim{}, fmt.Errorf("%w: owner", ErrInvalidPairingAttemptClaim)
	}
	expectedAgentKey, err := spec.ExpectedAgentPublicKey.Bytes()
	if err != nil || allZero(expectedAgentKey) {
		return PairingAttemptClaim{}, fmt.Errorf("%w: expected agent public key", ErrInvalidPairingAttemptClaim)
	}
	expectedAgentFingerprint, err := cryptox.FingerprintX25519PublicKey(spec.ExpectedAgentPublicKey)
	if err != nil {
		return PairingAttemptClaim{}, fmt.Errorf("%w: expected agent fingerprint", ErrInvalidPairingAttemptClaim)
	}
	if spec.ProtocolVersion != pairingAttemptProtocolVersion {
		return PairingAttemptClaim{}, fmt.Errorf("%w: protocol version", ErrInvalidPairingAttemptClaim)
	}
	deviceID, err := ids.Parse(spec.DeviceID)
	if err != nil {
		return PairingAttemptClaim{}, fmt.Errorf("%w: device id", ErrInvalidPairingAttemptClaim)
	}
	displayName, ok := normalizedPairingText(
		spec.DeviceDisplayName, maxPairingDeviceNameRunes, maxPairingDeviceNameBytes,
	)
	if !ok {
		return PairingAttemptClaim{}, fmt.Errorf("%w: device display name", ErrInvalidPairingAttemptClaim)
	}
	if !validPairingDevicePlatform(spec.DevicePlatform) {
		return PairingAttemptClaim{}, fmt.Errorf("%w: device platform", ErrInvalidPairingAttemptClaim)
	}
	publicKeyBytes, err := spec.DevicePublicKey.Bytes()
	if err != nil || allZero(publicKeyBytes) {
		return PairingAttemptClaim{}, fmt.Errorf("%w: device public key", ErrInvalidPairingAttemptClaim)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(spec.DevicePublicKey)
	if err != nil {
		return PairingAttemptClaim{}, fmt.Errorf("%w: device fingerprint", ErrInvalidPairingAttemptClaim)
	}

	return PairingAttemptClaim{
		ownerID: ownerID, expectedAgentPublicKey: spec.ExpectedAgentPublicKey,
		expectedAgentFingerprint: expectedAgentFingerprint, protocolVersion: spec.ProtocolVersion,
		deviceID: deviceID, deviceDisplayName: displayName, devicePlatform: spec.DevicePlatform,
		devicePublicKey: spec.DevicePublicKey, deviceFingerprint: fingerprint,
	}, nil
}

func (claim PairingAttemptClaim) valid() bool {
	ownerID, err := auth.ParseUserID(claim.ownerID.String())
	if err != nil || ownerID != claim.ownerID {
		return false
	}
	expectedAgentKey, err := claim.expectedAgentPublicKey.Bytes()
	if err != nil || allZero(expectedAgentKey) || claim.protocolVersion != pairingAttemptProtocolVersion {
		return false
	}
	expectedAgentFingerprint, err := cryptox.FingerprintX25519PublicKey(claim.expectedAgentPublicKey)
	if err != nil || !expectedAgentFingerprint.Equal(claim.expectedAgentFingerprint) {
		return false
	}
	deviceID, err := ids.Parse(claim.deviceID.String())
	if err != nil || deviceID != claim.deviceID {
		return false
	}
	displayName, ok := normalizedPairingText(
		claim.deviceDisplayName, maxPairingDeviceNameRunes, maxPairingDeviceNameBytes,
	)
	if !ok || displayName != claim.deviceDisplayName || !validPairingDevicePlatform(claim.devicePlatform) {
		return false
	}
	publicKeyBytes, err := claim.devicePublicKey.Bytes()
	if err != nil || allZero(publicKeyBytes) {
		return false
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(claim.devicePublicKey)
	return err == nil && fingerprint.Equal(claim.deviceFingerprint)
}

func validPairingDevicePlatform(platform device.Platform) bool {
	return platform == device.PlatformIOS || platform == device.PlatformIPadOS ||
		platform == device.PlatformAndroid
}
