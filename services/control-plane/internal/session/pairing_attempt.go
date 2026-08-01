package session

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
)

const (
	pairingAttemptProtocolVersion    = 1
	pairingAttemptEncodedIDLen       = 32
	pairingBootstrapCredentialLen    = 32
	pairingAttemptBootstrapHashLen   = sha256.Size
	pairingBootstrapCredentialDomain = "coderoam:m3:pairing-bootstrap-credential:v1"
	pairingBootstrapHashInputLen     = len(pairingBootstrapCredentialDomain) + 1 +
		pairingAttemptEncodedIDLen + 1 + pairingBootstrapCredentialLen
	maxPairingAttemptLifetime   = 5 * time.Minute
	maxPairingAttemptFailures   = 8
	maxPairingAgentNameRunes    = 128
	maxPairingAgentNameBytes    = maxPairingAgentNameRunes * utf8.UTFMax
	maxPairingAgentVersionBytes = 64
)

var (
	ErrInvalidPairingAttempt             = errors.New("invalid pairing attempt")
	ErrInvalidPairingBootstrapCredential = errors.New("invalid pairing bootstrap credential")
)

type pairingAttemptState string

const (
	pairingAttemptStateOpen    pairingAttemptState = "open"
	pairingAttemptStateClaimed pairingAttemptState = "claimed"
)

// PairingAttemptSpec contains public candidate metadata and an already domain-separated
// bootstrap-credential hash. It must never contain the raw bootstrap credential or pairing secret.
type PairingAttemptSpec struct {
	ID                      string
	AgentPublicKey          cryptox.X25519PublicKey
	AgentDisplayName        string
	AgentVersion            string
	ProtocolVersion         int
	RelayRegion             string
	BootstrapCredentialHash []byte
	CreatedAt               time.Time
	ExpiresAt               time.Time
}

type PairingAttempt struct {
	id                      ids.ID
	agentPublicKey          cryptox.X25519PublicKey
	agentFingerprint        cryptox.X25519Fingerprint
	agentDisplayName        string
	agentVersion            string
	protocolVersion         int
	relayRegion             string
	bootstrapCredentialHash [pairingAttemptBootstrapHashLen]byte
	failedAttemptCount      int
	state                   pairingAttemptState
	createdAt               time.Time
	expiresAt               time.Time
	updatedAt               time.Time
	lockedAt                time.Time
}

// HashPairingBootstrapCredential binds one exact 256-bit credential to its canonical
// pairing attempt. Callers retain ownership of the raw credential and must never persist it.
func HashPairingBootstrapCredential(encodedID string, credential []byte) ([sha256.Size]byte, error) {
	attemptID, err := ids.Parse(encodedID)
	if err != nil || len(credential) != pairingBootstrapCredentialLen || allZero(credential) {
		return [sha256.Size]byte{}, ErrInvalidPairingBootstrapCredential
	}

	var input [pairingBootstrapHashInputLen]byte
	offset := copy(input[:], pairingBootstrapCredentialDomain)
	offset++
	offset += copy(input[offset:], attemptID.String())
	offset++
	copy(input[offset:], credential)
	defer clear(input[:])

	return sha256.Sum256(input[:]), nil
}

// NewPairingAttempt validates and normalizes one fresh, unclaimed pairing attempt.
func NewPairingAttempt(spec PairingAttemptSpec) (PairingAttempt, error) {
	id, err := ids.Parse(spec.ID)
	if err != nil {
		return PairingAttempt{}, fmt.Errorf("%w: id", ErrInvalidPairingAttempt)
	}
	publicKeyBytes, err := spec.AgentPublicKey.Bytes()
	if err != nil || allZero(publicKeyBytes) {
		return PairingAttempt{}, fmt.Errorf("%w: agent public key", ErrInvalidPairingAttempt)
	}
	fingerprint, err := cryptox.FingerprintX25519PublicKey(spec.AgentPublicKey)
	if err != nil {
		return PairingAttempt{}, fmt.Errorf("%w: agent fingerprint", ErrInvalidPairingAttempt)
	}
	displayName, ok := normalizedPairingText(
		spec.AgentDisplayName, maxPairingAgentNameRunes, maxPairingAgentNameBytes,
	)
	if !ok {
		return PairingAttempt{}, fmt.Errorf("%w: agent display name", ErrInvalidPairingAttempt)
	}
	version, ok := normalizedPairingText(spec.AgentVersion, maxPairingAgentVersionBytes, maxPairingAgentVersionBytes)
	if !ok {
		return PairingAttempt{}, fmt.Errorf("%w: agent version", ErrInvalidPairingAttempt)
	}
	if spec.ProtocolVersion != pairingAttemptProtocolVersion {
		return PairingAttempt{}, fmt.Errorf("%w: protocol version", ErrInvalidPairingAttempt)
	}
	if !validRelayRegion(spec.RelayRegion) {
		return PairingAttempt{}, fmt.Errorf("%w: relay region", ErrInvalidPairingAttempt)
	}
	if len(spec.BootstrapCredentialHash) != pairingAttemptBootstrapHashLen ||
		allZero(spec.BootstrapCredentialHash) {
		return PairingAttempt{}, fmt.Errorf("%w: bootstrap credential hash", ErrInvalidPairingAttempt)
	}
	createdAt := spec.CreatedAt.UTC().Truncate(time.Microsecond)
	expiresAt := spec.ExpiresAt.UTC().Truncate(time.Microsecond)
	if createdAt.IsZero() || expiresAt.IsZero() || !expiresAt.After(createdAt) ||
		expiresAt.Sub(createdAt) > maxPairingAttemptLifetime {
		return PairingAttempt{}, fmt.Errorf("%w: lifetime", ErrInvalidPairingAttempt)
	}

	attempt := PairingAttempt{
		id: id, agentPublicKey: spec.AgentPublicKey, agentFingerprint: fingerprint,
		agentDisplayName: displayName, agentVersion: version,
		protocolVersion: spec.ProtocolVersion, relayRegion: spec.RelayRegion,
		state: pairingAttemptStateOpen, createdAt: createdAt, expiresAt: expiresAt, updatedAt: createdAt,
	}
	copy(attempt.bootstrapCredentialHash[:], spec.BootstrapCredentialHash)
	return attempt, nil
}

func (attempt PairingAttempt) validForCreate() bool {
	if attempt.state != pairingAttemptStateOpen || attempt.failedAttemptCount != 0 ||
		!attempt.updatedAt.Equal(attempt.createdAt) || !attempt.lockedAt.IsZero() {
		return false
	}
	hash := make([]byte, pairingAttemptBootstrapHashLen)
	copy(hash, attempt.bootstrapCredentialHash[:])
	rebuilt, err := NewPairingAttempt(PairingAttemptSpec{
		ID: attempt.id.String(), AgentPublicKey: attempt.agentPublicKey,
		AgentDisplayName: attempt.agentDisplayName, AgentVersion: attempt.agentVersion,
		ProtocolVersion: attempt.protocolVersion, RelayRegion: attempt.relayRegion,
		BootstrapCredentialHash: hash, CreatedAt: attempt.createdAt, ExpiresAt: attempt.expiresAt,
	})
	return err == nil && rebuilt.agentFingerprint.Equal(attempt.agentFingerprint) &&
		rebuilt.agentDisplayName == attempt.agentDisplayName && rebuilt.agentVersion == attempt.agentVersion
}

func normalizedPairingText(value string, maxRunes int, maxBytes int) (string, bool) {
	if len(value) > maxBytes || !utf8.ValidString(value) || strings.ContainsFunc(value, unsafePairingTextRune) {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > maxRunes {
		return "", false
	}
	return value, true
}

func unsafePairingTextRune(value rune) bool {
	return unicode.IsControl(value) || unicode.In(value, unicode.Cf, unicode.Zl, unicode.Zp)
}

func allZero(value []byte) bool {
	return len(value) > 0 && bytes.Equal(value, make([]byte, len(value)))
}
