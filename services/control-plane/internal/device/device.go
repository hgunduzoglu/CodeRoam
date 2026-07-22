package device

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

const (
	encodedDeviceIDLength = 32
	maxDeviceNameRunes    = 128
	maxDeviceNameBytes    = maxDeviceNameRunes * utf8.UTFMax
)

var (
	ErrInvalidDevice      = errors.New("invalid device")
	ErrDeviceAccessDenied = errors.New("device access denied")
)

type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformIPadOS  Platform = "ipados"
	PlatformAndroid Platform = "android"
)

type Device struct {
	id         string
	ownerID    auth.UserID
	name       string
	platform   Platform
	publicKey  cryptox.X25519PublicKey
	pairedAt   time.Time
	revocation *revocationState
}

type revocationState struct {
	mu        sync.RWMutex
	revokedAt *time.Time
}

func NewDevice(
	actor auth.Actor,
	id string,
	name string,
	platform Platform,
	publicKey cryptox.X25519PublicKey,
	pairedAt time.Time,
) (Device, error) {
	ownerID, ok := actor.UserID()
	if !ok {
		return Device{}, ErrDeviceAccessDenied
	}
	if len(id) != encodedDeviceIDLength || id != strings.ToLower(id) {
		return Device{}, fmt.Errorf("%w: id", ErrInvalidDevice)
	}
	if _, err := hex.DecodeString(id); err != nil {
		return Device{}, fmt.Errorf("%w: id", ErrInvalidDevice)
	}

	if len(name) > maxDeviceNameBytes || !utf8.ValidString(name) {
		return Device{}, fmt.Errorf("%w: name", ErrInvalidDevice)
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsFunc(name, unicode.IsControl) ||
		utf8.RuneCountInString(name) > maxDeviceNameRunes {
		return Device{}, fmt.Errorf("%w: name", ErrInvalidDevice)
	}
	if platform != PlatformIOS && platform != PlatformIPadOS && platform != PlatformAndroid {
		return Device{}, fmt.Errorf("%w: platform", ErrInvalidDevice)
	}
	if _, err := publicKey.Bytes(); err != nil {
		return Device{}, fmt.Errorf("%w: public key", ErrInvalidDevice)
	}
	if pairedAt.IsZero() {
		return Device{}, fmt.Errorf("%w: paired time", ErrInvalidDevice)
	}

	return Device{
		id:         id,
		ownerID:    ownerID,
		name:       name,
		platform:   platform,
		publicKey:  publicKey,
		pairedAt:   pairedAt.UTC(),
		revocation: &revocationState{},
	}, nil
}

func (device Device) CanAuthorize(actor auth.Actor) bool {
	actorID, ok := actor.UserID()
	if !ok || device.ownerID.String() == "" || device.ownerID != actorID || device.revocation == nil {
		return false
	}
	device.revocation.mu.RLock()
	defer device.revocation.mu.RUnlock()
	return device.revocation.revokedAt == nil
}

func (device *Device) Revoke(actor auth.Actor, revokedAt time.Time) error {
	if device == nil || device.revocation == nil {
		return ErrInvalidDevice
	}
	actorID, ok := actor.UserID()
	if !ok || device.ownerID.String() == "" || device.ownerID != actorID {
		return ErrDeviceAccessDenied
	}
	device.revocation.mu.Lock()
	defer device.revocation.mu.Unlock()
	if device.revocation.revokedAt != nil {
		return nil
	}
	if revokedAt.IsZero() || revokedAt.Before(device.pairedAt) {
		return fmt.Errorf("%w: revoked time", ErrInvalidDevice)
	}

	normalizedRevokedAt := revokedAt.UTC()
	device.revocation.revokedAt = &normalizedRevokedAt
	return nil
}
