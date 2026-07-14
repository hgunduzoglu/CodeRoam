package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// New returns a 128-bit opaque identifier encoded as lowercase hexadecimal.
// Domain packages may wrap this string in stronger local types.
func New() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
