package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"home-automation-analytics/aggregation/internal/reporoot"
	_ "modernc.org/sqlite"
)

// Export writes a timestamped production snapshot under the resolved runtime
// snapshot root (SNAPSHOT_DIR or repository data/snapshots).
func Export(ctx context.Context, db *sql.DB) (string, error) {
	outputPath, err := defaultSnapshotPath()
	if err != nil {
		return "", err
	}
	return exportToPath(ctx, db, outputPath)
}

// ExportForTest writes a deterministic test snapshot path for fixture-style use.
func ExportForTest(ctx context.Context, db *sql.DB, testName string, snapshotName string) (string, error) {
	sanitizedTestName, err := sanitizeNameComponent(testName)
	if err != nil {
		return "", err
	}
	sanitizedSnapshotName, err := sanitizeNameComponent(snapshotName)
	if err != nil {
		return "", err
	}
	outputPath, err := testSnapshotPath(sanitizedTestName, sanitizedSnapshotName)
	if err != nil {
		return "", err
	}
	return exportToPath(ctx, db, outputPath)
}

// exportToPath materializes a standalone SQLite snapshot by recreating schema
// and copying table rows from source DB into destination DB.
func exportToPath(ctx context.Context, db *sql.DB, outputPath string) (_ string, err error) {
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", err
	}
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	snapDB, err := sql.Open("sqlite", outputPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := snapDB.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	if _, err := snapDB.ExecContext(ctx, "VACUUM"); err != nil {
		return "", fmt.Errorf("prepare snapshot: %w", err)
	}

	tx, err := snapDB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := copySQLiteDB(ctx, db, tx); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	committed = true

	return outputPath, nil
}

// defaultSnapshotPath returns a timestamped runtime snapshot filename.
func defaultSnapshotPath() (string, error) {
	dir, err := snapshotRootDir()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("snapshot-%s.sqlite", time.Now().UTC().Format("20060102-150405"))
	return filepath.Join(dir, name), nil
}

// testSnapshotPath returns deterministic test snapshot path naming.
func testSnapshotPath(testName string, snapshotName string) (string, error) {
	dir, err := testSnapshotRootDir()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%s-snapshot.sqlite", testName, snapshotName)
	return filepath.Join(dir, name), nil
}

// testSnapshotRootDir resolves the test snapshot root using TEST_DATA_DIR first,
// then repository-root-relative fallbacks mirroring snapshotRootDir behavior.
func testSnapshotRootDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("TEST_DATA_DIR")); override != "" {
		root, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, "snapshots"), nil
	}

	execPath, err := os.Executable()
	if err == nil {
		if root, ok := reporoot.Find(filepath.Dir(execPath)); ok {
			return filepath.Join(root, "test-data", "snapshots"), nil
		}
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		if root, ok := reporoot.Find(filepath.Dir(file)); ok {
			return filepath.Join(root, "test-data", "snapshots"), nil
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory for test snapshot path: %w", err)
	}
	if root, ok := reporoot.Find(wd); ok {
		return filepath.Join(root, "test-data", "snapshots"), nil
	}
	return "", fmt.Errorf("could not resolve test snapshot root: set TEST_DATA_DIR")
}

// snapshotRootDir resolves the production snapshot root using SNAPSHOT_DIR
// override first, then falls back to a repository-root-relative absolute path.
func snapshotRootDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("SNAPSHOT_DIR")); override != "" {
		return filepath.Abs(override)
	}

	execPath, err := os.Executable()
	if err == nil {
		if root, ok := reporoot.Find(filepath.Dir(execPath)); ok {
			return filepath.Join(root, "data", "snapshots"), nil
		}
	}

	// Source-relative fallback keeps tests stable when cwd/executable point at
	// temporary directories.
	if _, file, _, ok := runtime.Caller(0); ok {
		if root, ok := reporoot.Find(filepath.Dir(file)); ok {
			return filepath.Join(root, "data", "snapshots"), nil
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory for snapshot path: %w", err)
	}
	if root, ok := reporoot.Find(wd); ok {
		return filepath.Join(root, "data", "snapshots"), nil
	}
	return "", fmt.Errorf("could not resolve snapshot root: set SNAPSHOT_DIR")
}

// copySQLiteDB copies user tables and their rows, then recreates non-table
// schema objects (indexes, triggers, views), excluding sqlite internal objects.
func copySQLiteDB(ctx context.Context, source *sql.DB, dest execContexter) error {
	rows, err := source.QueryContext(ctx, "SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return err
	}
	defer rows.Close()

	tables := make([]string, 0)
	schemaSQL := make([]string, 0)
	for rows.Next() {
		var name string
		var sqlText string
		if err := rows.Scan(&name, &sqlText); err != nil {
			return err
		}
		tables = append(tables, name)
		schemaSQL = append(schemaSQL, sqlText)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, stmt := range schemaSQL {
		if _, err := dest.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	for _, table := range tables {
		if err := copyTable(ctx, source, dest, table); err != nil {
			return err
		}
	}

	otherRows, err := source.QueryContext(ctx, "SELECT sql FROM sqlite_master WHERE type IN ('index','trigger','view') AND sql IS NOT NULL AND name NOT LIKE 'sqlite_%' ORDER BY type, name")
	if err != nil {
		return err
	}
	defer otherRows.Close()
	for otherRows.Next() {
		var stmt string
		if err := otherRows.Scan(&stmt); err != nil {
			return err
		}
		if _, err := dest.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := otherRows.Err(); err != nil {
		return err
	}
	return nil
}

// copyTable streams all rows from one table and inserts them into destination.
func copyTable(ctx context.Context, source *sql.DB, dest execContexter, table string) error {
	quotedTable := quoteIdentifier(table)
	rows, err := source.QueryContext(ctx, "SELECT * FROM "+quotedTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	placeholders := "?"
	for i := 1; i < len(cols); i++ {
		placeholders += ",?"
	}
	insertSQL := "INSERT INTO " + quotedTable + " (" + joinColumns(cols) + ") VALUES (" + placeholders + ")"

	// Use a prepared statement when destination supports it to avoid repeated
	// SQL parsing on large table copies.
	type prepareContexter interface {
		PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	}
	execInsert := func(values ...any) error {
		_, err := dest.ExecContext(ctx, insertSQL, values...)
		return err
	}
	if prep, ok := dest.(prepareContexter); ok {
		stmt, err := prep.PrepareContext(ctx, insertSQL)
		if err != nil {
			return err
		}
		defer stmt.Close()
		execInsert = func(values ...any) error {
			_, err := stmt.ExecContext(ctx, values...)
			return err
		}
	}

	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		if err := execInsert(values...); err != nil {
			return err
		}
		for i := range values {
			values[i] = nil
		}
	}
	return rows.Err()
}

type execContexter interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// joinColumns builds a comma-separated identifier list for generated SQL.
func joinColumns(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	out := quoteIdentifier(cols[0])
	for i := 1; i < len(cols); i++ {
		out += "," + quoteIdentifier(cols[i])
	}
	return out
}

// quoteIdentifier safely quotes SQLite identifiers for generated statements.
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sanitizeNameComponent(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("invalid snapshot component: empty")
	}
	if value == "." || value == ".." {
		return "", fmt.Errorf("invalid snapshot component: path traversal not allowed")
	}
	if filepath.Base(value) != value {
		return "", fmt.Errorf("invalid snapshot component: path traversal not allowed")
	}
	if strings.ContainsRune(value, filepath.Separator) || strings.Contains(value, "/") || strings.Contains(value, `\`) {
		return "", fmt.Errorf("invalid snapshot component: path separator not allowed")
	}
	return value, nil
}
