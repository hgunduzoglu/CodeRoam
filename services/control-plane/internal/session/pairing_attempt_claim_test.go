package session

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/device"
)

func TestNewPairingAttemptClaimNormalizesOwnerBoundCandidate(t *testing.T) {
	spec := validPairingAttemptClaimSpec(t)
	wantFingerprint, err := cryptox.FingerprintX25519PublicKey(spec.DevicePublicKey)
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey() error = %v", err)
	}

	claim, err := NewPairingAttemptClaim(spec)
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	ownerID, ok := spec.Actor.UserID()
	if !ok || claim.ownerID != ownerID ||
		!claim.expectedAgentPublicKey.Equal(spec.ExpectedAgentPublicKey) ||
		claim.protocolVersion != pairingAttemptProtocolVersion || claim.deviceID.String() != spec.DeviceID ||
		claim.deviceDisplayName != "Owner phone" || claim.devicePlatform != device.PlatformIOS ||
		!claim.devicePublicKey.Equal(spec.DevicePublicKey) ||
		!claim.deviceFingerprint.Equal(wantFingerprint) || !claim.valid() {
		t.Fatal("NewPairingAttemptClaim() did not preserve canonical owner-bound metadata")
	}
}

func TestNewPairingAttemptClaimRejectsInvalidBoundaries(t *testing.T) {
	tests := map[string]func(*PairingAttemptClaimSpec){
		"missing owner": func(spec *PairingAttemptClaimSpec) { spec.Actor = auth.Actor{} },
		"missing expected agent key": func(spec *PairingAttemptClaimSpec) {
			spec.ExpectedAgentPublicKey = cryptox.X25519PublicKey{}
		},
		"unsupported protocol": func(spec *PairingAttemptClaimSpec) { spec.ProtocolVersion++ },
		"invalid device id":    func(spec *PairingAttemptClaimSpec) { spec.DeviceID = "invalid" },
		"empty display name":   func(spec *PairingAttemptClaimSpec) { spec.DeviceDisplayName = "  " },
		"oversized display name": func(spec *PairingAttemptClaimSpec) {
			spec.DeviceDisplayName = strings.Repeat("a", maxPairingDeviceNameBytes+1)
		},
		"control in display name": func(spec *PairingAttemptClaimSpec) {
			spec.DeviceDisplayName = "owner\nphone"
		},
		"bidi control in display name": func(spec *PairingAttemptClaimSpec) {
			spec.DeviceDisplayName = "owner\u202ephone"
		},
		"unsupported platform": func(spec *PairingAttemptClaimSpec) {
			spec.DevicePlatform = device.Platform("linux")
		},
		"missing public key": func(spec *PairingAttemptClaimSpec) {
			spec.DevicePublicKey = cryptox.X25519PublicKey{}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validPairingAttemptClaimSpec(t)
			mutate(&spec)
			if _, err := NewPairingAttemptClaim(spec); !errors.Is(err, ErrInvalidPairingAttemptClaim) {
				t.Fatalf("NewPairingAttemptClaim() error = %v", err)
			}
		})
	}
}

func TestPairingAttemptClaimValidationFailsClosedAfterMutation(t *testing.T) {
	valid, err := NewPairingAttemptClaim(validPairingAttemptClaimSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttemptClaim() error = %v", err)
	}
	tests := map[string]func(*PairingAttemptClaim){
		"missing owner": func(claim *PairingAttemptClaim) { claim.ownerID = auth.UserID{} },
		"changed expected agent key": func(claim *PairingAttemptClaim) {
			key, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x66}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey() error = %v", parseErr)
			}
			claim.expectedAgentPublicKey = key
		},
		"missing expected agent fingerprint": func(claim *PairingAttemptClaim) {
			claim.expectedAgentFingerprint = cryptox.X25519Fingerprint{}
		},
		"unsupported protocol": func(claim *PairingAttemptClaim) { claim.protocolVersion++ },
		"missing device id":    func(claim *PairingAttemptClaim) { claim.deviceID = ids.ID{} },
		"unnormalized display name": func(claim *PairingAttemptClaim) {
			claim.deviceDisplayName = " Owner phone"
		},
		"unsupported platform": func(claim *PairingAttemptClaim) {
			claim.devicePlatform = device.Platform("linux")
		},
		"changed public key": func(claim *PairingAttemptClaim) {
			key, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x65}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey() error = %v", parseErr)
			}
			claim.devicePublicKey = key
		},
		"missing fingerprint": func(claim *PairingAttemptClaim) {
			claim.deviceFingerprint = cryptox.X25519Fingerprint{}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			claim := valid
			mutate(&claim)
			if claim.valid() {
				t.Fatal("valid() = true for mutated claim")
			}
		})
	}
}

func validPairingAttemptClaimSpec(t *testing.T) PairingAttemptClaimSpec {
	t.Helper()
	publicKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x64}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	expectedAgentKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(expected agent) error = %v", err)
	}
	return PairingAttemptClaimSpec{
		Actor:                  newSessionTestActor(t, "0123456789abcdef0123456789abcdef"),
		ExpectedAgentPublicKey: expectedAgentKey, ProtocolVersion: pairingAttemptProtocolVersion,
		DeviceID: "1123456789abcdef0123456789abcdef", DeviceDisplayName: "  Owner phone  ",
		DevicePlatform: device.PlatformIOS, DevicePublicKey: publicKey,
	}
}
