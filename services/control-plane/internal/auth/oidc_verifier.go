package auth

import (
	"context"
	"errors"
	"reflect"
)

type verifiedOIDCClaims struct {
	issuer  string
	subject string
}

type oidcTokenVerifier interface {
	VerifyOIDCToken(context.Context, string) (verifiedOIDCClaims, error)
}

type oidcIdentityFinder interface {
	FindUserIDByOIDCIdentity(context.Context, OIDCIdentity) (UserID, error)
}

type OIDCIdentityVerifier struct {
	tokens     oidcTokenVerifier
	identities oidcIdentityFinder
}

func NewOIDCIdentityVerifier(
	tokens oidcTokenVerifier,
	identities oidcIdentityFinder,
) (*OIDCIdentityVerifier, error) {
	if isNilOIDCVerifierDependency(tokens) {
		return nil, errors.New("OIDC token verifier is required")
	}
	if isNilOIDCVerifierDependency(identities) {
		return nil, errors.New("OIDC identity finder is required")
	}
	return &OIDCIdentityVerifier{tokens: tokens, identities: identities}, nil
}

func (verifier *OIDCIdentityVerifier) Verify(ctx context.Context, evidence string) (UserID, error) {
	if verifier == nil || isNilOIDCVerifierDependency(verifier.tokens) ||
		isNilOIDCVerifierDependency(verifier.identities) || ctx == nil {
		return UserID{}, ErrAuthenticationUnavailable
	}
	if err := ctx.Err(); err != nil {
		return UserID{}, err
	}
	claims, err := verifier.tokens.VerifyOIDCToken(ctx, evidence)
	if err != nil {
		return UserID{}, err
	}
	if err := ctx.Err(); err != nil {
		return UserID{}, err
	}
	identity, err := NewOIDCIdentity(claims.issuer, claims.subject)
	if err != nil {
		return UserID{}, ErrIdentityRejected
	}
	userID, err := verifier.identities.FindUserIDByOIDCIdentity(ctx, identity)
	if errors.Is(err, ErrOIDCIdentityNotFound) {
		return UserID{}, ErrIdentityRejected
	}
	return userID, err
}

func isNilOIDCVerifierDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
