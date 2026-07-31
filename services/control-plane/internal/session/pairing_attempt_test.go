package session

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
)

func TestNewPairingAttemptNormalizesAndCopiesBoundedMetadata(t *testing.T) {
	spec := validPairingAttemptSpec(t)
	hashInput := spec.BootstrapCredentialHash
	wantFingerprint, err := cryptox.FingerprintX25519PublicKey(spec.AgentPublicKey)
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey() error = %v", err)
	}

	attempt, err := NewPairingAttempt(spec)
	if err != nil {
		t.Fatalf("NewPairingAttempt() error = %v", err)
	}
	hashInput[0] ^= 0xff
	if attempt.id.String() != spec.ID || !attempt.agentPublicKey.Equal(spec.AgentPublicKey) ||
		!attempt.agentFingerprint.Equal(wantFingerprint) || attempt.agentDisplayName != "M3 agent" ||
		attempt.agentVersion != "0.1.0" || attempt.protocolVersion != pairingAttemptProtocolVersion ||
		attempt.relayRegion != "eu-test-1" || attempt.bootstrapCredentialHash[0] != 0x24 ||
		attempt.failedAttemptCount != 0 || attempt.state != pairingAttemptStateOpen ||
		!attempt.createdAt.Equal(spec.CreatedAt) || attempt.createdAt.Location() != time.UTC ||
		!attempt.expiresAt.Equal(spec.ExpiresAt) || attempt.expiresAt.Location() != time.UTC ||
		!attempt.updatedAt.Equal(spec.CreatedAt) {
		t.Fatal("NewPairingAttempt() did not preserve canonical open-attempt metadata")
	}
}

func TestNewPairingAttemptRejectsInvalidBoundaries(t *testing.T) {
	zeroKey, err := cryptox.ParseX25519PublicKey(make([]byte, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(zero) error = %v", err)
	}
	tests := map[string]func(*PairingAttemptSpec){
		"invalid id": func(spec *PairingAttemptSpec) { spec.ID = "invalid" },
		"missing public key": func(spec *PairingAttemptSpec) {
			spec.AgentPublicKey = cryptox.X25519PublicKey{}
		},
		"zero public key":    func(spec *PairingAttemptSpec) { spec.AgentPublicKey = zeroKey },
		"empty display name": func(spec *PairingAttemptSpec) { spec.AgentDisplayName = "  " },
		"oversized display name": func(spec *PairingAttemptSpec) {
			spec.AgentDisplayName = strings.Repeat("a", maxPairingAgentNameBytes+1)
		},
		"control in display name": func(spec *PairingAttemptSpec) { spec.AgentDisplayName = "agent\nname" },
		"bidi control in display name": func(spec *PairingAttemptSpec) {
			spec.AgentDisplayName = "agent\u202ename"
		},
		"line separator in display name": func(spec *PairingAttemptSpec) {
			spec.AgentDisplayName = "agent\u2028name"
		},
		"invalid display name utf8": func(spec *PairingAttemptSpec) {
			spec.AgentDisplayName = string([]byte{0xff})
		},
		"empty version": func(spec *PairingAttemptSpec) { spec.AgentVersion = "" },
		"oversized version": func(spec *PairingAttemptSpec) {
			spec.AgentVersion = strings.Repeat("v", maxPairingAgentVersionBytes+1)
		},
		"control in version":             func(spec *PairingAttemptSpec) { spec.AgentVersion = "0.1\n" },
		"bidi isolate in version":        func(spec *PairingAttemptSpec) { spec.AgentVersion = "0.1\u2066" },
		"paragraph separator in version": func(spec *PairingAttemptSpec) { spec.AgentVersion = "0.1\u2029" },
		"invalid version utf8": func(spec *PairingAttemptSpec) {
			spec.AgentVersion = string([]byte{0xff})
		},
		"unsupported protocol":        func(spec *PairingAttemptSpec) { spec.ProtocolVersion++ },
		"invalid relay region":        func(spec *PairingAttemptSpec) { spec.RelayRegion = "EU-test-1" },
		"consecutive relay separator": func(spec *PairingAttemptSpec) { spec.RelayRegion = "eu--test" },
		"short bootstrap hash": func(spec *PairingAttemptSpec) {
			spec.BootstrapCredentialHash = bytes.Repeat([]byte{0x24}, pairingAttemptBootstrapHashLen-1)
		},
		"zero bootstrap hash": func(spec *PairingAttemptSpec) {
			spec.BootstrapCredentialHash = make([]byte, pairingAttemptBootstrapHashLen)
		},
		"zero creation time":   func(spec *PairingAttemptSpec) { spec.CreatedAt = time.Time{} },
		"zero expiry time":     func(spec *PairingAttemptSpec) { spec.ExpiresAt = time.Time{} },
		"nonpositive lifetime": func(spec *PairingAttemptSpec) { spec.ExpiresAt = spec.CreatedAt },
		"sub-microsecond lifetime": func(spec *PairingAttemptSpec) {
			spec.CreatedAt = spec.CreatedAt.Truncate(time.Microsecond)
			spec.ExpiresAt = spec.CreatedAt.Add(time.Nanosecond)
		},
		"excess lifetime": func(spec *PairingAttemptSpec) {
			spec.ExpiresAt = spec.CreatedAt.Add(maxPairingAttemptLifetime + time.Microsecond)
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validPairingAttemptSpec(t)
			mutate(&spec)
			if _, err := NewPairingAttempt(spec); !errors.Is(err, ErrInvalidPairingAttempt) {
				t.Fatalf("NewPairingAttempt() error = %v, want ErrInvalidPairingAttempt", err)
			}
		})
	}
}

func TestNewPairingAttemptAcceptsCanonicalMicrosecondLifetime(t *testing.T) {
	spec := validPairingAttemptSpec(t)
	spec.CreatedAt = spec.CreatedAt.Truncate(time.Microsecond)
	spec.ExpiresAt = spec.CreatedAt.Add(time.Microsecond)
	if _, err := NewPairingAttempt(spec); err != nil {
		t.Fatalf("NewPairingAttempt(microsecond lifetime) error = %v", err)
	}
}

func TestPairingAttemptValidForCreateFailsClosed(t *testing.T) {
	valid, err := NewPairingAttempt(validPairingAttemptSpec(t))
	if err != nil {
		t.Fatalf("NewPairingAttempt() error = %v", err)
	}
	if !valid.validForCreate() {
		t.Fatal("validForCreate() = false for constructor output")
	}

	tests := map[string]func(*PairingAttempt){
		"changed fingerprint": func(attempt *PairingAttempt) {
			attempt.agentFingerprint = cryptox.X25519Fingerprint{}
		},
		"unnormalized display name": func(attempt *PairingAttempt) {
			attempt.agentDisplayName = " M3 agent"
		},
		"failed attempt": func(attempt *PairingAttempt) { attempt.failedAttemptCount = 1 },
		"changed state":  func(attempt *PairingAttempt) { attempt.state = "claimed" },
		"changed update time": func(attempt *PairingAttempt) {
			attempt.updatedAt = attempt.updatedAt.Add(time.Second)
		},
		"zero bootstrap hash": func(attempt *PairingAttempt) {
			attempt.bootstrapCredentialHash = [pairingAttemptBootstrapHashLen]byte{}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			attempt := valid
			mutate(&attempt)
			if attempt.validForCreate() {
				t.Fatal("validForCreate() = true for mutated attempt")
			}
		})
	}
}

func validPairingAttemptSpec(t *testing.T) PairingAttemptSpec {
	t.Helper()
	publicKey, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	createdAt := time.Date(2026, time.July, 31, 16, 0, 0, 0, time.FixedZone("test", 3*60*60))
	return PairingAttemptSpec{
		ID: strings.Repeat("a", 32), AgentPublicKey: publicKey,
		AgentDisplayName: "  M3 agent  ", AgentVersion: "  0.1.0  ",
		ProtocolVersion: pairingAttemptProtocolVersion, RelayRegion: "eu-test-1",
		BootstrapCredentialHash: bytes.Repeat([]byte{0x24}, pairingAttemptBootstrapHashLen),
		CreatedAt:               createdAt, ExpiresAt: createdAt.Add(maxPairingAttemptLifetime),
	}
}
