//go:build darwin && cgo

package identity

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOpenIdentityDirectoryRejectsExtendedACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("create identity directory: %v", err)
	}
	if output, err := exec.Command("chmod", "+a", "everyone allow read", path).CombinedOutput(); err != nil {
		t.Fatalf("add test ACL: %v: %s", err, output)
	}

	if _, err := openIdentityDirectory(path, false); !errors.Is(err, ErrInsecureStorage) {
		t.Fatalf("openIdentityDirectory() error = %v, want %v", err, ErrInsecureStorage)
	}
}
