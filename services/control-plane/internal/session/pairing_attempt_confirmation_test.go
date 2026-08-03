package session

import (
	"bytes"
	"errors"
	"testing"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestNewMobilePairingConfirmationCopiesBoundedPeerObservation(t *testing.T) {
	spec := validMobilePairingConfirmationSpec(t)
	wantBinding := append([]byte(nil), spec.ChannelBinding...)
	wantOwner, ok := spec.Actor.UserID()
	if !ok {
		t.Fatal("valid confirmation actor has no owner")
	}

	confirmation, err := NewMobilePairingConfirmation(spec)
	if err != nil {
		t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
	}
	spec.ChannelBinding[0] ^= 0xff
	if confirmation.ownerID != wantOwner ||
		confirmation.protocolVersion != pairingAttemptProtocolVersion ||
		!bytes.Equal(confirmation.channelBinding[:], wantBinding) ||
		!confirmation.observedAgentPublicKey.Equal(spec.ObservedAgentPublicKey) ||
		!confirmation.valid() {
		t.Fatal("NewMobilePairingConfirmation() did not preserve canonical confirmation metadata")
	}
}

func TestNewMobilePairingConfirmationRejectsInvalidBoundaries(t *testing.T) {
	zeroKey, err := cryptox.ParseX25519PublicKey(make([]byte, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(zero) error = %v", err)
	}
	tests := map[string]func(*MobilePairingConfirmationSpec){
		"missing owner":        func(spec *MobilePairingConfirmationSpec) { spec.Actor = auth.Actor{} },
		"unsupported protocol": func(spec *MobilePairingConfirmationSpec) { spec.ProtocolVersion++ },
		"missing binding":      func(spec *MobilePairingConfirmationSpec) { spec.ChannelBinding = nil },
		"short binding": func(spec *MobilePairingConfirmationSpec) {
			spec.ChannelBinding = bytes.Repeat([]byte{0x51}, pairingChannelBindingLen-1)
		},
		"long binding": func(spec *MobilePairingConfirmationSpec) {
			spec.ChannelBinding = bytes.Repeat([]byte{0x51}, pairingChannelBindingLen+1)
		},
		"zero binding": func(spec *MobilePairingConfirmationSpec) {
			spec.ChannelBinding = make([]byte, pairingChannelBindingLen)
		},
		"missing observed key": func(spec *MobilePairingConfirmationSpec) {
			spec.ObservedAgentPublicKey = cryptox.X25519PublicKey{}
		},
		"zero observed key": func(spec *MobilePairingConfirmationSpec) {
			spec.ObservedAgentPublicKey = zeroKey
		},
		"malformed observed fingerprint": func(spec *MobilePairingConfirmationSpec) {
			spec.ObservedAgentKeyFingerprint = "invalid"
		},
		"mismatched observed fingerprint": func(spec *MobilePairingConfirmationSpec) {
			otherKey, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x77}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey(other) error = %v", parseErr)
			}
			fingerprint, fingerprintErr := cryptox.FingerprintX25519PublicKey(otherKey)
			if fingerprintErr != nil {
				t.Fatalf("FingerprintX25519PublicKey(other) error = %v", fingerprintErr)
			}
			spec.ObservedAgentKeyFingerprint, _ = fingerprint.String()
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validMobilePairingConfirmationSpec(t)
			mutate(&spec)
			if _, err := NewMobilePairingConfirmation(spec); !errors.Is(
				err, ErrInvalidPairingAttemptConfirmation,
			) {
				t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
			}
		})
	}
}

func TestMobilePairingConfirmationValidationFailsClosedAfterMutation(t *testing.T) {
	valid, err := NewMobilePairingConfirmation(validMobilePairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewMobilePairingConfirmation() error = %v", err)
	}
	tests := map[string]func(*MobilePairingConfirmation){
		"missing owner": func(confirmation *MobilePairingConfirmation) {
			confirmation.ownerID = auth.UserID{}
		},
		"unsupported protocol": func(confirmation *MobilePairingConfirmation) {
			confirmation.protocolVersion++
		},
		"zero binding": func(confirmation *MobilePairingConfirmation) {
			confirmation.channelBinding = [pairingChannelBindingLen]byte{}
		},
		"changed observed key": func(confirmation *MobilePairingConfirmation) {
			key, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x78}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey(changed) error = %v", parseErr)
			}
			confirmation.observedAgentPublicKey = key
		},
		"missing observed fingerprint": func(confirmation *MobilePairingConfirmation) {
			confirmation.observedAgentFingerprint = cryptox.X25519Fingerprint{}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			confirmation := valid
			mutate(&confirmation)
			if confirmation.valid() {
				t.Fatal("valid() = true for mutated confirmation")
			}
		})
	}
}

func validMobilePairingConfirmationSpec(t *testing.T) MobilePairingConfirmationSpec {
	t.Helper()
	agentKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(agent) error = %v", err)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(agentKey)
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey(agent) error = %v", err)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		t.Fatalf("agent fingerprint String() error = %v", err)
	}
	return MobilePairingConfirmationSpec{
		Actor:                  newSessionTestActor(t, "0123456789abcdef0123456789abcdef"),
		ProtocolVersion:        pairingAttemptProtocolVersion,
		ChannelBinding:         bytes.Repeat([]byte{0x51}, pairingChannelBindingLen),
		ObservedAgentPublicKey: agentKey, ObservedAgentKeyFingerprint: encodedFingerprint,
	}
}

func TestNewAgentPairingConfirmationCopiesBoundedPeerObservation(t *testing.T) {
	spec := validAgentPairingConfirmationSpec(t)
	wantBinding := append([]byte(nil), spec.ChannelBinding...)

	confirmation, err := NewAgentPairingConfirmation(spec)
	if err != nil {
		t.Fatalf("NewAgentPairingConfirmation() error = %v", err)
	}
	spec.ChannelBinding[0] ^= 0xff
	if confirmation.protocolVersion != pairingAttemptProtocolVersion ||
		!bytes.Equal(confirmation.channelBinding[:], wantBinding) ||
		!confirmation.observedDevicePublicKey.Equal(spec.ObservedDevicePublicKey) ||
		!confirmation.valid() {
		t.Fatal("NewAgentPairingConfirmation() did not preserve canonical confirmation metadata")
	}
}

func TestNewAgentPairingConfirmationRejectsInvalidBoundaries(t *testing.T) {
	zeroKey, err := cryptox.ParseX25519PublicKey(make([]byte, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(zero) error = %v", err)
	}
	tests := map[string]func(*AgentPairingConfirmationSpec){
		"unsupported protocol": func(spec *AgentPairingConfirmationSpec) { spec.ProtocolVersion++ },
		"missing binding":      func(spec *AgentPairingConfirmationSpec) { spec.ChannelBinding = nil },
		"short binding": func(spec *AgentPairingConfirmationSpec) {
			spec.ChannelBinding = bytes.Repeat([]byte{0x51}, pairingChannelBindingLen-1)
		},
		"long binding": func(spec *AgentPairingConfirmationSpec) {
			spec.ChannelBinding = bytes.Repeat([]byte{0x51}, pairingChannelBindingLen+1)
		},
		"zero binding": func(spec *AgentPairingConfirmationSpec) {
			spec.ChannelBinding = make([]byte, pairingChannelBindingLen)
		},
		"missing observed key": func(spec *AgentPairingConfirmationSpec) {
			spec.ObservedDevicePublicKey = cryptox.X25519PublicKey{}
		},
		"zero observed key": func(spec *AgentPairingConfirmationSpec) {
			spec.ObservedDevicePublicKey = zeroKey
		},
		"malformed observed fingerprint": func(spec *AgentPairingConfirmationSpec) {
			spec.ObservedDeviceKeyFingerprint = "invalid"
		},
		"mismatched observed fingerprint": func(spec *AgentPairingConfirmationSpec) {
			otherKey, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x77}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey(other device) error = %v", parseErr)
			}
			fingerprint, fingerprintErr := cryptox.FingerprintX25519PublicKey(otherKey)
			if fingerprintErr != nil {
				t.Fatalf("FingerprintX25519PublicKey(other device) error = %v", fingerprintErr)
			}
			spec.ObservedDeviceKeyFingerprint, _ = fingerprint.String()
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validAgentPairingConfirmationSpec(t)
			mutate(&spec)
			if _, err := NewAgentPairingConfirmation(spec); !errors.Is(
				err, ErrInvalidPairingAttemptConfirmation,
			) {
				t.Fatalf("NewAgentPairingConfirmation() error = %v", err)
			}
		})
	}
}

func TestAgentPairingConfirmationValidationFailsClosedAfterMutation(t *testing.T) {
	valid, err := NewAgentPairingConfirmation(validAgentPairingConfirmationSpec(t))
	if err != nil {
		t.Fatalf("NewAgentPairingConfirmation() error = %v", err)
	}
	tests := map[string]func(*AgentPairingConfirmation){
		"unsupported protocol": func(confirmation *AgentPairingConfirmation) {
			confirmation.protocolVersion++
		},
		"zero binding": func(confirmation *AgentPairingConfirmation) {
			confirmation.channelBinding = [pairingChannelBindingLen]byte{}
		},
		"changed observed key": func(confirmation *AgentPairingConfirmation) {
			key, parseErr := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x78}, 32))
			if parseErr != nil {
				t.Fatalf("ParseX25519PublicKey(changed device) error = %v", parseErr)
			}
			confirmation.observedDevicePublicKey = key
		},
		"missing observed fingerprint": func(confirmation *AgentPairingConfirmation) {
			confirmation.observedDeviceFingerprint = cryptox.X25519Fingerprint{}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			confirmation := valid
			mutate(&confirmation)
			if confirmation.valid() {
				t.Fatal("valid() = true for mutated agent confirmation")
			}
		})
	}
}

func validAgentPairingConfirmationSpec(t *testing.T) AgentPairingConfirmationSpec {
	t.Helper()
	deviceKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x64}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(device) error = %v", err)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(deviceKey)
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey(device) error = %v", err)
	}
	encodedFingerprint, err := fingerprint.String()
	if err != nil {
		t.Fatalf("device fingerprint String() error = %v", err)
	}
	return AgentPairingConfirmationSpec{
		ProtocolVersion:         pairingAttemptProtocolVersion,
		ChannelBinding:          bytes.Repeat([]byte{0x51}, pairingChannelBindingLen),
		ObservedDevicePublicKey: deviceKey, ObservedDeviceKeyFingerprint: encodedFingerprint,
	}
}
