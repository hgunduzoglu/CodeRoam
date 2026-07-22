package device

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

type verifierStub struct {
	userID auth.UserID
}

func (verifier verifierStub) Verify(context.Context, string) (auth.UserID, error) {
	return verifier.userID, nil
}

type userFinderStub struct {
	user auth.User
}

func (finder userFinderStub) FindByID(context.Context, auth.UserID) (auth.User, error) {
	return finder.user, nil
}

func TestNewDevice(t *testing.T) {
	actor := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	publicKey := newTestPublicKey(t, 0x42)
	pairedAt := time.Date(2026, time.July, 17, 18, 0, 0, 0, time.FixedZone("test", 3*60*60))

	device, err := NewDevice(
		actor,
		"1123456789abcdef0123456789abcdef",
		"  Husam's iPad  ",
		PlatformIPadOS,
		publicKey,
		pairedAt,
	)
	if err != nil {
		t.Fatalf("NewDevice() error = %v", err)
	}
	if device.id != "1123456789abcdef0123456789abcdef" || device.name != "Husam's iPad" ||
		device.platform != PlatformIPadOS || !device.publicKey.Equal(publicKey) {
		t.Fatal("NewDevice() did not preserve validated identity metadata")
	}
	if !device.pairedAt.Equal(pairedAt) || device.pairedAt.Location() != time.UTC {
		t.Fatalf("pairedAt = %v", device.pairedAt)
	}
	if !device.CanAuthorize(actor) {
		t.Fatal("new device did not authorize its owner")
	}

	tests := map[string]struct {
		actor     auth.Actor
		id        string
		name      string
		platform  Platform
		publicKey cryptox.X25519PublicKey
		pairedAt  time.Time
		want      error
	}{
		"zero actor": {
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Phone",
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrDeviceAccessDenied,
		},
		"empty id": {
			actor:     actor,
			name:      "Phone",
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"uppercase id": {
			actor:     actor,
			id:        "1123456789ABCDEF0123456789ABCDEF",
			name:      "Phone",
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"non hexadecimal id": {
			actor:     actor,
			id:        "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			name:      "Phone",
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"empty name": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"invalid name encoding": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			name:      string([]byte{0xff}),
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"name with control character": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Phone\nName",
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"oversized name": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			name:      strings.Repeat("a", maxDeviceNameRunes+1),
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"oversized blank name": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			name:      strings.Repeat(" ", maxDeviceNameBytes+1),
			platform:  PlatformIOS,
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"unknown platform": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Phone",
			platform:  Platform("desktop"),
			publicKey: publicKey,
			pairedAt:  pairedAt,
			want:      ErrInvalidDevice,
		},
		"zero public key": {
			actor:    actor,
			id:       "1123456789abcdef0123456789abcdef",
			name:     "Phone",
			platform: PlatformIOS,
			pairedAt: pairedAt,
			want:     ErrInvalidDevice,
		},
		"zero paired time": {
			actor:     actor,
			id:        "1123456789abcdef0123456789abcdef",
			name:      "Phone",
			platform:  PlatformIOS,
			publicKey: publicKey,
			want:      ErrInvalidDevice,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			device, err := NewDevice(
				test.actor,
				test.id,
				test.name,
				test.platform,
				test.publicKey,
				test.pairedAt,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewDevice() error = %v, want %v", err, test.want)
			}
			if device.CanAuthorize(actor) {
				t.Fatal("invalid device authorized an actor")
			}
		})
	}
}

func TestDeviceRevocationFailsClosed(t *testing.T) {
	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	foreignActor := newTestActor(t, "2123456789abcdef0123456789abcdef", "foreign@example.com")
	pairedAt := time.Date(2026, time.July, 17, 15, 0, 0, 0, time.UTC)
	device, err := NewDevice(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Phone",
		PlatformAndroid,
		newTestPublicKey(t, 0x42),
		pairedAt,
	)
	if err != nil {
		t.Fatalf("NewDevice() error = %v", err)
	}

	if device.CanAuthorize(auth.Actor{}) || device.CanAuthorize(foreignActor) {
		t.Fatal("device authorized a zero or foreign actor")
	}
	if err := device.Revoke(foreignActor, pairedAt.Add(time.Hour)); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("Revoke(foreign) error = %v, want ErrDeviceAccessDenied", err)
	}
	if !device.CanAuthorize(owner) || device.revocation.revokedAt != nil {
		t.Fatal("foreign revocation changed device state")
	}
	if err := device.Revoke(owner, pairedAt.Add(-time.Second)); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("Revoke(before pairing) error = %v, want ErrInvalidDevice", err)
	}
	if !device.CanAuthorize(owner) || device.revocation.revokedAt != nil {
		t.Fatal("invalid revocation time changed device state")
	}

	firstRevokedAt := pairedAt.Add(time.Hour)
	if err := device.Revoke(owner, firstRevokedAt); err != nil {
		t.Fatalf("Revoke(owner) error = %v", err)
	}
	if device.CanAuthorize(owner) || device.revocation.revokedAt == nil ||
		!device.revocation.revokedAt.Equal(firstRevokedAt) {
		t.Fatal("revoked device remained authorized or lost its revocation time")
	}
	if err := device.Revoke(owner, firstRevokedAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke(owner repeated) error = %v", err)
	}
	if !device.revocation.revokedAt.Equal(firstRevokedAt) {
		t.Fatal("repeated revocation replaced the original time")
	}

	var nilDevice *Device
	if err := nilDevice.Revoke(owner, firstRevokedAt); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("nil Device Revoke() error = %v, want ErrInvalidDevice", err)
	}
}

func TestDeviceCopiesShareRevocationState(t *testing.T) {
	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	pairedAt := time.Date(2026, time.July, 17, 15, 0, 0, 0, time.UTC)
	device, err := NewDevice(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Phone",
		PlatformIOS,
		newTestPublicKey(t, 0x42),
		pairedAt,
	)
	if err != nil {
		t.Fatalf("NewDevice() error = %v", err)
	}
	retainedCopy := device
	if err := device.Revoke(owner, pairedAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if retainedCopy.CanAuthorize(owner) {
		t.Fatal("device copy retained authorization after revocation")
	}
}

func TestDeviceAuthorizationIsRaceSafeDuringRevocation(t *testing.T) {
	owner := newTestActor(t, "0123456789abcdef0123456789abcdef", "owner@example.com")
	pairedAt := time.Date(2026, time.July, 17, 15, 0, 0, 0, time.UTC)
	device, err := NewDevice(
		owner,
		"1123456789abcdef0123456789abcdef",
		"Phone",
		PlatformAndroid,
		newTestPublicKey(t, 0x42),
		pairedAt,
	)
	if err != nil {
		t.Fatalf("NewDevice() error = %v", err)
	}
	retainedCopy := device
	start := make(chan struct{})
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for range 100 {
				_ = retainedCopy.CanAuthorize(owner)
			}
		}()
	}
	close(start)
	if err := device.Revoke(owner, pairedAt.Add(time.Hour)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	readers.Wait()
	if retainedCopy.CanAuthorize(owner) {
		t.Fatal("device copy authorized after concurrent revocation")
	}
}

func newTestActor(t *testing.T, encodedID, email string) auth.Actor {
	t.Helper()
	user, err := auth.NewUser(
		encodedID,
		email,
		"Test User",
		time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	userID, err := auth.ParseUserID(encodedID)
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	service, err := auth.NewService(userFinderStub{user: user}, verifierStub{userID: userID})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	actor, err := service.Authenticate(context.Background(), "test-evidence")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	return actor
}

func newTestPublicKey(t *testing.T, value byte) cryptox.X25519PublicKey {
	t.Helper()
	key, err := cryptox.ParseX25519PublicKey(bytes.Repeat([]byte{value}, 32))
	if err != nil {
		t.Fatalf("ParseX25519PublicKey() error = %v", err)
	}
	return key
}
