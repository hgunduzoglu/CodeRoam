// Package identity persists the workspace agent's stable Noise identity.
package identity

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
)

const (
	identityStorageVersion = 1
	maxIdentityFileSize    = 1024
)

var (
	// ErrIdentityNotFound indicates that explicit initialization is required.
	ErrIdentityNotFound = errors.New("agent identity: not found")
	// ErrIdentityExists prevents creation from replacing a pinned identity.
	ErrIdentityExists = errors.New("agent identity: already exists")
	// ErrInvalidIdentity indicates corrupt or unsupported persisted key material.
	ErrInvalidIdentity = errors.New("agent identity: invalid")
	// ErrInsecureStorage indicates permissions or file types that could expose key material.
	ErrInsecureStorage = errors.New("agent identity: insecure storage")
)

type storedIdentity struct {
	Version    int    `json:"version"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type fileIdentity struct {
	privateKey *ecdh.PrivateKey
	publicKey  cryptox.X25519PublicKey
}

func (i *fileIdentity) PublicKey() cryptox.X25519PublicKey {
	return i.publicKey
}

// FileProvider keeps normal loading separate from explicit first-time creation.
type FileProvider struct {
	path               string
	random             io.Reader
	afterDirectoryOpen func()
	afterPendingSync   func() error
	afterPublish       func() error
}

var _ cryptox.IdentityProvider = (*FileProvider)(nil)

// NewFileProvider validates an absolute, canonical identity-file path without touching disk.
func NewFileProvider(path string) (*FileProvider, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, ErrInvalidIdentity
	}
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) {
		return nil, ErrInvalidIdentity
	}
	resolved, err := resolveIdentityPath(path)
	if err != nil {
		return nil, err
	}
	return &FileProvider{path: resolved, random: rand.Reader}, nil
}

// Load restores an existing identity and never creates or replaces key material.
func (p *FileProvider) Load(ctx context.Context) (cryptox.StaticIdentity, error) {
	if p == nil || ctx == nil {
		return nil, ErrInvalidIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load agent identity: %w", err)
	}
	directory, err := openIdentityDirectory(filepath.Dir(p.path), false)
	if err != nil {
		return nil, err
	}
	defer directory.close()
	if p.afterDirectoryOpen != nil {
		p.afterDirectoryOpen()
	}

	encoded, err := directory.readEntry(filepath.Base(p.path))
	if err != nil {
		return nil, err
	}
	identity, err := parseIdentity(encoded)
	if err != nil {
		return nil, err
	}
	if err := directory.ensurePathStillBound(); err != nil {
		return nil, err
	}
	return identity, nil
}

// Create generates and atomically persists a new identity, failing if any entry already exists.
func (p *FileProvider) Create(ctx context.Context) (cryptox.StaticIdentity, error) {
	if p == nil || ctx == nil {
		return nil, ErrInvalidIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("create agent identity: %w", err)
	}
	directory, err := openIdentityDirectory(filepath.Dir(p.path), true)
	if err != nil {
		return nil, err
	}
	defer directory.close()
	if err := directory.lockExclusive(ctx); err != nil {
		return nil, err
	}
	if p.afterDirectoryOpen != nil {
		p.afterDirectoryOpen()
	}

	identityName := filepath.Base(p.path)
	pendingName := "." + identityName + ".pending"
	if existing, exists, err := directory.entryStat(identityName); err != nil {
		return nil, err
	} else if exists {
		if err := removeMatchingPending(directory, identityName, pendingName, existing); err != nil {
			return nil, err
		}
		return nil, ErrIdentityExists
	}
	if _, exists, err := directory.entryStat(pendingName); err != nil {
		return nil, err
	} else if exists {
		return p.recoverPending(ctx, directory, identityName, pendingName)
	}

	privateKey, err := ecdh.X25519().GenerateKey(p.random)
	if err != nil {
		return nil, fmt.Errorf("generate agent identity: %w", err)
	}
	publicBytes := privateKey.PublicKey().Bytes()
	publicKey, err := cryptox.ParseX25519PublicKey(publicBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: generated public key", ErrInvalidIdentity)
	}
	privateBytes := privateKey.Bytes()
	defer clear(privateBytes)
	record := storedIdentity{
		Version:    identityStorageVersion,
		PrivateKey: base64.RawStdEncoding.EncodeToString(privateBytes),
		PublicKey:  base64.RawStdEncoding.EncodeToString(publicBytes),
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode agent identity: %w", err)
	}
	defer clear(encoded)

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("create agent identity: %w", err)
	}
	if err := directory.ensurePathStillBound(); err != nil {
		return nil, err
	}
	if err := directory.writePending(pendingName, encoded); err != nil {
		return nil, err
	}
	if err := directory.sync(); err != nil {
		return nil, err
	}
	if p.afterPendingSync != nil {
		if err := p.afterPendingSync(); err != nil {
			return nil, fmt.Errorf("after pending identity sync: %w", err)
		}
	}
	if err := directory.ensurePathStillBound(); err != nil {
		return nil, err
	}
	if err := directory.publish(pendingName, identityName); err != nil {
		return nil, err
	}
	if p.afterPublish != nil {
		if err := p.afterPublish(); err != nil {
			return nil, fmt.Errorf("after identity publication: %w", err)
		}
	}
	if err := directory.ensurePathStillBound(); err != nil {
		return nil, err
	}
	if err := directory.remove(pendingName); err != nil {
		return nil, err
	}
	if err := directory.sync(); err != nil {
		return nil, err
	}
	return &fileIdentity{privateKey: privateKey, publicKey: publicKey}, nil
}

func resolveIdentityPath(path string) (string, error) {
	current := filepath.Dir(path)
	missing := []string{filepath.Base(path)}
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for _, component := range missing {
				resolved = filepath.Join(resolved, component)
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", ErrInvalidIdentity
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrInvalidIdentity
		}
		missing = append([]string{filepath.Base(current)}, missing...)
		current = parent
	}
}

func (p *FileProvider) recoverPending(
	ctx context.Context,
	directory *secureDirectory,
	identityName string,
	pendingName string,
) (cryptox.StaticIdentity, error) {
	encoded, err := directory.readEntry(pendingName)
	if err != nil {
		return nil, err
	}
	identity, err := parseIdentity(encoded)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("recover agent identity: %w", err)
	}
	if err := directory.ensurePathStillBound(); err != nil {
		return nil, err
	}
	if err := directory.publish(pendingName, identityName); err != nil {
		return nil, err
	}
	if err := directory.remove(pendingName); err != nil {
		return nil, err
	}
	if err := directory.sync(); err != nil {
		return nil, err
	}
	return identity, nil
}

func removeMatchingPending(
	directory *secureDirectory,
	identityName string,
	pendingName string,
	identityStat unixStat,
) error {
	pendingStat, exists, err := directory.entryStat(pendingName)
	if err != nil || !exists {
		return err
	}
	if !sameUnixFile(identityStat, pendingStat) {
		return ErrInvalidIdentity
	}
	if err := directory.remove(pendingName); err != nil {
		return err
	}
	return directory.sync()
}

func parseIdentity(encoded []byte) (*fileIdentity, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var record storedIdentity
	if err := decoder.Decode(&record); err != nil {
		return nil, ErrInvalidIdentity
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidIdentity
	}
	if record.Version != identityStorageVersion {
		return nil, ErrInvalidIdentity
	}
	canonical, err := json.Marshal(record)
	if err != nil || !bytes.Equal(canonical, encoded) {
		return nil, ErrInvalidIdentity
	}

	privateBytes, err := base64.RawStdEncoding.DecodeString(record.PrivateKey)
	if err != nil ||
		len(privateBytes) != 32 ||
		base64.RawStdEncoding.EncodeToString(privateBytes) != record.PrivateKey {
		return nil, ErrInvalidIdentity
	}
	defer clear(privateBytes)
	publicBytes, err := base64.RawStdEncoding.DecodeString(record.PublicKey)
	if err != nil ||
		len(publicBytes) != 32 ||
		base64.RawStdEncoding.EncodeToString(publicBytes) != record.PublicKey {
		return nil, ErrInvalidIdentity
	}
	privateKey, err := ecdh.X25519().NewPrivateKey(privateBytes)
	if err != nil || !bytes.Equal(privateKey.PublicKey().Bytes(), publicBytes) {
		return nil, ErrInvalidIdentity
	}
	publicKey, err := cryptox.ParseX25519PublicKey(publicBytes)
	if err != nil {
		return nil, ErrInvalidIdentity
	}
	return &fileIdentity{privateKey: privateKey, publicKey: publicKey}, nil
}
