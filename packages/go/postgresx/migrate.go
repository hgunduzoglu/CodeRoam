package postgresx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	migrationLockID          int64 = 0x434f4445524f414d
	migrationRollbackTimeout       = 5 * time.Second
)

var (
	ErrInvalidMigration  = errors.New("invalid PostgreSQL migration")
	ErrMigrationConflict = errors.New("PostgreSQL migration conflicts with applied ledger entry")
)

// Migration is one immutable, module-scoped PostgreSQL schema change. SQL must
// be trusted repository content and must not contain transaction control.
type Migration struct {
	Scope   string
	Version uint64
	Name    string
	SQL     string
}

// TransactionStarter is implemented by pgx connections and pools.
type TransactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

type preparedMigration struct {
	Migration
	checksum [sha256.Size]byte
}

// ApplyMigrations applies each migration once in scope/version order.
func ApplyMigrations(ctx context.Context, db TransactionStarter, migrations []Migration) error {
	prepared, err := prepareMigrations(migrations)
	if err != nil {
		return err
	}

	for _, migration := range prepared {
		if err := applyMigration(ctx, db, migration); err != nil {
			return err
		}
	}
	return nil
}

func prepareMigrations(migrations []Migration) ([]preparedMigration, error) {
	prepared := make([]preparedMigration, len(migrations))
	for index, migration := range migrations {
		switch {
		case migration.Scope == "", migration.Scope != strings.TrimSpace(migration.Scope), len(migration.Scope) > 64:
			return nil, fmt.Errorf("%w: migration %d has a non-canonical scope", ErrInvalidMigration, index)
		case migration.Version == 0, migration.Version > math.MaxInt64:
			return nil, fmt.Errorf("%w: %q has an unsupported version", ErrInvalidMigration, migration.Scope)
		case migration.Name == "", migration.Name != strings.TrimSpace(migration.Name), len(migration.Name) > 128:
			return nil, fmt.Errorf("%w: %q version %d has a non-canonical name", ErrInvalidMigration, migration.Scope, migration.Version)
		case strings.TrimSpace(migration.SQL) == "":
			return nil, fmt.Errorf("%w: %q version %d has empty SQL", ErrInvalidMigration, migration.Scope, migration.Version)
		}

		prepared[index] = preparedMigration{
			Migration: migration,
			checksum:  sha256.Sum256([]byte(migration.SQL)),
		}
	}

	sort.Slice(prepared, func(left, right int) bool {
		if prepared[left].Scope == prepared[right].Scope {
			return prepared[left].Version < prepared[right].Version
		}
		return prepared[left].Scope < prepared[right].Scope
	})
	for index := 1; index < len(prepared); index++ {
		previous := prepared[index-1]
		current := prepared[index]
		if previous.Scope == current.Scope && previous.Version == current.Version {
			return nil, fmt.Errorf("%w: duplicate %q version %d", ErrInvalidMigration, current.Scope, current.Version)
		}
	}
	return prepared, nil
}

func applyMigration(ctx context.Context, db TransactionStarter, migration preparedMigration) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %q version %d: %w", migration.Scope, migration.Version, err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), migrationRollbackTimeout)
		defer cancel()
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}
		rollbackErr = fmt.Errorf("rollback migration %q version %d: %w", migration.Scope, migration.Version, rollbackErr)
		if err == nil {
			err = rollbackErr
			return
		}
		err = errors.Join(err, rollbackErr)
	}()

	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("lock migration ledger: %w", err)
	}
	if _, err = tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS coderoam_meta`); err != nil {
		return fmt.Errorf("create migration ledger schema: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS coderoam_meta.schema_migrations (
			scope text NOT NULL,
			version bigint NOT NULL CHECK (version > 0),
			name text NOT NULL,
			checksum bytea NOT NULL CHECK (octet_length(checksum) = 32),
			applied_at timestamptz NOT NULL DEFAULT now(),
			PRIMARY KEY (scope, version)
		)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	var appliedName string
	var appliedChecksum []byte
	err = tx.QueryRow(ctx, `
		SELECT name, checksum
		FROM coderoam_meta.schema_migrations
		WHERE scope = $1 AND version = $2`, migration.Scope, int64(migration.Version)).Scan(&appliedName, &appliedChecksum)
	switch {
	case err == nil:
		if appliedName != migration.Name || !bytes.Equal(appliedChecksum, migration.checksum[:]) {
			return fmt.Errorf("%w: %q version %d", ErrMigrationConflict, migration.Scope, migration.Version)
		}
	case errors.Is(err, pgx.ErrNoRows):
		if err = tx.Conn().PgConn().Exec(ctx, migration.SQL).Close(); err != nil {
			return fmt.Errorf("apply migration %q version %d (%s): %w", migration.Scope, migration.Version, migration.Name, err)
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO coderoam_meta.schema_migrations (scope, version, name, checksum)
			VALUES ($1, $2, $3, $4)`, migration.Scope, int64(migration.Version), migration.Name, migration.checksum[:]); err != nil {
			return fmt.Errorf("record migration %q version %d: %w", migration.Scope, migration.Version, err)
		}
	default:
		return fmt.Errorf("read migration ledger for %q version %d: %w", migration.Scope, migration.Version, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %q version %d: %w", migration.Scope, migration.Version, err)
	}
	return nil
}
