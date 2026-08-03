package cryptox

import (
	"context"
	"crypto/ecdh"
	"errors"
	"fmt"
)

const x25519PublicKeySize = 32

// ErrInvalidPublicKey indicates that public identity material is not a canonical, usable key.
var ErrInvalidPublicKey = errors.New("cryptox: invalid X25519 public key")

// X25519PublicKey is the public, non-secret portion of a static identity.
type X25519PublicKey struct {
	encoded      [x25519PublicKeySize]byte
	initialized  bool
	doNotCompare [0]func()
}

// ParseX25519PublicKey validates and copies an untrusted canonical public key.
func ParseX25519PublicKey(encoded []byte) (X25519PublicKey, error) {
	if len(encoded) != x25519PublicKeySize {
		return X25519PublicKey{}, fmt.Errorf(
			"%w: got %d bytes, want %d",
			ErrInvalidPublicKey,
			len(encoded),
			x25519PublicKeySize,
		)
	}
	if !isCanonicalX25519PublicKey(encoded) || !isUsableX25519PublicKey(encoded) {
		return X25519PublicKey{}, ErrInvalidPublicKey
	}

	var key X25519PublicKey
	copy(key.encoded[:], encoded)
	key.initialized = true
	return key, nil
}

// Bytes returns an independent copy of an initialized public key.
func (k X25519PublicKey) Bytes() ([]byte, error) {
	if !k.initialized ||
		!isCanonicalX25519PublicKey(k.encoded[:]) ||
		!isUsableX25519PublicKey(k.encoded[:]) {
		return nil, ErrInvalidPublicKey
	}

	encoded := make([]byte, x25519PublicKeySize)
	copy(encoded, k.encoded[:])
	return encoded, nil
}

func isCanonicalX25519PublicKey(encoded []byte) bool {
	if len(encoded) != x25519PublicKeySize || encoded[31] > 0x7f {
		return false
	}
	if encoded[31] < 0x7f {
		return true
	}
	for index := 30; index >= 1; index-- {
		if encoded[index] != 0xff {
			return true
		}
	}
	return encoded[0] < 0xed
}

func isUsableX25519PublicKey(encoded []byte) bool {
	curve := ecdh.X25519()
	validationScalar := [x25519PublicKeySize]byte{1}
	privateKey, err := curve.NewPrivateKey(validationScalar[:])
	if err != nil {
		return false
	}
	publicKey, err := curve.NewPublicKey(encoded)
	if err != nil {
		return false
	}
	_, err = privateKey.ECDH(publicKey)
	return err == nil
}

// Equal reports whether two initialized public keys have identical encodings.
func (k X25519PublicKey) Equal(other X25519PublicKey) bool {
	return k.initialized && other.initialized && k.encoded == other.encoded
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
