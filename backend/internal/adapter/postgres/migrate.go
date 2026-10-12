package postgres

import (
	"cmp"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const migrationLockID = 727_001

// ErrLegacyMigrations means the database was migrated under the old NNN_
// numbering. Migrate never rewrites recorded history, so the fix is a reset.
var ErrLegacyMigrations = errors.New("schema_migrations holds pre-timestamp versions (NNN_*.sql); run `make db-reset`")

// migration is one embedded file. version is its 14-digit UTC timestamp
// prefix (YYYYMMDDHHMMSS), which is also its order and its schema_migrations key.
type migration struct {
	version int64
	file    string
	name    string
}

var migrationFile = regexp.MustCompile(`^(\d{14})_[a-z0-9_]+\.sql$`)

// parseMigrations is the one place the naming rule lives. A file that breaks
// it, a prefix that is not a real UTC time, or two files with one version
// fail here, before any SQL runs.
func parseMigrations(fsys fs.FS) ([]migration, error) {
	files, err := fs.Glob(fsys, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	out := make([]migration, 0, len(files))
	seen := map[int64]string{}
	for _, file := range files {
		name := path.Base(file)
		m := migrationFile.FindStringSubmatch(name)
		if m == nil {
			return nil, fmt.Errorf("migration %s: name must be YYYYMMDDHHMMSS_snake_case.sql (make migration name=...)", name)
		}
		if _, err := time.Parse("20060102150405", m[1]); err != nil {
			return nil, fmt.Errorf("migration %s: prefix is not a timestamp: %w", name, err)
		}
		version, _ := strconv.ParseInt(m[1], 10, 64)
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("migration %s: version %d already used by %s", name, version, prev)
		}
		seen[version] = name
		out = append(out, migration{version: version, file: file, name: name})
	}
	slices.SortFunc(out, func(a, b migration) int { return cmp.Compare(a.version, b.version) })
	return out, nil
}

func Migrate(ctx context.Context, db *DB) error {
	migrations, err := parseMigrations(migrationsFS)
	if err != nil {
		return err
	}
	if _, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT      PRIMARY KEY,
			name       TEXT        NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	var legacy bool
	if err := db.Pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version < 10000000000000)").Scan(&legacy); err != nil {
		return err
	}
	if legacy {
		return ErrLegacyMigrations
	}
	for _, m := range migrations {
		if err := apply(ctx, db, m); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	return nil
}

func apply(ctx context.Context, db *DB, m migration) error {
	return pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrationLockID); err != nil {
			return err
		}
		var done bool
		if err := tx.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", m.version).Scan(&done); err != nil {
			return err
		}
		if done {
			return nil
		}
		body, err := migrationsFS.ReadFile(m.file)
		if err != nil {
			return err
		}
		// No arguments → simple protocol, so one file may contain many statements.
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.version, m.name); err != nil {
			return err
		}
		slog.InfoContext(ctx, "applied migration", "name", m.name)
		return nil
	})
}
