package auth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

const (
	oidcVerificationTimeout   = 5 * time.Second
	maxOIDCFutureIssuedAt     = 2 * time.Minute
	maxOIDCTokenLifetime      = time.Hour
	minOIDCJWKSRefresh        = 30 * time.Second
	maxOIDCJWKSKeys           = 64
	maxOIDCJWKSResponseBytes  = 256 * 1024
	maxOIDCJWKSResponseHeader = 32 * 1024
)

var (
	errOIDCProviderUnavailable = errors.New("OIDC provider unavailable")
	errOIDCKeySourceMarker     = errors.New("\x00coderoam OIDC key source unavailable\x00")
	errOIDCSignatureRejected   = errors.New("OIDC token signature rejected")
)

type oidcIDTokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

type RemoteOIDCTokenVerifier struct {
	tokens       oidcIDTokenVerifier
	audience     string
	algorithm    string
	now          func() time.Time
	operationMax time.Duration
}

type oidcJWKSTransport struct {
	next    http.RoundTripper
	target  string
	alg     string
	now     func() time.Time
	keySets *OIDCJWKSCache
}

type oidcJWKSCacheKey struct {
	target    string
	algorithm string
}

type oidcJWKSCacheEntry struct {
	body        []byte
	lastAttempt time.Time
	inFlight    chan struct{}
}

type OIDCJWKSCache struct {
	mu       sync.Mutex
	byKeySet map[oidcJWKSCacheKey]*oidcJWKSCacheEntry
}

type cachedOIDCKeySet struct {
	client    *http.Client
	target    string
	algorithm string
}

func NewOIDCJWKSCache() *OIDCJWKSCache {
	return &OIDCJWKSCache{byKeySet: make(map[oidcJWKSCacheKey]*oidcJWKSCacheEntry)}
}

func NewRemoteOIDCTokenVerifier(
	config OIDCVerifierConfig,
	keySets *OIDCJWKSCache,
) (*RemoteOIDCTokenVerifier, error) {
	if !config.valid() {
		return nil, errors.New("invalid OIDC verifier configuration")
	}
	if keySets == nil || keySets.byKeySet == nil {
		return nil, errors.New("OIDC JWKS cache is required")
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("OIDC HTTP transport unavailable")
	}
	transport := base.Clone()
	transport.MaxResponseHeaderBytes = maxOIDCJWKSResponseHeader
	client := &http.Client{
		Transport: &oidcJWKSTransport{
			next: transport, target: config.JWKSURL, alg: config.SigningAlgorithm,
			now: time.Now, keySets: keySets,
		},
		Timeout: oidcVerificationTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	keySet := &cachedOIDCKeySet{
		client: client, target: config.JWKSURL, algorithm: config.SigningAlgorithm,
	}
	tokens := oidc.NewVerifier(config.Issuer, keySet, &oidc.Config{
		ClientID:             config.Audience,
		SupportedSigningAlgs: []string{config.SigningAlgorithm},
	})
	return &RemoteOIDCTokenVerifier{
		tokens:       tokens,
		audience:     config.Audience,
		algorithm:    config.SigningAlgorithm,
		now:          time.Now,
		operationMax: oidcVerificationTimeout,
	}, nil
}

func (keySet *cachedOIDCKeySet) VerifySignature(
	ctx context.Context,
	evidence string,
) ([]byte, error) {
	if keySet == nil || keySet.client == nil || ctx == nil || keySet.target == "" ||
		keySet.algorithm == "" {
		return nil, errOIDCKeySourceMarker
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, keySet.target, nil)
	if err != nil {
		return nil, errOIDCKeySourceMarker
	}
	response, err := keySet.client.Do(request)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		return nil, errOIDCKeySourceMarker
	}
	if response == nil || response.Body == nil || response.StatusCode != http.StatusOK ||
		response.ContentLength > maxOIDCJWKSResponseBytes {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, errOIDCKeySourceMarker
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxOIDCJWKSResponseBytes+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || len(body) > maxOIDCJWKSResponseBytes {
		return nil, errOIDCKeySourceMarker
	}
	var keys jose.JSONWebKeySet
	if err := json.Unmarshal(body, &keys); err != nil || len(keys.Keys) == 0 ||
		len(keys.Keys) > maxOIDCJWKSKeys {
		return nil, errOIDCKeySourceMarker
	}
	for _, key := range keys.Keys {
		if !key.Valid() || !oidcJWKSupportsAlgorithm(key, keySet.algorithm) {
			return nil, errOIDCKeySourceMarker
		}
	}
	signed, err := jose.ParseSigned(evidence, []jose.SignatureAlgorithm{
		jose.SignatureAlgorithm(keySet.algorithm),
	})
	if err != nil || len(signed.Signatures) != 1 {
		return nil, errOIDCSignatureRejected
	}
	keyID := signed.Signatures[0].Protected.KeyID
	for _, key := range keys.Keys {
		if keyID != "" && key.KeyID != keyID {
			continue
		}
		if payload, err := signed.Verify(&key); err == nil {
			return payload, nil
		}
	}
	return nil, errOIDCSignatureRejected
}

func (verifier *RemoteOIDCTokenVerifier) VerifyOIDCToken(
	ctx context.Context,
	evidence string,
) (verifiedOIDCClaims, error) {
	if verifier == nil || isNilOIDCVerifierDependency(verifier.tokens) ||
		verifier.now == nil || verifier.operationMax <= 0 ||
		!validOIDCAudience(verifier.audience) || verifier.algorithm == "" || ctx == nil {
		return verifiedOIDCClaims{}, errOIDCProviderUnavailable
	}
	if err := ctx.Err(); err != nil {
		return verifiedOIDCClaims{}, err
	}
	if len(evidence) == 0 || len(evidence) > maxAuthenticationEvidenceBytes ||
		!utf8.ValidString(evidence) || strings.ContainsFunc(evidence, unicode.IsControl) ||
		strings.Count(evidence, ".") != 2 {
		return verifiedOIDCClaims{}, ErrIdentityRejected
	}
	signed, err := jose.ParseSigned(evidence, []jose.SignatureAlgorithm{
		jose.SignatureAlgorithm(verifier.algorithm),
	})
	if err != nil || len(signed.Signatures) != 1 ||
		signed.Signatures[0].Protected.ExtraHeaders[jose.HeaderType] != "JWT" {
		return verifiedOIDCClaims{}, ErrIdentityRejected
	}

	operationCtx, cancel := context.WithTimeout(ctx, verifier.operationMax)
	defer cancel()
	token, err := verifier.tokens.Verify(operationCtx, evidence)
	if operationErr := operationCtx.Err(); operationErr != nil {
		return verifiedOIDCClaims{}, operationErr
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return verifiedOIDCClaims{}, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return verifiedOIDCClaims{}, context.DeadlineExceeded
		}
		// go-oidc currently formats KeySet failures with %v, so the private marker is
		// intentionally recognizable without exposing the JWKS response or token.
		if strings.Contains(err.Error(), errOIDCKeySourceMarker.Error()) {
			return verifiedOIDCClaims{}, errOIDCProviderUnavailable
		}
		return verifiedOIDCClaims{}, ErrIdentityRejected
	}
	if token == nil {
		return verifiedOIDCClaims{}, ErrIdentityRejected
	}

	now := verifier.now()
	if now.IsZero() {
		return verifiedOIDCClaims{}, errOIDCProviderUnavailable
	}
	identity, err := NewOIDCIdentity(token.Issuer, token.Subject)
	if err != nil || !slices.Contains(token.Audience, verifier.audience) ||
		token.Expiry.IsZero() || !token.Expiry.After(now) || token.IssuedAt.IsZero() ||
		token.IssuedAt.After(now.Add(maxOIDCFutureIssuedAt)) || !token.Expiry.After(token.IssuedAt) ||
		token.Expiry.Sub(token.IssuedAt) > maxOIDCTokenLifetime {
		return verifiedOIDCClaims{}, ErrIdentityRejected
	}
	var claims struct {
		AuthorizedParty string `json:"azp"`
	}
	if err := token.Claims(&claims); err != nil ||
		(len(token.Audience) > 1 && claims.AuthorizedParty == "") ||
		(claims.AuthorizedParty != "" && claims.AuthorizedParty != verifier.audience) {
		return verifiedOIDCClaims{}, ErrIdentityRejected
	}
	return verifiedOIDCClaims{issuer: identity.issuer, subject: identity.subject}, nil
}

func (transport *oidcJWKSTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || isNilOIDCVerifierDependency(transport.next) || request == nil ||
		transport.now == nil || transport.keySets == nil || transport.alg == "" ||
		request.URL == nil || request.Method != http.MethodGet ||
		request.URL.String() != transport.target {
		return nil, errOIDCKeySourceMarker
	}
	now := transport.now()
	if now.IsZero() {
		return nil, errOIDCKeySourceMarker
	}
	cacheKey := oidcJWKSCacheKey{target: transport.target, algorithm: transport.alg}
	cachedBody, fetch := transport.keySets.beforeFetch(request.Context(), cacheKey, now)
	if len(cachedBody) > 0 {
		return &http.Response{
			StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(cachedBody)),
			ContentLength: int64(len(cachedBody)), Header: make(http.Header), Request: request,
		}, nil
	}
	if !fetch {
		return nil, errOIDCKeySourceMarker
	}
	var validatedBody []byte
	defer func() { transport.keySets.finishFetch(cacheKey, validatedBody) }()
	response, err := transport.next.RoundTrip(request)
	if err != nil {
		return nil, errOIDCKeySourceMarker
	}
	if response == nil || response.Body == nil {
		return nil, errOIDCKeySourceMarker
	}
	if response.StatusCode != http.StatusOK || response.ContentLength > maxOIDCJWKSResponseBytes {
		_ = response.Body.Close()
		return nil, errOIDCKeySourceMarker
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxOIDCJWKSResponseBytes+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || len(body) > maxOIDCJWKSResponseBytes || !json.Valid(body) {
		return nil, errOIDCKeySourceMarker
	}
	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal(body, &keySet); err != nil || len(keySet.Keys) == 0 ||
		len(keySet.Keys) > maxOIDCJWKSKeys {
		return nil, errOIDCKeySourceMarker
	}
	filtered := jose.JSONWebKeySet{Keys: make([]jose.JSONWebKey, 0, len(keySet.Keys))}
	for _, key := range keySet.Keys {
		if !key.Valid() {
			return nil, errOIDCKeySourceMarker
		}
		if oidcJWKSupportsAlgorithm(key, transport.alg) {
			filtered.Keys = append(filtered.Keys, key)
		}
	}
	if len(filtered.Keys) == 0 {
		return nil, errOIDCKeySourceMarker
	}
	validatedBody, err = json.Marshal(filtered)
	if err != nil || len(validatedBody) > maxOIDCJWKSResponseBytes {
		return nil, errOIDCKeySourceMarker
	}
	response.Body = io.NopCloser(bytes.NewReader(validatedBody))
	response.ContentLength = int64(len(validatedBody))
	return response, nil
}

func oidcJWKSupportsAlgorithm(key jose.JSONWebKey, algorithm string) bool {
	if !key.IsPublic() || (key.Use != "" && key.Use != "sig") ||
		(key.Algorithm != "" && key.Algorithm != algorithm) {
		return false
	}
	switch algorithm {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		publicKey, ok := key.Key.(*rsa.PublicKey)
		return ok && publicKey.N != nil && publicKey.N.BitLen() >= 2048
	case "ES256":
		publicKey, ok := key.Key.(*ecdsa.PublicKey)
		return ok && publicKey.Curve == elliptic.P256()
	case "ES384":
		publicKey, ok := key.Key.(*ecdsa.PublicKey)
		return ok && publicKey.Curve == elliptic.P384()
	case "ES512":
		publicKey, ok := key.Key.(*ecdsa.PublicKey)
		return ok && publicKey.Curve == elliptic.P521()
	case "EdDSA":
		_, ok := key.Key.(ed25519.PublicKey)
		return ok
	default:
		return false
	}
}

func (cache *OIDCJWKSCache) beforeFetch(
	ctx context.Context,
	key oidcJWKSCacheKey,
	now time.Time,
) ([]byte, bool) {
	if cache == nil || cache.byKeySet == nil || ctx == nil || key.target == "" ||
		key.algorithm == "" || now.IsZero() {
		return nil, false
	}
	for {
		cache.mu.Lock()
		entry := cache.byKeySet[key]
		if entry == nil {
			entry = &oidcJWKSCacheEntry{}
			cache.byKeySet[key] = entry
		}
		if entry.inFlight != nil {
			finished := entry.inFlight
			cache.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, false
			case <-finished:
				continue
			}
		}
		if !entry.lastAttempt.IsZero() && now.Before(entry.lastAttempt.Add(minOIDCJWKSRefresh)) {
			body := bytes.Clone(entry.body)
			cache.mu.Unlock()
			return body, false
		}
		entry.lastAttempt = now
		entry.inFlight = make(chan struct{})
		cache.mu.Unlock()
		return nil, true
	}
}

func (cache *OIDCJWKSCache) finishFetch(key oidcJWKSCacheKey, body []byte) {
	if cache == nil || cache.byKeySet == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	entry := cache.byKeySet[key]
	if entry == nil || entry.inFlight == nil {
		return
	}
	entry.body = bytes.Clone(body)
	close(entry.inFlight)
	entry.inFlight = nil
}
