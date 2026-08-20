package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

func TestRepositoryRegisterPairedRejectsInvalidBoundaries(t *testing.T) {
	ownerID, err := auth.ParseUserID("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	repository, err := NewRepository(&transactionStarterStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	deviceID := "1123456789abcdef0123456789abcdef"
	publicKey := newTestPublicKey(t, 0x42)

	var nilRepository *Repository
	if err := nilRepository.RegisterPaired(
		context.Background(), nil, ownerID, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("nil Repository RegisterPaired() error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	if err := repository.RegisterPaired(
		nil, nil, ownerID, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("RegisterPaired(nil context) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.RegisterPaired(
		canceledCtx, nil, ownerID, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("RegisterPaired(canceled context) error = %v, want context.Canceled", err)
	}
	if err := repository.RegisterPaired(
		context.Background(), nil, auth.UserID{}, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, ErrDeviceAccessDenied) {
		t.Fatalf("RegisterPaired(zero owner) error = %v, want ErrDeviceAccessDenied", err)
	}
	if err := repository.RegisterPaired(
		context.Background(), nil, ownerID, "not-a-device-id", "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("RegisterPaired(invalid id) error = %v, want ErrInvalidDevice", err)
	}
	if err := repository.RegisterPaired(
		context.Background(), nil, ownerID, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now.Add(time.Second),
	); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("RegisterPaired(future pairing) error = %v, want ErrInvalidDevice", err)
	}
	zeroClockRepository, err := NewRepository(&transactionStarterStub{}, func() time.Time { return time.Time{} })
	if err != nil {
		t.Fatalf("NewRepository(zero clock) error = %v", err)
	}
	if err := zeroClockRepository.RegisterPaired(
		context.Background(), nil, ownerID, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("RegisterPaired(zero clock) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
	if err := repository.RegisterPaired(
		context.Background(), nil, ownerID, deviceID, "Husam's iPhone", PlatformIOS, publicKey, now,
	); !errors.Is(err, ErrDevicePersistenceUnavailable) {
		t.Fatalf("RegisterPaired(nil transaction) error = %v, want ErrDevicePersistenceUnavailable", err)
	}
}
