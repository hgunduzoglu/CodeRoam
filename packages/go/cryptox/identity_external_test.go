package cryptox_test

import (
	"errors"
	"testing"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
)

func TestX25519PublicKeyZeroValueCannotBeEncoded(t *testing.T) {
	var key cryptox.X25519PublicKey
	if _, err := key.Bytes(); !errors.Is(err, cryptox.ErrInvalidPublicKey) {
		t.Fatalf("Bytes() error = %v, want %v", err, cryptox.ErrInvalidPublicKey)
	}
}

func TestX25519FingerprintZeroValueCannotBeEncoded(t *testing.T) {
	var fingerprint cryptox.X25519Fingerprint
	if _, err := fingerprint.String(); !errors.Is(err, cryptox.ErrInvalidFingerprint) {
		t.Fatalf("String() error = %v, want %v", err, cryptox.ErrInvalidFingerprint)
	}
}
