package db

import (
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies pending SQL migrations embedded in the binary.
func RunMigrations() error {
	if DB == nil {
		return fmt.Errorf("db not initialized")
	}
	// Ensure schema_migrations table exists
	if _, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Discover migrations
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	files := make([]string, 0)
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, f := range files {
		version := strings.TrimSuffix(f, ".up.sql")
		var exists bool
		if err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration version %s: %w", version, err)
		}
		if exists {
			continue
		}
		path := filepath.ToSlash(filepath.Join("migrations", f))
		b, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		statements := splitSQLStatements(string(b))
		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := DB.Exec(stmt); err != nil {
				return fmt.Errorf("apply migration %s failed: %w\nstmt: %s", version, err, stmt)
			}
		}
		if _, err := DB.Exec("INSERT INTO schema_migrations(version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		log.Printf("applied migration %s", version)
	}
	return nil
}

// very simple splitter on ';' not inside dollar-quoted strings; good enough for our simple schema
func splitSQLStatements(sqlText string) []string {
	parts := strings.Split(sqlText, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
