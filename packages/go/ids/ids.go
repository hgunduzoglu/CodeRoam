package ids

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const encodedLength = 32

var ErrInvalid = errors.New("invalid opaque identifier")

type ID struct {
	encoded string
}

// New returns a cryptographically random 128-bit opaque identifier.
func New() (ID, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return ID{}, fmt.Errorf("generate id: %w", err)
	}
	return ID{encoded: hex.EncodeToString(raw[:])}, nil
}

func Parse(value string) (ID, error) {
	if len(value) != encodedLength || value != strings.ToLower(value) {
		return ID{}, ErrInvalid
	}
	if _, err := hex.DecodeString(value); err != nil {
		return ID{}, ErrInvalid
	}
	return ID{encoded: value}, nil
}

func (id ID) String() string {
	return id.encoded
}
