package cryptox

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	x25519FingerprintPrefix = "x25519-sha256:"
	x25519FingerprintSize   = len(x25519FingerprintPrefix) + sha256.Size*2
)

// ErrInvalidFingerprint indicates that an X25519 fingerprint is absent or noncanonical.
var ErrInvalidFingerprint = errors.New("cryptox: invalid X25519 fingerprint")

// X25519Fingerprint is a canonical SHA-256 fingerprint of an X25519 public key.
type X25519Fingerprint struct {
	digest       [sha256.Size]byte
	initialized  bool
	doNotCompare [0]func()
}

// FingerprintX25519PublicKey computes the canonical fingerprint of an initialized public key.
func FingerprintX25519PublicKey(key X25519PublicKey) (X25519Fingerprint, error) {
	encoded, err := key.Bytes()
	if err != nil {
		return X25519Fingerprint{}, fmt.Errorf("%w: %v", ErrInvalidFingerprint, err)
	}

	return X25519Fingerprint{
		digest:      sha256.Sum256(encoded),
		initialized: true,
	}, nil
}

// ParseX25519Fingerprint validates and parses an exact canonical fingerprint.
func ParseX25519Fingerprint(encoded string) (X25519Fingerprint, error) {
	if len(encoded) != x25519FingerprintSize || !strings.HasPrefix(encoded, x25519FingerprintPrefix) {
		return X25519Fingerprint{}, ErrInvalidFingerprint
	}

	digestText := encoded[len(x25519FingerprintPrefix):]
	for _, character := range digestText {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return X25519Fingerprint{}, ErrInvalidFingerprint
		}
	}

	var fingerprint X25519Fingerprint
	if _, err := hex.Decode(fingerprint.digest[:], []byte(digestText)); err != nil {
		return X25519Fingerprint{}, fmt.Errorf("%w: %v", ErrInvalidFingerprint, err)
	}
	fingerprint.initialized = true
	return fingerprint, nil
}

// String returns the canonical encoding of an initialized fingerprint.
func (f X25519Fingerprint) String() (string, error) {
	if !f.initialized {
		return "", ErrInvalidFingerprint
	}

	return x25519FingerprintPrefix + hex.EncodeToString(f.digest[:]), nil
}

// Equal reports whether two initialized fingerprints have identical digests.
func (f X25519Fingerprint) Equal(other X25519Fingerprint) bool {
	return f.initialized && other.initialized && f.digest == other.digest
}
