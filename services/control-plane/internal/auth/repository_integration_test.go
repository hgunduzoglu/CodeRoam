package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
)

func TestRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := postgresx.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	applyAuthIntegrationMigrations(t, ctx, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin auth repository transaction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := tx.Rollback(cleanupCtx); err != nil {
			t.Errorf("rollback auth repository transaction: %v", err)
		}
	})

	repository, err := NewRepository(tx)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	createdAt := time.Date(2026, time.July, 17, 16, 0, 0, 0, time.UTC)
	user, err := NewUser("0123456789abcdef0123456789abcdef", "PERSON@example.com", "Ada Lovelace", createdAt)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if err := repository.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	identity, err := NewOIDCIdentity("https://identity.example/realms/coderoam", "Case-Sensitive-Subject")
	if err != nil {
		t.Fatalf("NewOIDCIdentity() error = %v", err)
	}
	if err := repository.LinkOIDCIdentity(ctx, identity, user.id, createdAt.Add(time.Minute)); err != nil {
		t.Fatalf("LinkOIDCIdentity() error = %v", err)
	}
	resolvedID, err := repository.FindUserIDByOIDCIdentity(ctx, identity)
	if err != nil {
		t.Fatalf("FindUserIDByOIDCIdentity() error = %v", err)
	}
	if resolvedID != user.id {
		t.Fatalf("FindUserIDByOIDCIdentity() = %q, want %q", resolvedID.String(), user.id.String())
	}
	differentCase, err := NewOIDCIdentity(identity.issuer, "case-sensitive-subject")
	if err != nil {
		t.Fatalf("NewOIDCIdentity() different case error = %v", err)
	}
	if _, err := repository.FindUserIDByOIDCIdentity(ctx, differentCase); !errors.Is(err, ErrOIDCIdentityNotFound) {
		t.Fatalf("FindUserIDByOIDCIdentity() different case error = %v, want ErrOIDCIdentityNotFound", err)
	}

	stored, err := repository.FindByID(ctx, user.id)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if stored.id != user.id || stored.email != user.email || stored.displayName != user.displayName || !stored.createdAt.Equal(user.createdAt) {
		t.Fatal("FindByID() did not round-trip the stored user")
	}
	service, err := NewService(repository, &identityVerifierStub{userID: user.id})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	actor, err := service.Authenticate(ctx, "integration-evidence")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	actorUserID, ok := actor.UserID()
	if !ok || actorUserID != user.id {
		t.Fatal("Authenticate() did not resolve the persisted user")
	}

	duplicateID, err := NewUser(user.id.String(), "other@example.com", "Other", createdAt)
	if err != nil {
		t.Fatalf("NewUser() duplicate ID error = %v", err)
	}
	duplicateIDTx, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin duplicate ID savepoint: %v", err)
	}
	duplicateIDRepository, err := NewRepository(duplicateIDTx)
	if err != nil {
		t.Fatalf("NewRepository() duplicate ID error = %v", err)
	}
	duplicateIDErr := duplicateIDRepository.Create(ctx, duplicateID)
	if err := duplicateIDTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback duplicate ID savepoint: %v", err)
	}
	if !errors.Is(duplicateIDErr, ErrUserAlreadyExists) {
		t.Fatalf("Create() duplicate ID error = %v, want ErrUserAlreadyExists", duplicateIDErr)
	}
	duplicateEmail, err := NewUser("1123456789abcdef0123456789abcdef", user.email, "Other", createdAt)
	if err != nil {
		t.Fatalf("NewUser() duplicate email error = %v", err)
	}
	duplicateEmailTx, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin duplicate email savepoint: %v", err)
	}
	duplicateEmailRepository, err := NewRepository(duplicateEmailTx)
	if err != nil {
		t.Fatalf("NewRepository() duplicate email error = %v", err)
	}
	duplicateEmailErr := duplicateEmailRepository.Create(ctx, duplicateEmail)
	if err := duplicateEmailTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback duplicate email savepoint: %v", err)
	}
	if !errors.Is(duplicateEmailErr, ErrEmailInUse) {
		t.Fatalf("Create() duplicate email error = %v, want ErrEmailInUse", duplicateEmailErr)
	}

	otherUser, err := NewUser("4123456789abcdef0123456789abcdef", "other-identity@example.com", "Other Identity", createdAt)
	if err != nil {
		t.Fatalf("NewUser() other identity error = %v", err)
	}
	if err := repository.Create(ctx, otherUser); err != nil {
		t.Fatalf("Create() other identity user error = %v", err)
	}
	duplicateIdentityTx, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin duplicate identity savepoint: %v", err)
	}
	duplicateIdentityRepository, err := NewRepository(duplicateIdentityTx)
	if err != nil {
		t.Fatalf("NewRepository() duplicate identity error = %v", err)
	}
	duplicateIdentityErr := duplicateIdentityRepository.LinkOIDCIdentity(
		ctx, identity, otherUser.id, createdAt.Add(2*time.Minute),
	)
	if err := duplicateIdentityTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback duplicate identity savepoint: %v", err)
	}
	if !errors.Is(duplicateIdentityErr, ErrOIDCIdentityAlreadyLinked) {
		t.Fatalf("LinkOIDCIdentity() duplicate error = %v, want ErrOIDCIdentityAlreadyLinked", duplicateIdentityErr)
	}

	missingIdentity, err := NewOIDCIdentity(identity.issuer, "missing-user")
	if err != nil {
		t.Fatalf("NewOIDCIdentity() missing user error = %v", err)
	}
	missingUserID, err := ParseUserID("5123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() missing identity user error = %v", err)
	}
	missingIdentityTx, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin missing identity user savepoint: %v", err)
	}
	missingIdentityRepository, err := NewRepository(missingIdentityTx)
	if err != nil {
		t.Fatalf("NewRepository() missing identity user error = %v", err)
	}
	missingIdentityErr := missingIdentityRepository.LinkOIDCIdentity(
		ctx, missingIdentity, missingUserID, createdAt.Add(3*time.Minute),
	)
	if err := missingIdentityTx.Rollback(ctx); err != nil {
		t.Fatalf("rollback missing identity user savepoint: %v", err)
	}
	if !errors.Is(missingIdentityErr, ErrUserNotFound) {
		t.Fatalf("LinkOIDCIdentity() missing user error = %v, want ErrUserNotFound", missingIdentityErr)
	}

	missingID, err := ParseUserID("2123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("ParseUserID() missing ID error = %v", err)
	}
	if _, err := repository.FindByID(ctx, missingID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("FindByID() missing error = %v, want ErrUserNotFound", err)
	}

	corruptID := "3123456789abcdef0123456789abcdef"
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth.users (id, email, display_name, created_at)
		VALUES ($1, $2, $3, $4)`, corruptID, "person@EXAMPLE.COM", "Stored", createdAt); err != nil {
		t.Fatalf("insert corrupt auth user: %v", err)
	}
	parsedCorruptID, err := ParseUserID(corruptID)
	if err != nil {
		t.Fatalf("ParseUserID() corrupt ID error = %v", err)
	}
	if _, err := repository.FindByID(ctx, parsedCorruptID); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("FindByID() corrupt row error = %v, want ErrInvalidUser", err)
	}
}

func applyAuthIntegrationMigrations(
	t *testing.T,
	ctx context.Context,
	database postgresx.TransactionStarter,
) {
	t.Helper()
	files := []struct {
		version uint64
		name    string
		path    string
	}{
		{version: 1, name: "init", path: "migrations/000001_init.sql"},
		{version: 2, name: "oidc_identities", path: "migrations/000002_oidc_identities.sql"},
	}
	for _, file := range files {
		migrationSQL, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("read auth migration %q: %v", file.name, err)
		}
		if err := postgresx.ApplyMigrations(ctx, database, []postgresx.Migration{{
			Scope: "auth", Version: file.version, Name: file.name, SQL: string(migrationSQL),
		}}); err != nil {
			t.Fatalf("apply auth migration %q: %v", file.name, err)
		}
	}
}
