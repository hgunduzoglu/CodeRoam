package cryptox

import (
	"context"
	"errors"
	"fmt"
)

const x25519PublicKeySize = 32

// ErrInvalidPublicKey indicates that public identity material is not a valid encoded size.
var ErrInvalidPublicKey = errors.New("cryptox: invalid X25519 public key")

// X25519PublicKey is the public, non-secret portion of a static identity.
type X25519PublicKey [x25519PublicKeySize]byte

// ParseX25519PublicKey validates and copies untrusted encoded public-key bytes.
func ParseX25519PublicKey(encoded []byte) (X25519PublicKey, error) {
	if len(encoded) != x25519PublicKeySize {
		return X25519PublicKey{}, fmt.Errorf(
			"%w: got %d bytes, want %d",
			ErrInvalidPublicKey,
			len(encoded),
			x25519PublicKeySize,
		)
	}

	var key X25519PublicKey
	copy(key[:], encoded)
	return key, nil
}

// Bytes returns an independent copy of the encoded public key.
func (k X25519PublicKey) Bytes() []byte {
	encoded := make([]byte, x25519PublicKeySize)
	copy(encoded, k[:])
	return encoded
}

// StaticIdentity is an opaque private identity owned by an audited crypto adapter.
// Private-key bytes are intentionally not part of this interface.
type StaticIdentity interface {
	PublicKey() X25519PublicKey
}

// IdentityProvider loads the existing static identity or creates one only when none exists.
// Implementations must return errors for unreadable or corrupt identities instead of replacing
// them, because silent replacement would break peer key pinning.
type IdentityProvider interface {
	LoadOrCreate(context.Context) (StaticIdentity, error)
}
