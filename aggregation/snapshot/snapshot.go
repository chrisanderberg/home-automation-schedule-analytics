package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Export writes a timestamped production snapshot under data/snapshots.
func Export(ctx context.Context, db *sql.DB) (string, error) {
	outputPath := defaultSnapshotPath()
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
	outputPath := testSnapshotPath(sanitizedTestName, sanitizedSnapshotName)
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
	_ = os.Remove(outputPath)

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
func defaultSnapshotPath() string {
	dir := filepath.Join("data", "snapshots")
	name := fmt.Sprintf("snapshot-%s.sqlite", time.Now().UTC().Format("20060102-150405"))
	return filepath.Join(dir, name)
}

// testSnapshotPath returns deterministic test snapshot path naming.
func testSnapshotPath(testName string, snapshotName string) string {
	dir := filepath.Join("test-data", "snapshots")
	name := fmt.Sprintf("%s-%s-snapshot.sqlite", testName, snapshotName)
	return filepath.Join(dir, name)
}

// copySQLiteDB copies user tables and their rows (excluding sqlite internal tables).
func copySQLiteDB(ctx context.Context, source *sql.DB, dest execContexter) error {
	rows, err := source.QueryContext(ctx, "SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
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

	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		if _, err := dest.ExecContext(ctx, insertSQL, values...); err != nil {
			return err
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
