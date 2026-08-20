//go:build linux || darwin

package identity

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestValidateDirectoryStatRejectsForeignOwner(t *testing.T) {
	stat := unix.Stat_t{
		Mode: unix.S_IFDIR | 0o700,
		Uid:  uint32(os.Geteuid() + 1),
	}
	if err := validateDirectoryStat(-1, &stat); !errors.Is(err, ErrInsecureStorage) {
		t.Fatalf("validateDirectoryStat() error = %v, want %v", err, ErrInsecureStorage)
	}
}
