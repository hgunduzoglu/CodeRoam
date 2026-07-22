package workspace

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/cryptox"
	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

const (
	maxAgentNameRunes    = 128
	maxAgentNameBytes    = maxAgentNameRunes * utf8.UTFMax
	maxAgentVersionBytes = 64
)

var (
	ErrInvalidAgent      = errors.New("invalid workspace agent")
	ErrAgentAccessDenied = errors.New("workspace agent access denied")
)

type Agent struct {
	id         ids.ID
	ownerID    auth.UserID
	name       string
	publicKey  cryptox.X25519PublicKey
	version    string
	createdAt  time.Time
	revocation *agentRevocationState
}

type agentRevocationState struct {
	mu        sync.RWMutex
	revokedAt *time.Time
}

func NewAgent(
	actor auth.Actor,
	encodedID string,
	name string,
	publicKey cryptox.X25519PublicKey,
	version string,
	createdAt time.Time,
) (Agent, error) {
	ownerID, ok := actor.UserID()
	if !ok {
		return Agent{}, ErrAgentAccessDenied
	}
	agentID, err := ids.Parse(encodedID)
	if err != nil {
		return Agent{}, fmt.Errorf("%w: id", ErrInvalidAgent)
	}
	if len(name) > maxAgentNameBytes || !utf8.ValidString(name) {
		return Agent{}, fmt.Errorf("%w: name", ErrInvalidAgent)
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsFunc(name, unicode.IsControl) ||
		utf8.RuneCountInString(name) > maxAgentNameRunes {
		return Agent{}, fmt.Errorf("%w: name", ErrInvalidAgent)
	}
	if len(version) > maxAgentVersionBytes || !utf8.ValidString(version) {
		return Agent{}, fmt.Errorf("%w: version", ErrInvalidAgent)
	}
	version = strings.TrimSpace(version)
	if version == "" || strings.ContainsFunc(version, unicode.IsControl) {
		return Agent{}, fmt.Errorf("%w: version", ErrInvalidAgent)
	}
	if _, err := publicKey.Bytes(); err != nil {
		return Agent{}, fmt.Errorf("%w: public key", ErrInvalidAgent)
	}
	if createdAt.IsZero() {
		return Agent{}, fmt.Errorf("%w: creation time", ErrInvalidAgent)
	}

	return Agent{
		id:         agentID,
		ownerID:    ownerID,
		name:       name,
		publicKey:  publicKey,
		version:    version,
		createdAt:  createdAt.UTC(),
		revocation: &agentRevocationState{},
	}, nil
}

func (agent Agent) CanAuthorize(actor auth.Actor) bool {
	actorID, ok := actor.UserID()
	if !ok || agent.ownerID.String() == "" || agent.ownerID != actorID || agent.revocation == nil {
		return false
	}
	agent.revocation.mu.RLock()
	defer agent.revocation.mu.RUnlock()
	return agent.revocation.revokedAt == nil
}

func (agent *Agent) Revoke(actor auth.Actor, revokedAt time.Time) error {
	if agent == nil || agent.revocation == nil {
		return ErrInvalidAgent
	}
	actorID, ok := actor.UserID()
	if !ok || agent.ownerID.String() == "" || agent.ownerID != actorID {
		return ErrAgentAccessDenied
	}
	agent.revocation.mu.Lock()
	defer agent.revocation.mu.Unlock()
	if agent.revocation.revokedAt != nil {
		return nil
	}
	if revokedAt.IsZero() || revokedAt.Before(agent.createdAt) {
		return fmt.Errorf("%w: revoked time", ErrInvalidAgent)
	}

	normalizedRevokedAt := revokedAt.UTC()
	agent.revocation.revokedAt = &normalizedRevokedAt
	return nil
}
