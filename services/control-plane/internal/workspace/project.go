package workspace

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hgunduzoglu/coderoam/packages/go/ids"
	"github.com/hgunduzoglu/coderoam/services/control-plane/internal/auth"
)

const (
	maxProjectNameRunes = 128
	maxProjectNameBytes = maxProjectNameRunes * utf8.UTFMax
	maxProjectRootBytes = 4096
)

var (
	ErrInvalidProject      = errors.New("invalid workspace project")
	ErrProjectAccessDenied = errors.New("workspace project access denied")
)

type Project struct {
	id            ids.ID
	ownerID       auth.UserID
	environmentID ids.ID
	name          string
	rootPath      string
	createdAt     time.Time
}

func NewProject(
	actor auth.Actor,
	encodedID string,
	environment Environment,
	name string,
	rootPath string,
	createdAt time.Time,
) (Project, error) {
	ownerID, ok := actor.UserID()
	if !ok || environment.id.String() == "" || !environment.OwnedBy(actor) {
		return Project{}, ErrProjectAccessDenied
	}
	projectID, err := ids.Parse(encodedID)
	if err != nil {
		return Project{}, fmt.Errorf("%w: id", ErrInvalidProject)
	}
	if len(name) > maxProjectNameBytes || !utf8.ValidString(name) {
		return Project{}, fmt.Errorf("%w: name", ErrInvalidProject)
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsFunc(name, unicode.IsControl) ||
		utf8.RuneCountInString(name) > maxProjectNameRunes {
		return Project{}, fmt.Errorf("%w: name", ErrInvalidProject)
	}
	if len(rootPath) > maxProjectRootBytes || !utf8.ValidString(rootPath) ||
		strings.ContainsFunc(rootPath, unicode.IsControl) || !path.IsAbs(rootPath) ||
		rootPath == "/" || path.Clean(rootPath) != rootPath {
		return Project{}, fmt.Errorf("%w: root path", ErrInvalidProject)
	}
	if createdAt.IsZero() || createdAt.Before(environment.createdAt) {
		return Project{}, fmt.Errorf("%w: creation time", ErrInvalidProject)
	}

	return Project{
		id:            projectID,
		ownerID:       ownerID,
		environmentID: environment.id,
		name:          name,
		rootPath:      rootPath,
		createdAt:     createdAt.UTC(),
	}, nil
}

func (project Project) OwnedBy(actor auth.Actor) bool {
	actorID, ok := actor.UserID()
	return ok && project.ownerID.String() != "" && project.ownerID == actorID
}
