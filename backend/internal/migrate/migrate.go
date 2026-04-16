package migrate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	db  *pgxpool.Pool
	dir string
}

type Migration struct {
	Version   string
	Filename  string
	AppliedAt *time.Time
}

func New(dbURL, migrationsDir string) (*Migrator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Migrator{
		db:  db,
		dir: migrationsDir,
	}, nil
}

func (m *Migrator) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

func (m *Migrator) Up() error {
	ctx := context.Background()

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	files, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	if len(files) == 0 {
		slog.Info("no migration files found")
		return nil
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, file := range files {
		version := strings.TrimSuffix(file, ".sql")
		if _, ok := applied[version]; ok {
			slog.Debug("migration already applied", "file", file)
			continue
		}

		if err := m.applyMigration(ctx, file); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}
	}

	return nil
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	_, err := m.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	return err
}

func (m *Migrator) getMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}

	var files []string
	re := regexp.MustCompile(`^\d+_.*\.sql$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if re.MatchString(name) {
			files = append(files, name)
		}
	}

	sort.Strings(files)
	return files, nil
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			continue
		}
		applied[version] = true
	}

	return applied, nil
}

func (m *Migrator) applyMigration(ctx context.Context, filename string) error {
	path := filepath.Join(m.dir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute migration
	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration
	version := strings.TrimSuffix(filename, ".sql")
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`,
		version,
	); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	slog.Info("applied migration", "version", version)
	return nil
}

func (m *Migrator) Status() ([]Migration, error) {
	ctx := context.Background()

	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, err
	}

	files, err := m.getMigrationFiles()
	if err != nil {
		return nil, err
	}

	rows, err := m.db.Query(ctx,
		`SELECT version, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appliedMap := make(map[string]*time.Time)
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.AppliedAt); err != nil {
			continue
		}
		appliedMap[m.Version] = m.AppliedAt
	}

	var migrations []Migration
	for _, file := range files {
		version := strings.TrimSuffix(file, ".sql")
		m := Migration{
			Version:  version,
			Filename: file,
		}
		if appliedAt, ok := appliedMap[version]; ok {
			m.AppliedAt = appliedAt
		}
		migrations = append(migrations, m)
	}

	return migrations, nil
}
