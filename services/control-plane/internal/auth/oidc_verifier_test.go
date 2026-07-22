package auth

import (
	"context"
	"errors"
	"testing"
)

type oidcTokenVerifierStub struct {
	claims   verifiedOIDCClaims
	err      error
	calls    int
	evidence string
}

func (verifier *oidcTokenVerifierStub) VerifyOIDCToken(
	_ context.Context,
	evidence string,
) (verifiedOIDCClaims, error) {
	verifier.calls++
	verifier.evidence = evidence
	return verifier.claims, verifier.err
}

type oidcIdentityFinderStub struct {
	userID   UserID
	err      error
	calls    int
	identity OIDCIdentity
}

func (finder *oidcIdentityFinderStub) FindUserIDByOIDCIdentity(
	_ context.Context,
	identity OIDCIdentity,
) (UserID, error) {
	finder.calls++
	finder.identity = identity
	return finder.userID, finder.err
}

func TestNewOIDCIdentityVerifierRequiresDependencies(t *testing.T) {
	tokens := &oidcTokenVerifierStub{}
	identities := &oidcIdentityFinderStub{}
	var nilTokens *oidcTokenVerifierStub
	var nilIdentities *oidcIdentityFinderStub

	if verifier, err := NewOIDCIdentityVerifier(nil, identities); err == nil || verifier != nil {
		t.Fatalf("NewOIDCIdentityVerifier(nil, identities) = (%v, %v), want error", verifier, err)
	}
	if verifier, err := NewOIDCIdentityVerifier(tokens, nil); err == nil || verifier != nil {
		t.Fatalf("NewOIDCIdentityVerifier(tokens, nil) = (%v, %v), want error", verifier, err)
	}
	if verifier, err := NewOIDCIdentityVerifier(nilTokens, identities); err == nil || verifier != nil {
		t.Fatalf("NewOIDCIdentityVerifier(nilTokens, identities) = (%v, %v), want error", verifier, err)
	}
	if verifier, err := NewOIDCIdentityVerifier(tokens, nilIdentities); err == nil || verifier != nil {
		t.Fatalf("NewOIDCIdentityVerifier(tokens, nilIdentities) = (%v, %v), want error", verifier, err)
	}
}

func TestOIDCIdentityVerifierResolvesExactVerifiedIdentity(t *testing.T) {
	userID, err := ParseUserID("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	tokens := &oidcTokenVerifierStub{claims: verifiedOIDCClaims{
		issuer:  "https://identity.example/realms/CodeRoam",
		subject: "Case-Sensitive-Subject",
	}}
	identities := &oidcIdentityFinderStub{userID: userID}
	verifier, err := NewOIDCIdentityVerifier(tokens, identities)
	if err != nil {
		t.Fatalf("NewOIDCIdentityVerifier() error = %v", err)
	}

	got, err := verifier.Verify(context.Background(), "signed-id-token")
	if err != nil || got != userID {
		t.Fatalf("Verify() = (%v, %v), want (%v, nil)", got, err, userID)
	}
	if tokens.calls != 1 || tokens.evidence != "signed-id-token" || identities.calls != 1 ||
		identities.identity.issuer != tokens.claims.issuer ||
		identities.identity.subject != tokens.claims.subject {
		t.Fatal("Verify() did not resolve the exact verified issuer and subject")
	}
}

func TestOIDCIdentityVerifierFailsClosed(t *testing.T) {
	tests := map[string]struct {
		claims          verifiedOIDCClaims
		tokenErr        error
		identityErr     error
		want            error
		wantFinderCalls int
	}{
		"rejected token": {
			tokenErr: ErrIdentityRejected,
			want:     ErrIdentityRejected,
		},
		"unavailable token verifier": {
			tokenErr: errors.New("provider unavailable"),
			want:     ErrAuthenticationUnavailable,
		},
		"invalid verified issuer": {
			claims: verifiedOIDCClaims{issuer: "http://identity.example", subject: "subject"},
			want:   ErrIdentityRejected,
		},
		"invalid verified subject": {
			claims: verifiedOIDCClaims{issuer: "https://identity.example", subject: ""},
			want:   ErrIdentityRejected,
		},
		"unlinked identity": {
			claims:          verifiedOIDCClaims{issuer: "https://identity.example", subject: "subject"},
			identityErr:     ErrOIDCIdentityNotFound,
			want:            ErrIdentityRejected,
			wantFinderCalls: 1,
		},
		"identity store unavailable": {
			claims:          verifiedOIDCClaims{issuer: "https://identity.example", subject: "subject"},
			identityErr:     errors.New("database unavailable"),
			want:            ErrAuthenticationUnavailable,
			wantFinderCalls: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tokens := &oidcTokenVerifierStub{claims: test.claims, err: test.tokenErr}
			identities := &oidcIdentityFinderStub{err: test.identityErr}
			verifier, err := NewOIDCIdentityVerifier(tokens, identities)
			if err != nil {
				t.Fatalf("NewOIDCIdentityVerifier() error = %v", err)
			}
			_, err = verifier.Verify(context.Background(), "signed-id-token")
			if test.want == ErrAuthenticationUnavailable {
				if errors.Is(err, ErrIdentityRejected) || err == nil {
					t.Fatalf("Verify() error = %v, want unavailable dependency error", err)
				}
			} else if !errors.Is(err, test.want) {
				t.Fatalf("Verify() error = %v, want %v", err, test.want)
			}
			if identities.calls != test.wantFinderCalls {
				t.Fatalf("identity finder calls = %d, want %d", identities.calls, test.wantFinderCalls)
			}
		})
	}
}

func TestOIDCIdentityVerifierPreservesCancellation(t *testing.T) {
	t.Run("before token verification", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		tokens := &oidcTokenVerifierStub{}
		identities := &oidcIdentityFinderStub{}
		verifier, err := NewOIDCIdentityVerifier(tokens, identities)
		if err != nil {
			t.Fatalf("NewOIDCIdentityVerifier() error = %v", err)
		}
		if _, err := verifier.Verify(ctx, "signed-id-token"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Verify() error = %v, want context.Canceled", err)
		}
		if tokens.calls != 0 || identities.calls != 0 {
			t.Fatal("Verify() called dependencies for an already canceled context")
		}
	})

	t.Run("from token verifier", func(t *testing.T) {
		verifier, err := NewOIDCIdentityVerifier(
			&oidcTokenVerifierStub{err: context.DeadlineExceeded},
			&oidcIdentityFinderStub{},
		)
		if err != nil {
			t.Fatalf("NewOIDCIdentityVerifier() error = %v", err)
		}
		if _, err := verifier.Verify(context.Background(), "signed-id-token"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Verify() error = %v, want context.DeadlineExceeded", err)
		}
	})

	var nilVerifier *OIDCIdentityVerifier
	if _, err := nilVerifier.Verify(context.Background(), "signed-id-token"); !errors.Is(err, ErrAuthenticationUnavailable) {
		t.Fatalf("nil Verify() error = %v, want ErrAuthenticationUnavailable", err)
	}

	t.Run("typed nil dependencies", func(t *testing.T) {
		for name, verifier := range map[string]*OIDCIdentityVerifier{
			"token verifier": {
				tokens:     (*oidcTokenVerifierStub)(nil),
				identities: &oidcIdentityFinderStub{},
			},
			"identity finder": {
				tokens:     &oidcTokenVerifierStub{},
				identities: (*oidcIdentityFinderStub)(nil),
			},
		} {
			t.Run(name, func(t *testing.T) {
				if _, err := verifier.Verify(context.Background(), "signed-id-token"); !errors.Is(err, ErrAuthenticationUnavailable) {
					t.Fatalf("Verify() error = %v, want ErrAuthenticationUnavailable", err)
				}
			})
		}
	})
}
