package identity

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileProviderCreatesAndLoadsStableOwnerOnlyIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "identity.json")
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}

	created, err := provider.Create(context.Background())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	loaded, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !created.PublicKey().Equal(loaded.PublicKey()) {
		t.Fatal("Load() changed the persisted public identity")
	}
	assertMode(t, filepath.Dir(path), 0o700)
	assertMode(t, path, 0o600)
}

func TestFileProviderNeverCreatesOrReplacesDuringLoadOrCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "identity.json")
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	if _, err := provider.Load(context.Background()); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("Load() missing error = %v, want %v", err, ErrIdentityNotFound)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() created an identity entry: %v", err)
	}

	first, err := provider.Create(context.Background())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first identity: %v", err)
	}
	if _, err := provider.Create(context.Background()); !errors.Is(err, ErrIdentityExists) {
		t.Fatalf("second Create() error = %v, want %v", err, ErrIdentityExists)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read retained identity: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("second Create() replaced persisted key material")
	}
	loaded, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() after duplicate create error = %v", err)
	}
	if !first.PublicKey().Equal(loaded.PublicKey()) {
		t.Fatal("duplicate Create() changed the active identity")
	}
}

func TestFileProviderRejectsCorruptIdentityWithoutReplacingIt(t *testing.T) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	valid := storedIdentity{
		Version:    identityStorageVersion,
		PrivateKey: base64.RawStdEncoding.EncodeToString(privateKey.Bytes()),
		PublicKey:  base64.RawStdEncoding.EncodeToString(privateKey.PublicKey().Bytes()),
	}
	canonical := mustJSON(t, valid)
	versionField := []byte(`"version":1`)
	privateField := []byte(`"private_key":"` + valid.PrivateKey + `"`)
	publicField := []byte(`"public_key":"` + valid.PublicKey + `"`)
	mismatchedPublic := make([]byte, 32)
	mismatchedPublic[0] = 1

	tests := map[string][]byte{
		"truncated JSON": []byte(`{"version":1`),
		"unknown field":  append(canonical[:len(canonical)-1], []byte(`,"extra":true}`)...),
		"duplicate version": bytes.Replace(
			canonical,
			versionField,
			append(append([]byte(nil), versionField...), append([]byte(","), versionField...)...),
			1,
		),
		"duplicate private key": bytes.Replace(
			canonical,
			privateField,
			append(append([]byte(nil), privateField...), append([]byte(","), privateField...)...),
			1,
		),
		"duplicate public key": bytes.Replace(
			canonical,
			publicField,
			append(append([]byte(nil), publicField...), append([]byte(","), publicField...)...),
			1,
		),
		"wrong version": mustJSON(t, storedIdentity{
			Version:    2,
			PrivateKey: valid.PrivateKey,
			PublicKey:  valid.PublicKey,
		}),
		"bad private encoding": mustJSON(t, storedIdentity{
			Version:    identityStorageVersion,
			PrivateKey: "***",
			PublicKey:  valid.PublicKey,
		}),
		"mismatched public key": mustJSON(t, storedIdentity{
			Version:    identityStorageVersion,
			PrivateKey: valid.PrivateKey,
			PublicKey:  base64.RawStdEncoding.EncodeToString(mismatchedPublic),
		}),
		"trailing document": append(mustJSON(t, valid), []byte(` {}`)...),
		"oversized file":    bytes.Repeat([]byte{'x'}, maxIdentityFileSize+1),
	}

	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "identity", "identity.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatalf("create identity directory: %v", err)
			}
			if err := os.WriteFile(path, encoded, 0o600); err != nil {
				t.Fatalf("write corrupt identity: %v", err)
			}
			provider, err := NewFileProvider(path)
			if err != nil {
				t.Fatalf("NewFileProvider() error = %v", err)
			}
			if _, err := provider.Load(context.Background()); !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("Load() error = %v, want %v", err, ErrInvalidIdentity)
			}
			if _, err := provider.Create(context.Background()); !errors.Is(err, ErrIdentityExists) {
				t.Fatalf("Create() error = %v, want %v", err, ErrIdentityExists)
			}
			retained, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read retained corrupt identity: %v", err)
			}
			if string(retained) != string(encoded) {
				t.Fatal("Create() replaced corrupt identity material")
			}
		})
	}
}

func TestFileProviderRejectsInsecureFileAndDirectory(t *testing.T) {
	t.Run("file permissions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "identity", "identity.json")
		provider := createTestIdentity(t, path)
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("chmod identity: %v", err)
		}
		if _, err := provider.Load(context.Background()); !errors.Is(err, ErrInsecureStorage) {
			t.Fatalf("Load() error = %v, want %v", err, ErrInsecureStorage)
		}
	})

	t.Run("directory permissions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "identity", "identity.json")
		provider := createTestIdentity(t, path)
		if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("chmod identity directory: %v", err)
		}
		if _, err := provider.Load(context.Background()); !errors.Is(err, ErrInsecureStorage) {
			t.Fatalf("Load() error = %v, want %v", err, ErrInsecureStorage)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		targetPath := filepath.Join(root, "target", "identity.json")
		_ = createTestIdentity(t, targetPath)
		linkDirectory := filepath.Join(root, "link")
		if err := os.Mkdir(linkDirectory, 0o700); err != nil {
			t.Fatalf("create link directory: %v", err)
		}
		linkPath := filepath.Join(linkDirectory, "identity.json")
		if err := os.Symlink(targetPath, linkPath); err != nil {
			t.Fatalf("create identity symlink: %v", err)
		}
		provider, err := NewFileProvider(linkPath)
		if err != nil {
			t.Fatalf("NewFileProvider() error = %v", err)
		}
		if _, err := provider.Load(context.Background()); !errors.Is(err, ErrInsecureStorage) {
			t.Fatalf("Load() error = %v, want %v", err, ErrInsecureStorage)
		}
	})
}

func TestFileProviderConcurrentCreatePublishesExactlyOneIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "identity.json")
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}

	const attempts = 16
	var successes atomic.Int32
	var unexpected atomic.Int32
	var wait sync.WaitGroup
	wait.Add(attempts)
	for range attempts {
		go func() {
			defer wait.Done()
			if _, err := provider.Create(context.Background()); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrIdentityExists) {
				unexpected.Add(1)
			}
		}()
	}
	wait.Wait()
	if successes.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf(
			"concurrent Create() successes=%d unexpected_errors=%d, want 1 and 0",
			successes.Load(),
			unexpected.Load(),
		)
	}
	if _, err := provider.Load(context.Background()); err != nil {
		t.Fatalf("Load() after concurrent Create() error = %v", err)
	}
}

func TestFileProviderRecoversCrashSafePendingIdentity(t *testing.T) {
	t.Run("before publication", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "identity", "identity.json")
		crash := errors.New("simulated crash after pending sync")
		provider, err := NewFileProvider(path)
		if err != nil {
			t.Fatalf("NewFileProvider() error = %v", err)
		}
		provider.afterPendingSync = func() error { return crash }
		if _, err := provider.Create(context.Background()); !errors.Is(err, crash) {
			t.Fatalf("Create() error = %v, want %v", err, crash)
		}

		pendingPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".pending")
		pendingBytes, err := os.ReadFile(pendingPath)
		if err != nil {
			t.Fatalf("read pending identity: %v", err)
		}
		pendingIdentity, err := parseIdentity(pendingBytes)
		if err != nil {
			t.Fatalf("parse pending identity: %v", err)
		}
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("canonical identity unexpectedly exists: %v", err)
		}

		recovery, err := NewFileProvider(path)
		if err != nil {
			t.Fatalf("NewFileProvider(recovery) error = %v", err)
		}
		recovered, err := recovery.Create(context.Background())
		if err != nil {
			t.Fatalf("recovery Create() error = %v", err)
		}
		if !pendingIdentity.PublicKey().Equal(recovered.PublicKey()) {
			t.Fatal("recovery generated a second identity")
		}
		if _, err := os.Lstat(pendingPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("recovery retained pending identity: %v", err)
		}
	})

	t.Run("after publication", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "identity", "identity.json")
		crash := errors.New("simulated crash after publication")
		provider, err := NewFileProvider(path)
		if err != nil {
			t.Fatalf("NewFileProvider() error = %v", err)
		}
		provider.afterPublish = func() error { return crash }
		if _, err := provider.Create(context.Background()); !errors.Is(err, crash) {
			t.Fatalf("Create() error = %v, want %v", err, crash)
		}
		pendingPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".pending")

		recovery, err := NewFileProvider(path)
		if err != nil {
			t.Fatalf("NewFileProvider(recovery) error = %v", err)
		}
		before, err := recovery.Load(context.Background())
		if err != nil {
			t.Fatalf("Load() after publication crash error = %v", err)
		}
		if _, err := recovery.Create(context.Background()); !errors.Is(err, ErrIdentityExists) {
			t.Fatalf("cleanup Create() error = %v, want %v", err, ErrIdentityExists)
		}
		after, err := recovery.Load(context.Background())
		if err != nil {
			t.Fatalf("Load() after pending cleanup error = %v", err)
		}
		if !before.PublicKey().Equal(after.PublicKey()) {
			t.Fatal("post-publication recovery changed the identity")
		}
		if _, err := os.Lstat(pendingPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cleanup retained pending hard link: %v", err)
		}
	})
}

func TestFileProviderRejectsUnresolvedPendingIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "identity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create identity directory: %v", err)
	}
	pendingPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".pending")
	if err := os.WriteFile(pendingPath, []byte(`{"version":1`), 0o600); err != nil {
		t.Fatalf("write unresolved pending identity: %v", err)
	}
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	if _, err := provider.Create(context.Background()); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidIdentity)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Create() published a fresh identity: %v", err)
	}
	if retained, err := os.ReadFile(pendingPath); err != nil || string(retained) != `{"version":1` {
		t.Fatalf("Create() changed unresolved pending state: bytes=%q error=%v", retained, err)
	}
}

func TestFileProviderDetectsDirectorySwapAfterAnchoring(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "identity", "identity.json")
	provider := createTestIdentity(t, path)
	original, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(original) error = %v", err)
	}

	replacementPath := filepath.Join(root, "replacement", "identity.json")
	replacement := createTestIdentity(t, replacementPath)
	replacementIdentity, err := replacement.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(replacement) error = %v", err)
	}
	if original.PublicKey().Equal(replacementIdentity.PublicKey()) {
		t.Fatal("test identities unexpectedly match")
	}

	identityDirectory := filepath.Dir(path)
	backupDirectory := filepath.Join(root, "identity-original")
	replacementDirectory := filepath.Dir(replacementPath)
	provider.afterDirectoryOpen = func() {
		provider.afterDirectoryOpen = nil
		if err := os.Rename(identityDirectory, backupDirectory); err != nil {
			t.Fatalf("move original identity directory: %v", err)
		}
		if err := os.Rename(replacementDirectory, identityDirectory); err != nil {
			t.Fatalf("swap replacement identity directory: %v", err)
		}
	}
	if _, err := provider.Load(context.Background()); !errors.Is(err, ErrInsecureStorage) {
		t.Fatalf("Load() after directory swap error = %v, want %v", err, ErrInsecureStorage)
	}
}

func TestFileProviderRejectsWritableAncestor(t *testing.T) {
	root := t.TempDir()
	writableParent := filepath.Join(root, "shared")
	if err := os.Mkdir(writableParent, 0o700); err != nil {
		t.Fatalf("create writable ancestor: %v", err)
	}
	if err := os.Chmod(writableParent, 0o770); err != nil {
		t.Fatalf("make ancestor group-writable: %v", err)
	}
	path := filepath.Join(writableParent, "identity", "identity.json")
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}

	if _, err := provider.Create(context.Background()); !errors.Is(err, ErrInsecureStorage) {
		t.Fatalf("Create() error = %v, want %v", err, ErrInsecureStorage)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Create() published identity below writable ancestor: %v", err)
	}
}

func TestFileProviderLockAcquisitionHonorsContextDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "identity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create identity directory: %v", err)
	}
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	held, err := openIdentityDirectory(filepath.Dir(provider.path), false)
	if err != nil {
		t.Fatalf("open held identity directory: %v", err)
	}
	defer held.close()
	if err := held.lockExclusive(context.Background()); err != nil {
		t.Fatalf("hold identity lock: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := provider.Create(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Create() error = %v, want %v", err, context.DeadlineExceeded)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Create() ignored lock deadline for %s", elapsed)
	}
	pendingPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".pending")
	for _, unexpected := range []string{path, pendingPath} {
		if _, err := os.Lstat(unexpected); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("deadline Create() touched %s: %v", unexpected, err)
		}
	}
}

func TestFileProviderCanceledCreateHasNoFilesystemSideEffect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "identity.json")
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := provider.Create(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want %v", err, context.Canceled)
	}
	if _, err := os.Lstat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled Create() touched identity storage: %v", err)
	}
}

func createTestIdentity(t *testing.T, path string) *FileProvider {
	t.Helper()
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return provider
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}

func mustJSON(t *testing.T, value storedIdentity) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal stored identity: %v", err)
	}
	return encoded
}
