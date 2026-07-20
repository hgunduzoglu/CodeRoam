package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

type oidcIDTokenVerifierStub struct {
	token    *oidc.IDToken
	err      error
	calls    int
	evidence string
}

func (verifier *oidcIDTokenVerifierStub) Verify(
	_ context.Context,
	evidence string,
) (*oidc.IDToken, error) {
	verifier.calls++
	verifier.evidence = evidence
	return verifier.token, verifier.err
}

type oidcRoundTripperFunc func(*http.Request) (*http.Response, error)

func (roundTrip oidcRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestNewRemoteOIDCTokenVerifierRequiresValidConfiguration(t *testing.T) {
	valid := OIDCVerifierConfig{
		Issuer:           "https://identity.example/realms/coderoam",
		Audience:         "coderoam-mobile",
		JWKSURL:          "https://identity.example/realms/coderoam/keys",
		SigningAlgorithm: "RS256",
	}
	keySets := NewOIDCJWKSCache()
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatal("default HTTP transport is unavailable")
	}
	verifier, err := NewRemoteOIDCTokenVerifier(valid, keySets, baseTransport)
	if err != nil {
		t.Fatalf("NewRemoteOIDCTokenVerifier() error = %v", err)
	}
	if verifier == nil || verifier.tokens == nil || verifier.audience != valid.Audience ||
		verifier.now == nil || verifier.operationMax != oidcVerificationTimeout {
		t.Fatal("NewRemoteOIDCTokenVerifier() did not retain bounded verified configuration")
	}

	invalid := valid
	invalid.SigningAlgorithm = "none"
	if verifier, err := NewRemoteOIDCTokenVerifier(invalid, keySets, baseTransport); err == nil || verifier != nil {
		t.Fatalf("NewRemoteOIDCTokenVerifier(invalid) = (%v, %v), want error", verifier, err)
	}
	if verifier, err := NewRemoteOIDCTokenVerifier(valid, nil, baseTransport); err == nil || verifier != nil {
		t.Fatalf("NewRemoteOIDCTokenVerifier(valid, nil) = (%v, %v), want error", verifier, err)
	}
	if verifier, err := NewRemoteOIDCTokenVerifier(valid, &OIDCJWKSCache{}, baseTransport); err == nil || verifier != nil {
		t.Fatalf("NewRemoteOIDCTokenVerifier(valid, zero cache) = (%v, %v), want error", verifier, err)
	}
	if verifier, err := NewRemoteOIDCTokenVerifier(valid, keySets, nil); err == nil || verifier != nil {
		t.Fatalf("NewRemoteOIDCTokenVerifier(valid, keySets, nil) = (%v, %v), want error", verifier, err)
	}
}

func TestRemoteOIDCTokenVerifierAcceptsSignedExactClaims(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	verifier := newStaticRemoteOIDCTokenVerifier(t, privateKey, now)

	for name, claims := range map[string]map[string]any{
		"single audience": validOIDCTestClaims(now),
		"multiple audiences with authorized party": func() map[string]any {
			claims := validOIDCTestClaims(now)
			claims["aud"] = []string{"other-service", "coderoam-mobile"}
			claims["azp"] = "coderoam-mobile"
			return claims
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			evidence := signOIDCTestToken(t, privateKey, jose.RS256, claims)
			identity, err := verifier.VerifyOIDCToken(context.Background(), evidence)
			if err != nil {
				t.Fatalf("VerifyOIDCToken() error = %v", err)
			}
			if identity.issuer != claims["iss"] || identity.subject != claims["sub"] {
				t.Fatalf("VerifyOIDCToken() = %v, want exact issuer and subject", identity)
			}
		})
	}
}

func TestRemoteOIDCTokenVerifierRejectsNonIDTokenTypes(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	verifier := newStaticRemoteOIDCTokenVerifier(t, privateKey, now)

	for name, tokenType := range map[string]string{
		"missing":      "",
		"access token": "at+jwt",
		"media type":   "application/jwt",
		"wrong case":   "jwt",
	} {
		t.Run(name, func(t *testing.T) {
			evidence := signOIDCTestTokenWithHeaders(
				t, privateKey, jose.RS256, validOIDCTestClaims(now), tokenType, "",
			)
			identity, err := verifier.VerifyOIDCToken(context.Background(), evidence)
			if !errors.Is(err, ErrIdentityRejected) || identity != (verifiedOIDCClaims{}) {
				t.Fatalf("VerifyOIDCToken() = (%v, %v), want zero ErrIdentityRejected", identity, err)
			}
		})
	}
}

func TestRemoteOIDCTokenVerifierJWKSIntegration(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	keySetJSON, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &privateKey.PublicKey, Algorithm: "RS256", Use: "sig",
	}}})
	if err != nil {
		t.Fatalf("json.Marshal(JWKS) error = %v", err)
	}
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(keySetJSON)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = oidcVerificationTimeout
	client.Transport = &oidcJWKSTransport{
		next: client.Transport, target: server.URL, alg: "RS256", now: func() time.Time { return now },
		keySets: NewOIDCJWKSCache(),
	}
	keys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", keys, &oidc.Config{
		ClientID:             "coderoam-mobile",
		SupportedSigningAlgs: []string{"RS256"},
		Now:                  func() time.Time { return now },
	})
	verifier := &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now:          func() time.Time { return now },
		operationMax: oidcVerificationTimeout,
	}
	evidence := signOIDCTestToken(t, privateKey, jose.RS256, validOIDCTestClaims(now))

	identity, err := verifier.VerifyOIDCToken(context.Background(), evidence)
	if err != nil {
		t.Fatalf("VerifyOIDCToken() error = %v", err)
	}
	if identity.issuer != "https://identity.example/realms/coderoam" ||
		identity.subject != "Case-Sensitive-Subject" || calls.Load() != 1 {
		t.Fatalf("VerifyOIDCToken() = %v, JWKS calls = %d", identity, calls.Load())
	}
}

func TestRemoteOIDCTokenVerifierFiltersDisallowedJWKSKeys(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	strongKey := newOIDCTestKey(t)
	weakKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(1024) error = %v", err)
	}
	encKey, wrongAlgorithmKey := newOIDCTestKey(t), newOIDCTestKey(t)
	body, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: &strongKey.PublicKey, KeyID: "strong", Algorithm: "RS256", Use: "sig"},
		{Key: &weakKey.PublicKey, KeyID: "weak", Algorithm: "RS256", Use: "sig"},
		{Key: &encKey.PublicKey, KeyID: "encryption", Algorithm: "RS256", Use: "enc"},
		{Key: &wrongAlgorithmKey.PublicKey, KeyID: "wrong-alg", Algorithm: "PS256", Use: "sig"},
	}})
	if err != nil {
		t.Fatalf("json.Marshal(JWKS) error = %v", err)
	}
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = response.Write(body)
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = oidcVerificationTimeout
	client.Transport = &oidcJWKSTransport{
		next: client.Transport, target: server.URL, alg: "RS256",
		now: func() time.Time { return now }, keySets: NewOIDCJWKSCache(),
	}
	remoteKeys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", remoteKeys, &oidc.Config{
		ClientID: "coderoam-mobile", SupportedSigningAlgs: []string{"RS256"},
		Now: func() time.Time { return now },
	})
	verifier := &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now: func() time.Time { return now }, operationMax: oidcVerificationTimeout,
	}
	strongEvidence := signOIDCTestTokenWithHeaders(
		t, strongKey, jose.RS256, validOIDCTestClaims(now), "JWT", "strong",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), strongEvidence); err != nil {
		t.Fatalf("strong-key VerifyOIDCToken() error = %v", err)
	}

	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	response, err := client.Transport.RoundTrip(request)
	if err != nil {
		t.Fatalf("cached RoundTrip() error = %v", err)
	}
	defer response.Body.Close()
	var filtered jose.JSONWebKeySet
	if err := json.NewDecoder(response.Body).Decode(&filtered); err != nil {
		t.Fatalf("decode filtered JWKS error = %v", err)
	}
	if len(filtered.Keys) != 1 || filtered.Keys[0].KeyID != "strong" {
		t.Fatalf("filtered JWKS keys = %v, want only strong", filtered.Keys)
	}

	weakEvidence := signOIDCTestTokenWithHeaders(
		t, weakKey, jose.RS256, validOIDCTestClaims(now), "JWT", "weak",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), weakEvidence); !errors.Is(err, ErrIdentityRejected) {
		t.Fatalf("weak-key VerifyOIDCToken() error = %v, want identity rejected", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("JWKS calls after filtered weak-key attempt = %d, want 1", calls.Load())
	}
}

func TestRemoteOIDCTokenVerifierClassifiesJWKSFailureAsUnavailable(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "provider-secret-details", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = oidcVerificationTimeout
	client.Transport = &oidcJWKSTransport{
		next: client.Transport, target: server.URL, alg: "RS256", now: func() time.Time { return now },
		keySets: NewOIDCJWKSCache(),
	}
	keys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", keys, &oidc.Config{
		ClientID: "coderoam-mobile", SupportedSigningAlgs: []string{"RS256"},
		Now: func() time.Time { return now },
	})
	verifier := &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now:          func() time.Time { return now },
		operationMax: oidcVerificationTimeout,
	}
	evidence := signOIDCTestToken(t, privateKey, jose.RS256, validOIDCTestClaims(now))

	_, err := verifier.VerifyOIDCToken(context.Background(), evidence)
	if !errors.Is(err, errOIDCProviderUnavailable) || strings.Contains(err.Error(), "provider-secret-details") {
		t.Fatalf("VerifyOIDCToken() error = %v, want sanitized provider unavailable", err)
	}
}

func TestRemoteOIDCTokenVerifierBoundsUnknownKeyRefreshes(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	clock := now
	privateKey := newOIDCTestKey(t)
	keySetJSON := oidcTestJWKS(t, privateKey, "trusted")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(keySetJSON))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = oidcVerificationTimeout
	client.Transport = &oidcJWKSTransport{
		next: client.Transport, target: server.URL, alg: "RS256", now: func() time.Time { return clock },
		keySets: NewOIDCJWKSCache(),
	}
	keys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", keys, &oidc.Config{
		ClientID: "coderoam-mobile", SupportedSigningAlgs: []string{"RS256"},
		Now: func() time.Time { return now },
	})
	verifier := &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now: func() time.Time { return now }, operationMax: oidcVerificationTimeout,
	}
	warmEvidence := signOIDCTestTokenWithHeaders(
		t, privateKey, jose.RS256, validOIDCTestClaims(now), "JWT", "trusted",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), warmEvidence); err != nil {
		t.Fatalf("warm VerifyOIDCToken() error = %v", err)
	}

	for index := 0; index < 10; index++ {
		evidence := signOIDCTestTokenWithHeaders(
			t, privateKey, jose.RS256, validOIDCTestClaims(now), "JWT", string(rune('a'+index)),
		)
		if _, err := verifier.VerifyOIDCToken(context.Background(), evidence); !errors.Is(err, ErrIdentityRejected) {
			t.Fatalf("unknown-key VerifyOIDCToken() error = %v, want identity rejected", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("JWKS calls within refresh cooldown = %d, want 1", calls.Load())
	}

	clock = clock.Add(minOIDCJWKSRefresh)
	evidence := signOIDCTestTokenWithHeaders(
		t, privateKey, jose.RS256, validOIDCTestClaims(now), "JWT", "after-cooldown",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), evidence); err == nil {
		t.Fatal("unknown-key VerifyOIDCToken() after cooldown succeeded")
	}
	if calls.Load() != 2 {
		t.Fatalf("JWKS calls after refresh cooldown = %d, want 2", calls.Load())
	}
}

func TestRemoteOIDCTokenVerifierRejectsRemovedKeyAfterFreshnessLimit(t *testing.T) {
	clock := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	oldKey, newKey := newOIDCTestKey(t), newOIDCTestKey(t)
	currentJWKS := oidcTestJWKS(t, oldKey, "old")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = response.Write([]byte(currentJWKS))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = oidcVerificationTimeout
	client.Transport = &oidcJWKSTransport{
		next: client.Transport, target: server.URL, alg: "RS256",
		now: func() time.Time { return clock }, keySets: NewOIDCJWKSCache(),
	}
	keys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", keys, &oidc.Config{
		ClientID: "coderoam-mobile", SupportedSigningAlgs: []string{"RS256"},
		Now: func() time.Time { return clock },
	})
	verifier := &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now: func() time.Time { return clock }, operationMax: oidcVerificationTimeout,
	}
	oldEvidence := signOIDCTestTokenWithHeaders(
		t, oldKey, jose.RS256, validOIDCTestClaims(clock), "JWT", "old",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), oldEvidence); err != nil {
		t.Fatalf("warm old-key VerifyOIDCToken() error = %v", err)
	}

	currentJWKS = oidcTestJWKS(t, newKey, "new")
	clock = clock.Add(minOIDCJWKSRefresh)
	oldEvidence = signOIDCTestTokenWithHeaders(
		t, oldKey, jose.RS256, validOIDCTestClaims(clock), "JWT", "old",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), oldEvidence); !errors.Is(err, ErrIdentityRejected) {
		t.Fatalf("removed old-key VerifyOIDCToken() error = %v, want identity rejected", err)
	}
	newEvidence := signOIDCTestTokenWithHeaders(
		t, newKey, jose.RS256, validOIDCTestClaims(clock), "JWT", "new",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), newEvidence); err != nil {
		t.Fatalf("new-key VerifyOIDCToken() error = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("JWKS calls across key rotation = %d, want 2", calls.Load())
	}
}

func TestRemoteOIDCTokenVerifierClearsStaleKeysAfterRefreshFailure(t *testing.T) {
	clock := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	validJWKS := oidcTestJWKS(t, privateKey, "old")
	failRefresh := false
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		if failRefresh {
			http.Error(response, "provider details", http.StatusServiceUnavailable)
			return
		}
		_, _ = response.Write([]byte(validJWKS))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = oidcVerificationTimeout
	client.Transport = &oidcJWKSTransport{
		next: client.Transport, target: server.URL, alg: "RS256",
		now: func() time.Time { return clock }, keySets: NewOIDCJWKSCache(),
	}
	keys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", keys, &oidc.Config{
		ClientID: "coderoam-mobile", SupportedSigningAlgs: []string{"RS256"},
		Now: func() time.Time { return clock },
	})
	verifier := &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now: func() time.Time { return clock }, operationMax: oidcVerificationTimeout,
	}
	evidence := signOIDCTestTokenWithHeaders(
		t, privateKey, jose.RS256, validOIDCTestClaims(clock), "JWT", "old",
	)
	if _, err := verifier.VerifyOIDCToken(context.Background(), evidence); err != nil {
		t.Fatalf("warm VerifyOIDCToken() error = %v", err)
	}

	failRefresh = true
	clock = clock.Add(minOIDCJWKSRefresh)
	evidence = signOIDCTestTokenWithHeaders(
		t, privateKey, jose.RS256, validOIDCTestClaims(clock), "JWT", "old",
	)
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := verifier.VerifyOIDCToken(context.Background(), evidence); !errors.Is(err, errOIDCProviderUnavailable) {
			t.Fatalf("stale-key attempt %d error = %v, want provider unavailable", attempt, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("JWKS calls across failed refresh and cooldown retry = %d, want 2", calls.Load())
	}
}

func TestRemoteOIDCTokenVerifierRejectsInvalidTokens(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	otherKey := newOIDCTestKey(t)
	verifier := newStaticRemoteOIDCTokenVerifier(t, privateKey, now)

	tests := map[string]struct {
		mutate    func(map[string]any)
		signing   *rsa.PrivateKey
		algorithm jose.SignatureAlgorithm
		raw       string
	}{
		"wrong issuer": {
			mutate: func(claims map[string]any) { claims["iss"] = "https://other.example" },
		},
		"wrong audience": {
			mutate: func(claims map[string]any) { claims["aud"] = "other-client" },
		},
		"expired": {
			mutate: func(claims map[string]any) { claims["exp"] = now.Add(-time.Minute).Unix() },
		},
		"future not-before": {
			mutate: func(claims map[string]any) { claims["nbf"] = now.Add(10 * time.Minute).Unix() },
		},
		"missing subject": {
			mutate: func(claims map[string]any) { delete(claims, "sub") },
		},
		"missing issued-at": {
			mutate: func(claims map[string]any) { delete(claims, "iat") },
		},
		"future issued-at": {
			mutate: func(claims map[string]any) { claims["iat"] = now.Add(3 * time.Minute).Unix() },
		},
		"expiry before issued-at": {
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(time.Minute).Unix()
				claims["exp"] = now.Add(30 * time.Second).Unix()
			},
		},
		"excessive token lifetime": {
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(-time.Minute).Unix()
				claims["exp"] = now.Add(maxOIDCTokenLifetime).Unix()
			},
		},
		"multiple audiences without authorized party": {
			mutate: func(claims map[string]any) {
				claims["aud"] = []string{"coderoam-mobile", "other-service"}
			},
		},
		"wrong authorized party": {
			mutate: func(claims map[string]any) { claims["azp"] = "other-client" },
		},
		"wrong signature": {
			signing: otherKey,
		},
		"wrong signing algorithm": {
			algorithm: jose.PS256,
		},
		"malformed": {
			raw: "not-a-token",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			claims := validOIDCTestClaims(now)
			if test.mutate != nil {
				test.mutate(claims)
			}
			evidence := test.raw
			if evidence == "" {
				signing := test.signing
				if signing == nil {
					signing = privateKey
				}
				algorithm := test.algorithm
				if algorithm == "" {
					algorithm = jose.RS256
				}
				evidence = signOIDCTestToken(t, signing, algorithm, claims)
			}
			identity, err := verifier.VerifyOIDCToken(context.Background(), evidence)
			if !errors.Is(err, ErrIdentityRejected) || identity != (verifiedOIDCClaims{}) {
				t.Fatalf("VerifyOIDCToken() = (%v, %v), want zero ErrIdentityRejected", identity, err)
			}
		})
	}
}

func TestRemoteOIDCTokenVerifierFailsClosedAtOperationalBoundaries(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	validEvidence := signOIDCTestToken(t, newOIDCTestKey(t), jose.RS256, validOIDCTestClaims(now))

	t.Run("rejects malformed evidence before verification", func(t *testing.T) {
		for name, evidence := range map[string]string{
			"empty":         "",
			"control":       "token\nvalue",
			"invalid UTF-8": string([]byte{0xff}),
			"oversized":     strings.Repeat("a", maxAuthenticationEvidenceBytes+1),
		} {
			t.Run(name, func(t *testing.T) {
				tokens := &oidcIDTokenVerifierStub{}
				verifier := &RemoteOIDCTokenVerifier{
					tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256", now: time.Now,
					operationMax: oidcVerificationTimeout,
				}
				if _, err := verifier.VerifyOIDCToken(context.Background(), evidence); !errors.Is(err, ErrIdentityRejected) {
					t.Fatalf("VerifyOIDCToken() error = %v, want ErrIdentityRejected", err)
				}
				if tokens.calls != 0 {
					t.Fatal("VerifyOIDCToken() verified malformed evidence")
				}
			})
		}
	})

	t.Run("preserves cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		tokens := &oidcIDTokenVerifierStub{}
		verifier := &RemoteOIDCTokenVerifier{
			tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256", now: time.Now,
			operationMax: oidcVerificationTimeout,
		}
		if _, err := verifier.VerifyOIDCToken(ctx, "signed-token"); !errors.Is(err, context.Canceled) {
			t.Fatalf("VerifyOIDCToken() error = %v, want context.Canceled", err)
		}
		if tokens.calls != 0 {
			t.Fatal("VerifyOIDCToken() called dependency for canceled context")
		}
	})

	for name, test := range map[string]struct {
		tokenErr error
		want     error
	}{
		"provider unavailable": {tokenErr: errOIDCKeySourceMarker, want: errOIDCProviderUnavailable},
		"invalid token":        {tokenErr: errors.New("invalid token"), want: ErrIdentityRejected},
		"canceled verifier":    {tokenErr: context.Canceled, want: context.Canceled},
		"deadline verifier":    {tokenErr: context.DeadlineExceeded, want: context.DeadlineExceeded},
	} {
		t.Run(name, func(t *testing.T) {
			verifier := &RemoteOIDCTokenVerifier{
				tokens:   &oidcIDTokenVerifierStub{err: test.tokenErr},
				audience: "coderoam-mobile", algorithm: "RS256", now: time.Now,
				operationMax: oidcVerificationTimeout,
			}
			if _, err := verifier.VerifyOIDCToken(context.Background(), validEvidence); !errors.Is(err, test.want) {
				t.Fatalf("VerifyOIDCToken() error = %v, want %v", err, test.want)
			}
		})
	}

	var nilVerifier *RemoteOIDCTokenVerifier
	if _, err := nilVerifier.VerifyOIDCToken(context.Background(), "signed-token"); !errors.Is(err, errOIDCProviderUnavailable) {
		t.Fatalf("nil VerifyOIDCToken() error = %v, want unavailable", err)
	}
	var nilTokens *oidcIDTokenVerifierStub
	if _, err := (&RemoteOIDCTokenVerifier{
		tokens: nilTokens, audience: "coderoam-mobile", algorithm: "RS256", now: time.Now,
		operationMax: oidcVerificationTimeout,
	}).VerifyOIDCToken(context.Background(), "signed-token"); !errors.Is(err, errOIDCProviderUnavailable) {
		t.Fatalf("typed-nil VerifyOIDCToken() error = %v, want unavailable", err)
	}
}

func TestOIDCJWKSTransportAllowsOnlyBoundedExactResponse(t *testing.T) {
	target := "https://identity.example/keys"
	expectedBody := oidcTestJWKS(t, newOIDCTestKey(t), "trusted")
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	transport := &oidcJWKSTransport{
		target: target, alg: "RS256", now: time.Now,
		keySets: NewOIDCJWKSCache(),
		next: oidcRoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Body:          io.NopCloser(strings.NewReader(expectedBody)),
				ContentLength: -1,
				Header:        make(http.Header),
			}, nil
		}),
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(responseBody) != expectedBody || response.ContentLength != int64(len(responseBody)) {
		t.Fatalf("RoundTrip() body = %q length = %d", responseBody, response.ContentLength)
	}
}

func TestOIDCJWKSCacheServesTwoRemoteKeySetsWithOneFetch(t *testing.T) {
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	privateKey := newOIDCTestKey(t)
	body := oidcTestJWKS(t, privateKey, "trusted")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(body))
	}))
	defer server.Close()

	keySets := NewOIDCJWKSCache()
	newVerifier := func() *RemoteOIDCTokenVerifier {
		clientValue := *server.Client()
		client := &clientValue
		client.Timeout = oidcVerificationTimeout
		client.Transport = &oidcJWKSTransport{
			next: client.Transport, target: server.URL, alg: "RS256",
			now: func() time.Time { return now }, keySets: keySets,
		}
		remoteKeys := &cachedOIDCKeySet{client: client, target: server.URL, algorithm: "RS256"}
		tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", remoteKeys, &oidc.Config{
			ClientID: "coderoam-mobile", SupportedSigningAlgs: []string{"RS256"},
			Now: func() time.Time { return now },
		})
		return &RemoteOIDCTokenVerifier{
			tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
			now: func() time.Time { return now }, operationMax: oidcVerificationTimeout,
		}
	}
	evidence := signOIDCTestTokenWithHeaders(
		t, privateKey, jose.RS256, validOIDCTestClaims(now), "JWT", "trusted",
	)
	for index, verifier := range []*RemoteOIDCTokenVerifier{newVerifier(), newVerifier()} {
		if _, err := verifier.VerifyOIDCToken(context.Background(), evidence); err != nil {
			t.Fatalf("verifier %d VerifyOIDCToken() error = %v after %d JWKS calls", index, err, calls.Load())
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("JWKS calls for two cold RemoteKeySets = %d, want 1", calls.Load())
	}
}

func TestOIDCJWKSupportsOnlyStrongConfiguredKeys(t *testing.T) {
	weakRSA, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(1024) error = %v", err)
	}
	newECDSA := func(curve elliptic.Curve) *ecdsa.PublicKey {
		privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			t.Fatalf("ecdsa.GenerateKey() error = %v", err)
		}
		return &privateKey.PublicKey
	}
	p256, p384, p521 := newECDSA(elliptic.P256()), newECDSA(elliptic.P384()), newECDSA(elliptic.P521())
	tests := map[string]struct {
		key       any
		algorithm string
		want      bool
	}{
		"strong RSA":     {key: &newOIDCTestKey(t).PublicKey, algorithm: "RS256", want: true},
		"weak RSA":       {key: &weakRSA.PublicKey, algorithm: "RS256"},
		"P-256 ES256":    {key: p256, algorithm: "ES256", want: true},
		"P-384 ES384":    {key: p384, algorithm: "ES384", want: true},
		"P-521 ES512":    {key: p521, algorithm: "ES512", want: true},
		"P-256 ES384":    {key: p256, algorithm: "ES384"},
		"P-256 ES512":    {key: p256, algorithm: "ES512"},
		"P-384 ES256":    {key: p384, algorithm: "ES256"},
		"P-384 ES512":    {key: p384, algorithm: "ES512"},
		"P-521 ES256":    {key: p521, algorithm: "ES256"},
		"P-521 ES384":    {key: p521, algorithm: "ES384"},
		"encryption use": {key: &newOIDCTestKey(t).PublicKey, algorithm: "RS256"},
		"wrong key alg":  {key: &newOIDCTestKey(t).PublicKey, algorithm: "RS256"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			key := jose.JSONWebKey{Key: test.key, Algorithm: test.algorithm, Use: "sig"}
			if name == "encryption use" {
				key.Use = "enc"
			}
			if name == "wrong key alg" {
				key.Algorithm = "PS256"
			}
			if got := oidcJWKSupportsAlgorithm(key, test.algorithm); got != test.want {
				t.Fatalf("oidcJWKSupportsAlgorithm() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOIDCJWKSTransportRejectsUnsafeRequestsAndResponses(t *testing.T) {
	target := "https://identity.example/keys"
	weakRSA, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(1024) error = %v", err)
	}
	marshalKeySet := func(key any, algorithm string) string {
		body, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: key, Algorithm: algorithm, Use: "sig",
		}}})
		if err != nil {
			t.Fatalf("json.Marshal(JWKS) error = %v", err)
		}
		return string(body)
	}
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey(P-384) error = %v", err)
	}
	manyKeys := jose.JSONWebKeySet{Keys: make([]jose.JSONWebKey, maxOIDCJWKSKeys+1)}
	manyKeysPublic := &newOIDCTestKey(t).PublicKey
	for index := range manyKeys.Keys {
		manyKeys.Keys[index] = jose.JSONWebKey{Key: manyKeysPublic, Algorithm: "RS256", Use: "sig"}
	}
	manyKeysBody, err := json.Marshal(manyKeys)
	if err != nil {
		t.Fatalf("json.Marshal(many JWKS keys) error = %v", err)
	}
	tests := map[string]struct {
		method        string
		url           string
		algorithm     string
		response      *http.Response
		nextErr       error
		wantNextCalls int
	}{
		"wrong method": {method: http.MethodPost, url: target},
		"wrong URL":    {method: http.MethodGet, url: "https://other.example/keys"},
		"redirect": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusFound, `{}`),
		},
		"provider error": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusServiceUnavailable, `provider details`),
		},
		"network error": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			nextErr: errors.New("network details"),
		},
		"invalid JSON": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, `not-json`),
		},
		"non-JWKS JSON": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, `{"error":"maintenance"}`),
		},
		"empty key set": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, `{"keys":[]}`),
		},
		"malformed key": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, `{"keys":[{"kty":"RSA"}]}`),
		},
		"weak RSA key": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, marshalKeySet(&weakRSA.PublicKey, "RS256")),
		},
		"mismatched ECDSA curve": {
			method: http.MethodGet, url: target, algorithm: "ES256", wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, marshalKeySet(&p384.PublicKey, "ES256")),
		},
		"too many keys": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(http.StatusOK, string(manyKeysBody)),
		},
		"oversized body": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: oidcHTTPTestResponse(
				http.StatusOK,
				strings.Repeat(" ", maxOIDCJWKSResponseBytes)+`{}`,
			),
		},
		"declared oversized body": {
			method: http.MethodGet, url: target, wantNextCalls: 1,
			response: &http.Response{
				StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)),
				ContentLength: maxOIDCJWKSResponseBytes + 1, Header: make(http.Header),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			calls := 0
			algorithm := test.algorithm
			if algorithm == "" {
				algorithm = "RS256"
			}
			transport := &oidcJWKSTransport{
				target: target, alg: algorithm, now: time.Now,
				keySets: NewOIDCJWKSCache(),
				next: oidcRoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
					calls++
					return test.response, test.nextErr
				}),
			}
			request, err := http.NewRequest(test.method, test.url, nil)
			if err != nil {
				t.Fatalf("http.NewRequest() error = %v", err)
			}
			response, err := transport.RoundTrip(request)
			if !errors.Is(err, errOIDCKeySourceMarker) || response != nil {
				t.Fatalf("RoundTrip() = (%v, %v), want nil key-source marker", response, err)
			}
			if calls != test.wantNextCalls {
				t.Fatalf("next RoundTrip() calls = %d, want %d", calls, test.wantNextCalls)
			}
		})
	}

	var nilTransport *oidcJWKSTransport
	if response, err := nilTransport.RoundTrip(nil); !errors.Is(err, errOIDCKeySourceMarker) || response != nil {
		t.Fatalf("nil RoundTrip() = (%v, %v), want nil key-source marker", response, err)
	}
}

func newStaticRemoteOIDCTokenVerifier(
	t *testing.T,
	privateKey *rsa.PrivateKey,
	now time.Time,
) *RemoteOIDCTokenVerifier {
	t.Helper()
	staticKeys := &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&privateKey.PublicKey}}
	tokens := oidc.NewVerifier("https://identity.example/realms/coderoam", staticKeys, &oidc.Config{
		ClientID:             "coderoam-mobile",
		SupportedSigningAlgs: []string{"RS256"},
		Now:                  func() time.Time { return now },
	})
	return &RemoteOIDCTokenVerifier{
		tokens: tokens, audience: "coderoam-mobile", algorithm: "RS256",
		now:          func() time.Time { return now },
		operationMax: oidcVerificationTimeout,
	}
}

func validOIDCTestClaims(now time.Time) map[string]any {
	return map[string]any{
		"iss": "https://identity.example/realms/coderoam",
		"sub": "Case-Sensitive-Subject",
		"aud": "coderoam-mobile",
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
}

func signOIDCTestToken(
	t *testing.T,
	privateKey *rsa.PrivateKey,
	algorithm jose.SignatureAlgorithm,
	claims map[string]any,
) string {
	t.Helper()
	return signOIDCTestTokenWithHeaders(t, privateKey, algorithm, claims, "JWT", "")
}

func signOIDCTestTokenWithHeaders(
	t *testing.T,
	privateKey *rsa.PrivateKey,
	algorithm jose.SignatureAlgorithm,
	claims map[string]any,
	tokenType string,
	keyID string,
) string {
	t.Helper()
	options := &jose.SignerOptions{}
	if tokenType != "" {
		options.WithType(jose.ContentType(tokenType))
	}
	if keyID != "" {
		options.WithHeader(jose.HeaderKey("kid"), keyID)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: algorithm, Key: privateKey},
		options,
	)
	if err != nil {
		t.Fatalf("jose.NewSigner() error = %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Signer.Sign() error = %v", err)
	}
	compact, err := signed.CompactSerialize()
	if err != nil {
		t.Fatalf("CompactSerialize() error = %v", err)
	}
	return compact
}

func newOIDCTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	return privateKey
}

func oidcHTTPTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    &http.Request{URL: &url.URL{}},
	}
}

func oidcTestJWKS(t *testing.T, privateKey *rsa.PrivateKey, keyID string) string {
	t.Helper()
	body, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &privateKey.PublicKey, KeyID: keyID, Algorithm: "RS256", Use: "sig",
	}}})
	if err != nil {
		t.Fatalf("json.Marshal(JWKS) error = %v", err)
	}
	return string(body)
}
