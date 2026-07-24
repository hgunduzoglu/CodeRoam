package cryptox

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestFingerprintX25519PublicKeyMatchesGoldenVector(t *testing.T) {
	encoded := make([]byte, x25519PublicKeySize)
	for index := range encoded {
		encoded[index] = byte(index)
	}
	key, err := ParseX25519PublicKey(encoded)
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}

	fingerprint, err := FingerprintX25519PublicKey(key)
	if err != nil {
		t.Fatalf("FingerprintX25519PublicKey() error = %v", err)
	}
	got, err := fingerprint.String()
	if err != nil {
		t.Fatalf("String() error = %v", err)
	}

	const want = "x25519-sha256:630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}

	parsed, err := ParseX25519Fingerprint(want)
	if err != nil {
		t.Fatalf("ParseX25519Fingerprint() error = %v", err)
	}
	if !parsed.Equal(fingerprint) {
		t.Fatal("ParseX25519Fingerprint() did not preserve the fingerprint")
	}
}

func TestFingerprintX25519PublicKeyRejectsUninitializedKey(t *testing.T) {
	var key X25519PublicKey

	if _, err := FingerprintX25519PublicKey(key); !errors.Is(err, ErrInvalidFingerprint) {
		t.Fatalf(
			"FingerprintX25519PublicKey() error = %v, want %v",
			err,
			ErrInvalidFingerprint,
		)
	}
}

func TestX25519FingerprintEqualFailsClosed(t *testing.T) {
	const firstEncoded = "x25519-sha256:630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd"
	const secondEncoded = "x25519-sha256:730dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd"

	first, err := ParseX25519Fingerprint(firstEncoded)
	if err != nil {
		t.Fatalf("ParseX25519Fingerprint(first) error = %v", err)
	}
	firstCopy, err := ParseX25519Fingerprint(firstEncoded)
	if err != nil {
		t.Fatalf("ParseX25519Fingerprint(first copy) error = %v", err)
	}
	second, err := ParseX25519Fingerprint(secondEncoded)
	if err != nil {
		t.Fatalf("ParseX25519Fingerprint(second) error = %v", err)
	}
	var zero X25519Fingerprint

	if !first.Equal(firstCopy) {
		t.Fatal("Equal() = false for identical initialized fingerprints")
	}
	if first.Equal(second) {
		t.Fatal("Equal() = true for different initialized fingerprints")
	}
	if zero.Equal(zero) || zero.Equal(first) || first.Equal(zero) {
		t.Fatal("Equal() = true for a comparison containing an uninitialized fingerprint")
	}
	if reflect.TypeOf(zero).Comparable() {
		t.Fatal("X25519Fingerprint remains comparable with ==")
	}
}

func TestParseX25519FingerprintRejectsNoncanonicalInput(t *testing.T) {
	const canonical = "x25519-sha256:630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd"

	tests := map[string]string{
		"empty":            "",
		"missing prefix":   canonical[len(x25519FingerprintPrefix):],
		"wrong prefix":     "ed25519-sha256:" + canonical[len(x25519FingerprintPrefix):],
		"short digest":     canonical[:len(canonical)-1],
		"long digest":      canonical + "0",
		"uppercase digest": x25519FingerprintPrefix + strings.ToUpper(canonical[len(x25519FingerprintPrefix):]),
		"non-hex digest":   canonical[:len(canonical)-1] + "g",
	}

	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseX25519Fingerprint(encoded); !errors.Is(err, ErrInvalidFingerprint) {
				t.Fatalf(
					"ParseX25519Fingerprint() error = %v, want %v",
					err,
					ErrInvalidFingerprint,
				)
			}
		})
	}
}
