package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type identityVerifierStub struct {
	userID   UserID
	err      error
	calls    int
	evidence string
}

func (verifier *identityVerifierStub) Verify(_ context.Context, evidence string) (UserID, error) {
	verifier.calls++
	verifier.evidence = evidence
	return verifier.userID, verifier.err
}

type userFinderStub struct {
	user  User
	err   error
	calls int
}

func (finder *userFinderStub) FindByID(_ context.Context, _ UserID) (User, error) {
	finder.calls++
	return finder.user, finder.err
}

func TestNewServiceRequiresDependencies(t *testing.T) {
	verifier := &identityVerifierStub{}
	finder := &userFinderStub{}

	if service, err := NewService(nil, verifier); err == nil || service != nil {
		t.Fatalf("NewService(nil, verifier) = (%v, %v), want error", service, err)
	}
	if service, err := NewService(finder, nil); err == nil || service != nil {
		t.Fatalf("NewService(finder, nil) = (%v, %v), want error", service, err)
	}
}

func TestServiceAuthenticate(t *testing.T) {
	createdAt := time.Date(2026, time.July, 17, 17, 0, 0, 0, time.UTC)
	userID, err := ParseUserID("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	user, err := NewUser(userID.String(), "person@example.com", "Ada", createdAt)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	t.Run("returns actor for verified existing user", func(t *testing.T) {
		verifier := &identityVerifierStub{userID: userID}
		finder := &userFinderStub{user: user}
		service, err := NewService(finder, verifier)
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		actor, err := service.Authenticate(context.Background(), "opaque-evidence")
		if err != nil {
			t.Fatalf("Authenticate() error = %v", err)
		}
		actorUserID, ok := actor.UserID()
		if !ok || actorUserID != userID {
			t.Fatal("Authenticate() returned an actor with the wrong user")
		}
		if verifier.evidence != "opaque-evidence" || finder.calls != 1 {
			t.Fatal("Authenticate() did not use both authentication dependencies")
		}
	})

	t.Run("rejects malformed evidence before verification", func(t *testing.T) {
		for name, evidence := range map[string]string{
			"empty":        "",
			"whitespace":   "   ",
			"control":      "token\nvalue",
			"invalid utf8": string([]byte{0xff}),
			"oversized":    strings.Repeat("a", maxAuthenticationEvidenceBytes+1),
		} {
			t.Run(name, func(t *testing.T) {
				verifier := &identityVerifierStub{userID: userID}
				service, err := NewService(&userFinderStub{user: user}, verifier)
				if err != nil {
					t.Fatalf("NewService() error = %v", err)
				}
				actor, err := service.Authenticate(context.Background(), evidence)
				if !errors.Is(err, ErrUnauthenticated) {
					t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
				}
				if _, ok := actor.UserID(); ok || verifier.calls != 0 {
					t.Fatal("Authenticate() verified malformed evidence or returned an actor")
				}
			})
		}
	})

	t.Run("rejects identity and user mismatches", func(t *testing.T) {
		otherUser, err := NewUser(
			"1123456789abcdef0123456789abcdef",
			"other@example.com",
			"Other",
			createdAt,
		)
		if err != nil {
			t.Fatalf("NewUser() error = %v", err)
		}
		tests := map[string]struct {
			verifier        *identityVerifierStub
			finder          *userFinderStub
			wantFinderCalls int
		}{
			"rejected evidence": {
				verifier: &identityVerifierStub{err: ErrIdentityRejected},
				finder:   &userFinderStub{user: user},
			},
			"zero verified id": {
				verifier: &identityVerifierStub{},
				finder:   &userFinderStub{user: user},
			},
			"missing user": {
				verifier:        &identityVerifierStub{userID: userID},
				finder:          &userFinderStub{err: ErrUserNotFound},
				wantFinderCalls: 1,
			},
			"mismatched user": {
				verifier:        &identityVerifierStub{userID: userID},
				finder:          &userFinderStub{user: otherUser},
				wantFinderCalls: 1,
			},
		}
		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				service, err := NewService(test.finder, test.verifier)
				if err != nil {
					t.Fatalf("NewService() error = %v", err)
				}
				actor, err := service.Authenticate(context.Background(), "opaque-evidence")
				if !errors.Is(err, ErrUnauthenticated) {
					t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
				}
				if _, ok := actor.UserID(); ok {
					t.Fatal("Authenticate() returned an actor for an identity mismatch")
				}
				if test.finder.calls != test.wantFinderCalls {
					t.Fatalf("FindByID() calls = %d, want %d", test.finder.calls, test.wantFinderCalls)
				}
			})
		}
	})

	t.Run("sanitizes dependency failures", func(t *testing.T) {
		for name, test := range map[string]struct {
			verifier *identityVerifierStub
			finder   *userFinderStub
		}{
			"verifier": {
				verifier: &identityVerifierStub{err: errors.New("provider leaked secret-evidence")},
				finder:   &userFinderStub{user: user},
			},
			"repository": {
				verifier: &identityVerifierStub{userID: userID},
				finder:   &userFinderStub{err: errors.New("database leaked person@example.com")},
			},
		} {
			t.Run(name, func(t *testing.T) {
				service, err := NewService(test.finder, test.verifier)
				if err != nil {
					t.Fatalf("NewService() error = %v", err)
				}
				_, err = service.Authenticate(context.Background(), "secret-evidence")
				if !errors.Is(err, ErrAuthenticationUnavailable) {
					t.Fatalf("Authenticate() error = %v, want ErrAuthenticationUnavailable", err)
				}
				if strings.Contains(err.Error(), "secret-evidence") || strings.Contains(err.Error(), "person@example.com") {
					t.Fatalf("Authenticate() exposed dependency details: %v", err)
				}
			})
		}
	})

	t.Run("preserves cancellation", func(t *testing.T) {
		for name, test := range map[string]struct {
			verifier *identityVerifierStub
			finder   *userFinderStub
			want     error
		}{
			"verifier canceled": {
				verifier: &identityVerifierStub{err: fmt.Errorf("provider leaked secret-evidence: %w", context.Canceled)},
				finder:   &userFinderStub{user: user},
				want:     context.Canceled,
			},
			"verifier deadline": {
				verifier: &identityVerifierStub{err: fmt.Errorf("provider leaked secret-evidence: %w", context.DeadlineExceeded)},
				finder:   &userFinderStub{user: user},
				want:     context.DeadlineExceeded,
			},
			"repository canceled": {
				verifier: &identityVerifierStub{userID: userID},
				finder:   &userFinderStub{err: fmt.Errorf("database leaked person@example.com: %w", context.Canceled)},
				want:     context.Canceled,
			},
			"repository deadline": {
				verifier: &identityVerifierStub{userID: userID},
				finder:   &userFinderStub{err: fmt.Errorf("database leaked person@example.com: %w", context.DeadlineExceeded)},
				want:     context.DeadlineExceeded,
			},
		} {
			t.Run(name, func(t *testing.T) {
				service, err := NewService(test.finder, test.verifier)
				if err != nil {
					t.Fatalf("NewService() error = %v", err)
				}
				_, err = service.Authenticate(context.Background(), "opaque-evidence")
				if err != test.want {
					t.Fatalf("Authenticate() error = %v, want canonical %v", err, test.want)
				}
				if strings.Contains(err.Error(), "secret-evidence") || strings.Contains(err.Error(), "person@example.com") {
					t.Fatalf("Authenticate() exposed cancellation details: %v", err)
				}
			})
		}
	})

	t.Run("does not verify an already canceled request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		verifier := &identityVerifierStub{userID: userID}
		finder := &userFinderStub{user: user}
		service, err := NewService(finder, verifier)
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		if _, err := service.Authenticate(ctx, "opaque-evidence"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Authenticate() error = %v, want context.Canceled", err)
		}
		if verifier.calls != 0 || finder.calls != 0 {
			t.Fatal("Authenticate() called dependencies for a canceled request")
		}
	})

	t.Run("rejects nil context", func(t *testing.T) {
		service, err := NewService(&userFinderStub{user: user}, &identityVerifierStub{userID: userID})
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		if _, err := service.Authenticate(nil, "opaque-evidence"); !errors.Is(err, ErrAuthenticationUnavailable) {
			t.Fatalf("Authenticate() error = %v, want ErrAuthenticationUnavailable", err)
		}
	})
}

func TestActorUserIDFailsClosed(t *testing.T) {
	if _, ok := (Actor{}).UserID(); ok {
		t.Fatal("zero Actor returned a usable user ID")
	}
}
