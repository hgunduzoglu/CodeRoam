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
type X25519PublicKey struct {
	encoded     [x25519PublicKeySize]byte
	initialized bool
}

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
	copy(key.encoded[:], encoded)
	key.initialized = true
	return key, nil
}

// Bytes returns an independent copy of an initialized public key.
func (k X25519PublicKey) Bytes() ([]byte, error) {
	if !k.initialized {
		return nil, ErrInvalidPublicKey
	}

	encoded := make([]byte, x25519PublicKeySize)
	copy(encoded, k.encoded[:])
	return encoded, nil
}

// StaticIdentity is an opaque private identity owned by an audited crypto adapter.
// Private-key bytes are intentionally not part of this interface.
type StaticIdentity interface {
	PublicKey() X25519PublicKey
}

// IdentityProvider keeps normal identity loading separate from explicit first-time creation.
type IdentityProvider interface {
	// Load must return an error when the identity is absent, unreadable, or corrupt. It must never
	// create or replace an identity because silent replacement would break peer key pinning.
	Load(context.Context) (StaticIdentity, error)

	// Create must persist a new identity atomically and fail if an identity already exists.
	Create(context.Context) (StaticIdentity, error)
}
