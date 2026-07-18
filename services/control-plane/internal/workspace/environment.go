package workspace

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

const (
	maxEnvironmentNameRunes     = 128
	maxEnvironmentNameBytes     = maxEnvironmentNameRunes * utf8.UTFMax
	maxEnvironmentProviderBytes = 128
)

var (
	ErrInvalidEnvironment      = errors.New("invalid workspace environment")
	ErrEnvironmentAccessDenied = errors.New("workspace environment access denied")
)

type Environment struct {
	id        ids.ID
	ownerID   auth.UserID
	agentID   ids.ID
	name      string
	provider  string
	createdAt time.Time
}

func NewEnvironment(
	actor auth.Actor,
	encodedID string,
	agent Agent,
	name string,
	provider string,
	createdAt time.Time,
) (Environment, error) {
	ownerID, ok := actor.UserID()
	if !ok || agent.id.String() == "" || !agent.CanAuthorize(actor) {
		return Environment{}, ErrEnvironmentAccessDenied
	}
	return newEnvironment(
		ownerID, encodedID, agent.id, name, provider, agent.createdAt, createdAt,
	)
}

func newEnvironment(
	ownerID auth.UserID,
	encodedID string,
	agentID ids.ID,
	name string,
	provider string,
	agentCreatedAt time.Time,
	createdAt time.Time,
) (Environment, error) {
	if ownerID.String() == "" || agentID.String() == "" {
		return Environment{}, ErrEnvironmentAccessDenied
	}
	environmentID, err := ids.Parse(encodedID)
	if err != nil {
		return Environment{}, fmt.Errorf("%w: id", ErrInvalidEnvironment)
	}
	if len(name) > maxEnvironmentNameBytes || !utf8.ValidString(name) {
		return Environment{}, fmt.Errorf("%w: name", ErrInvalidEnvironment)
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsFunc(name, unicode.IsControl) ||
		utf8.RuneCountInString(name) > maxEnvironmentNameRunes {
		return Environment{}, fmt.Errorf("%w: name", ErrInvalidEnvironment)
	}
	if len(provider) > maxEnvironmentProviderBytes || !utf8.ValidString(provider) {
		return Environment{}, fmt.Errorf("%w: provider", ErrInvalidEnvironment)
	}
	provider = strings.TrimSpace(provider)
	if provider == "" || strings.ContainsFunc(provider, unicode.IsControl) {
		return Environment{}, fmt.Errorf("%w: provider", ErrInvalidEnvironment)
	}
	if agentCreatedAt.IsZero() || createdAt.IsZero() || createdAt.Before(agentCreatedAt) {
		return Environment{}, fmt.Errorf("%w: creation time", ErrInvalidEnvironment)
	}

	return Environment{
		id:        environmentID,
		ownerID:   ownerID,
		agentID:   agentID,
		name:      name,
		provider:  provider,
		createdAt: createdAt.UTC(),
	}, nil
}

func (environment Environment) OwnedBy(actor auth.Actor) bool {
	actorID, ok := actor.UserID()
	return ok && environment.ownerID.String() != "" && environment.ownerID == actorID
}
