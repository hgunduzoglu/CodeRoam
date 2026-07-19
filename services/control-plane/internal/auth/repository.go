package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrUserNotFound              = errors.New("user not found")
	ErrUserAlreadyExists         = errors.New("user already exists")
	ErrEmailInUse                = errors.New("email already in use")
	ErrOIDCIdentityNotFound      = errors.New("OIDC identity not found")
	ErrOIDCIdentityAlreadyLinked = errors.New("OIDC identity already linked")
)

type database interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct {
	db database
}

func NewRepository(db database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("auth repository database is required")
	}
	return &Repository{db: db}, nil
}

func (repository *Repository) Create(ctx context.Context, user User) error {
	if user.id.String() == "" {
		return fmt.Errorf("%w: id", ErrInvalidUser)
	}
	_, err := repository.db.Exec(ctx, `
		INSERT INTO auth.users (id, email, display_name, created_at)
		VALUES ($1, $2, $3, $4)`, user.id.String(), user.email, user.displayName, user.createdAt)
	if err == nil {
		return nil
	}

	var databaseErr *pgconn.PgError
	if errors.As(err, &databaseErr) && databaseErr.Code == "23505" {
		switch databaseErr.ConstraintName {
		case "users_pkey":
			return ErrUserAlreadyExists
		case "users_email_key":
			return ErrEmailInUse
		}
	}
	return fmt.Errorf("create auth user: %w", err)
}

func (repository *Repository) FindByID(ctx context.Context, id UserID) (User, error) {
	if id.String() == "" {
		return User{}, fmt.Errorf("%w: id", ErrInvalidUser)
	}

	var storedID, email, displayName string
	var createdAt time.Time
	err := repository.db.QueryRow(ctx, `
		SELECT id, email, display_name, created_at
		FROM auth.users
		WHERE id = $1`, id.String()).Scan(&storedID, &email, &displayName, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find auth user: %w", err)
	}

	user, err := NewUser(storedID, email, displayName, createdAt)
	if err != nil || user.id != id || user.email != email || user.displayName != displayName || !user.createdAt.Equal(createdAt) {
		return User{}, fmt.Errorf("%w: stored representation", ErrInvalidUser)
	}
	return user, nil
}

func (repository *Repository) LinkOIDCIdentity(
	ctx context.Context,
	identity OIDCIdentity,
	userID UserID,
	linkedAt time.Time,
) error {
	if !identity.valid() || userID.String() == "" || linkedAt.IsZero() {
		return ErrInvalidOIDCIdentity
	}
	_, err := repository.db.Exec(ctx, `
		INSERT INTO auth.oidc_identities (issuer, subject, user_id, linked_at)
		VALUES ($1, $2, $3, $4)`, identity.issuer, identity.subject, userID.String(), linkedAt.UTC())
	if err == nil {
		return nil
	}
	var databaseErr *pgconn.PgError
	if errors.As(err, &databaseErr) {
		switch databaseErr.ConstraintName {
		case "oidc_identities_pkey":
			return ErrOIDCIdentityAlreadyLinked
		case "oidc_identities_user_id_fkey":
			return ErrUserNotFound
		}
	}
	return fmt.Errorf("link OIDC identity: %w", err)
}

func (repository *Repository) FindUserIDByOIDCIdentity(
	ctx context.Context,
	identity OIDCIdentity,
) (UserID, error) {
	if !identity.valid() {
		return UserID{}, ErrInvalidOIDCIdentity
	}
	var storedIssuer, storedSubject, storedUserID string
	err := repository.db.QueryRow(ctx, `
		SELECT issuer, subject, user_id
		FROM auth.oidc_identities
		WHERE issuer = $1 AND subject = $2`, identity.issuer, identity.subject).Scan(
		&storedIssuer, &storedSubject, &storedUserID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserID{}, ErrOIDCIdentityNotFound
	}
	if err != nil {
		return UserID{}, fmt.Errorf("find OIDC identity: %w", err)
	}
	storedIdentity, identityErr := NewOIDCIdentity(storedIssuer, storedSubject)
	userID, userErr := ParseUserID(storedUserID)
	if identityErr != nil || userErr != nil || storedIdentity != identity {
		return UserID{}, ErrInvalidOIDCIdentity
	}
	return userID, nil
}
