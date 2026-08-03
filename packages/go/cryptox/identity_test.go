package cryptox

import (
	"bytes"
	"encoding/hex"
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

func TestParseX25519PublicKeyRejectsNoncanonicalAndLowOrderInputs(t *testing.T) {
	tests := map[string]string{
		"zero":                  "0000000000000000000000000000000000000000000000000000000000000000",
		"one":                   "0100000000000000000000000000000000000000000000000000000000000000",
		"low order point one":   "e0eb7a7c3b41b8ae1656e3faf19fc46ada098deb9c32b1fd866205165f49b800",
		"low order point two":   "5f9c95bca3508c24b1d0b1559c83ef5b04445cc4581c8e86d8224eddd09f1157",
		"field prime minus one": "ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"field prime":           "edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"field prime plus one":  "eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"masked high bit alias": "0900000000000000000000000000000000000000000000000000000000000080",
	}

	for name, encodedHex := range tests {
		t.Run(name, func(t *testing.T) {
			encoded, err := hex.DecodeString(encodedHex)
			if err != nil {
				t.Fatalf("decode test key: %v", err)
			}
			if _, err := ParseX25519PublicKey(encoded); !errors.Is(err, ErrInvalidPublicKey) {
				t.Fatalf("ParseX25519PublicKey() error = %v, want %v", err, ErrInvalidPublicKey)
			}
		})
	}
}

func TestX25519PublicKeyBytesRevalidatesInitializedEncoding(t *testing.T) {
	key := X25519PublicKey{initialized: true}
	if _, err := key.Bytes(); !errors.Is(err, ErrInvalidPublicKey) {
		t.Fatalf("Bytes() error = %v, want %v", err, ErrInvalidPublicKey)
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
