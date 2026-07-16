package cryptox

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestParseX25519PublicKeyValidatesAndCopiesInput(t *testing.T) {
	encoded := bytes.Repeat([]byte{0x42}, x25519PublicKeySize)
	key, err := ParseX25519PublicKey(encoded)
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}

	encoded[0] = 0x99
	parsed, err := key.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	if got := parsed[0]; got != 0x42 {
		t.Fatalf("parsed key changed with input: first byte = %#x, want %#x", got, 0x42)
	}

	parsed[1] = 0x99
	returned, err := key.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	if got := returned[1]; got != 0x42 {
		t.Fatalf("key changed through Bytes result: second byte = %#x, want %#x", got, 0x42)
	}
}

func TestParseX25519PublicKeyRejectsInvalidLengths(t *testing.T) {
	for _, size := range []int{0, x25519PublicKeySize - 1, x25519PublicKeySize + 1} {
		t.Run("size_"+strconv.Itoa(size), func(t *testing.T) {
			_, err := ParseX25519PublicKey(make([]byte, size))
			if !errors.Is(err, ErrInvalidPublicKey) {
				t.Fatalf("ParseX25519PublicKey() error = %v, want %v", err, ErrInvalidPublicKey)
			}
		})
	}
}

func TestX25519PublicKeyEqualFailsClosed(t *testing.T) {
	first, err := ParseX25519PublicKey(bytes.Repeat([]byte{0x42}, x25519PublicKeySize))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(first) error = %v", err)
	}
	firstCopy, err := ParseX25519PublicKey(bytes.Repeat([]byte{0x42}, x25519PublicKeySize))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(first copy) error = %v", err)
	}
	second, err := ParseX25519PublicKey(bytes.Repeat([]byte{0x43}, x25519PublicKeySize))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey(second) error = %v", err)
	}
	var zero X25519PublicKey

	if !first.Equal(firstCopy) {
		t.Fatal("Equal() = false for identical initialized keys")
	}
	if first.Equal(second) {
		t.Fatal("Equal() = true for different initialized keys")
	}
	if zero.Equal(zero) || zero.Equal(first) || first.Equal(zero) {
		t.Fatal("Equal() = true for a comparison containing an uninitialized key")
	}
	if reflect.TypeOf(zero).Comparable() {
		t.Fatal("X25519PublicKey remains comparable with ==")
	}
}
