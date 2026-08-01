package session

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
)

func TestNewPairingAttemptNormalizesAndCopiesBoundedMetadata(t *testing.T) {
	spec := validPairingAttemptSpec(t)
	hashInput := spec.BootstrapCredentialHash
	wantHash := append([]byte(nil), hashInput...)
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
		attempt.relayRegion != "eu-test-1" || !bytes.Equal(attempt.bootstrapCredentialHash[:], wantHash) ||
		attempt.failedAttemptCount != 0 || attempt.state != pairingAttemptStateOpen ||
		!attempt.createdAt.Equal(spec.CreatedAt) || attempt.createdAt.Location() != time.UTC ||
		!attempt.expiresAt.Equal(spec.ExpiresAt) || attempt.expiresAt.Location() != time.UTC ||
		!attempt.updatedAt.Equal(spec.CreatedAt) {
		t.Fatal("NewPairingAttempt() did not preserve canonical open-attempt metadata")
	}
}

func TestHashPairingBootstrapCredentialBindsCanonicalAttempt(t *testing.T) {
	id := strings.Repeat("a", 32)
	credential := validPairingBootstrapCredential()
	digest, err := HashPairingBootstrapCredential(id, credential)
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential() error = %v", err)
	}
	const want = "14d85226f406aab079655f9db630c899a93f55ae4368c8bfdf5a392d1107c316"
	if got := fmt.Sprintf("%x", digest); got != want {
		t.Fatalf("HashPairingBootstrapCredential() = %s, want %s", got, want)
	}
	if !bytes.Equal(credential, validPairingBootstrapCredential()) {
		t.Fatal("HashPairingBootstrapCredential() mutated caller-owned credential bytes")
	}

	otherIDHash, err := HashPairingBootstrapCredential(strings.Repeat("b", 32), credential)
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential(other id) error = %v", err)
	}
	otherCredential := append([]byte(nil), credential...)
	otherCredential[0] ^= 0xff
	otherCredentialHash, err := HashPairingBootstrapCredential(id, otherCredential)
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential(other credential) error = %v", err)
	}
	if bytes.Equal(digest[:], otherIDHash[:]) || bytes.Equal(digest[:], otherCredentialHash[:]) {
		t.Fatal("bootstrap credential hash was not bound to both attempt id and credential")
	}
}

func TestHashPairingBootstrapCredentialRejectsInvalidInput(t *testing.T) {
	id := strings.Repeat("a", 32)
	tests := map[string]struct {
		id         string
		credential []byte
	}{
		"invalid id":       {id: "invalid", credential: validPairingBootstrapCredential()},
		"short credential": {id: id, credential: bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen-1)},
		"long credential":  {id: id, credential: bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen+1)},
		"zero credential":  {id: id, credential: make([]byte, pairingBootstrapCredentialLen)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := HashPairingBootstrapCredential(test.id, test.credential); !errors.Is(
				err, ErrInvalidPairingBootstrapCredential,
			) {
				t.Fatalf("HashPairingBootstrapCredential() error = %v", err)
			}
		})
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
		"locked attempt": func(attempt *PairingAttempt) { attempt.lockedAt = attempt.createdAt },
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
	id := strings.Repeat("a", 32)
	hash, err := HashPairingBootstrapCredential(id, validPairingBootstrapCredential())
	if err != nil {
		t.Fatalf("HashPairingBootstrapCredential(fixture) error = %v", err)
	}
	return PairingAttemptSpec{
		ID: id, AgentPublicKey: publicKey,
		AgentDisplayName: "  M3 agent  ", AgentVersion: "  0.1.0  ",
		ProtocolVersion: pairingAttemptProtocolVersion, RelayRegion: "eu-test-1",
		BootstrapCredentialHash: hash[:],
		CreatedAt:               createdAt, ExpiresAt: createdAt.Add(maxPairingAttemptLifetime),
	}
}

func validPairingBootstrapCredential() []byte {
	return bytes.Repeat([]byte{0x5a}, pairingBootstrapCredentialLen)
}
