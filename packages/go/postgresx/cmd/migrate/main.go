package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hgunduzoglu/coderoam/packages/go/postgresx"
	"github.com/jackc/pgx/v5"
)

const (
	runnerTimeout      = 2 * time.Minute
	runnerCloseTimeout = 5 * time.Second
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), runnerTimeout)
	defer cancel()
	if err := run(ctx, os.Getenv("POSTGRES_DSN"), os.DirFS("."), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "apply PostgreSQL migrations: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dsn string, root fs.FS, scopeDirectories []string) (err error) {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("POSTGRES_DSN is required")
	}
	migrations, err := loadMigrations(root, scopeDirectories)
	if err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), runnerCloseTimeout)
		defer cancel()
		if closeErr := conn.Close(closeCtx); closeErr != nil {
			closeErr = fmt.Errorf("close PostgreSQL migration connection: %w", closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()

	return postgresx.ApplyMigrations(ctx, conn, migrations)
}

func loadMigrations(root fs.FS, scopeDirectories []string) ([]postgresx.Migration, error) {
	if len(scopeDirectories) == 0 {
		return nil, fmt.Errorf("%w: no scope directories", postgresx.ErrInvalidMigration)
	}

	seenScopes := make(map[string]struct{}, len(scopeDirectories))
	var migrations []postgresx.Migration
	for _, scopeDirectory := range scopeDirectories {
		scope, directory, found := strings.Cut(scopeDirectory, "=")
		if !found || scope == "" || directory == "" || !fs.ValidPath(directory) || directory != strings.TrimSpace(directory) {
			return nil, fmt.Errorf("%w: invalid scope directory %q", postgresx.ErrInvalidMigration, scopeDirectory)
		}
		if _, exists := seenScopes[scope]; exists {
			return nil, fmt.Errorf("%w: duplicate scope %q", postgresx.ErrInvalidMigration, scope)
		}
		seenScopes[scope] = struct{}{}

		scopeMigrations, err := loadScope(root, scope, directory)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, scopeMigrations...)
	}
	return migrations, nil
}

func loadScope(root fs.FS, scope, directory string) ([]postgresx.Migration, error) {
	entries, err := fs.ReadDir(root, directory)
	if err != nil {
		return nil, fmt.Errorf("read migration scope %q: %w", scope, err)
	}

	var migrations []postgresx.Migration
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".sql" {
			continue
		}
		if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil, fmt.Errorf("%w: %q contains non-regular SQL file %q", postgresx.ErrInvalidMigration, scope, entry.Name())
		}

		baseName := strings.TrimSuffix(entry.Name(), ".sql")
		versionText, name, found := strings.Cut(baseName, "_")
		version, parseErr := strconv.ParseUint(versionText, 10, 64)
		if !found || len(versionText) != 6 || version == 0 || parseErr != nil || name == "" || name != strings.TrimSpace(name) {
			return nil, fmt.Errorf("%w: %q has invalid migration filename %q", postgresx.ErrInvalidMigration, scope, entry.Name())
		}

		migrationPath := path.Join(directory, entry.Name())
		contents, err := fs.ReadFile(root, migrationPath)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", migrationPath, err)
		}
		if strings.TrimSpace(string(contents)) == "" {
			return nil, fmt.Errorf("%w: %q is empty", postgresx.ErrInvalidMigration, migrationPath)
		}
		migrations = append(migrations, postgresx.Migration{
			Scope:   scope,
			Version: version,
			Name:    name,
			SQL:     string(contents),
		})
	}
	if len(migrations) == 0 {
		return nil, fmt.Errorf("%w: %q contains no SQL migrations", postgresx.ErrInvalidMigration, scope)
	}
	return migrations, nil
}
