package auth

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxAuthenticationEvidenceBytes = 8 * 1024

var (
	ErrUnauthenticated           = errors.New("unauthenticated")
	ErrAuthenticationUnavailable = errors.New("authentication unavailable")
	ErrIdentityRejected          = errors.New("identity rejected")
)

// IdentityVerifier is the provider adapter trust boundary. It must validate the supplied evidence
// and return ErrIdentityRejected for an invalid identity without including evidence in the error.
type IdentityVerifier interface {
	Verify(context.Context, string) (UserID, error)
}

type userFinder interface {
	FindByID(context.Context, UserID) (User, error)
}

type Service struct {
	users    userFinder
	verifier IdentityVerifier
}

// Actor is issued only after identity verification and local user resolution both succeed.
type Actor struct {
	userID UserID
}

func NewService(users userFinder, verifier IdentityVerifier) (*Service, error) {
	if users == nil {
		return nil, errors.New("auth service user finder is required")
	}
	if verifier == nil {
		return nil, errors.New("auth service identity verifier is required")
	}
	return &Service{users: users, verifier: verifier}, nil
}

func (service *Service) Authenticate(ctx context.Context, evidence string) (Actor, error) {
	if ctx == nil {
		return Actor{}, ErrAuthenticationUnavailable
	}
	if err := ctx.Err(); err != nil {
		return Actor{}, err
	}
	if len(evidence) == 0 || len(evidence) > maxAuthenticationEvidenceBytes ||
		!utf8.ValidString(evidence) || strings.ContainsFunc(evidence, unicode.IsControl) {
		return Actor{}, ErrUnauthenticated
	}
	if strings.TrimSpace(evidence) == "" {
		return Actor{}, ErrUnauthenticated
	}

	userID, err := service.verifier.Verify(ctx, evidence)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Actor{}, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return Actor{}, context.DeadlineExceeded
		}
		if errors.Is(err, ErrIdentityRejected) {
			return Actor{}, ErrUnauthenticated
		}
		return Actor{}, ErrAuthenticationUnavailable
	}
	if userID.String() == "" {
		return Actor{}, ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return Actor{}, err
	}

	user, err := service.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Actor{}, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return Actor{}, context.DeadlineExceeded
		}
		if errors.Is(err, ErrUserNotFound) {
			return Actor{}, ErrUnauthenticated
		}
		return Actor{}, ErrAuthenticationUnavailable
	}
	if user.id != userID {
		return Actor{}, ErrUnauthenticated
	}
	if err := ctx.Err(); err != nil {
		return Actor{}, err
	}
	return Actor{userID: userID}, nil
}

func (actor Actor) UserID() (UserID, bool) {
	if actor.userID.String() == "" {
		return UserID{}, false
	}
	return actor.userID, true
}
