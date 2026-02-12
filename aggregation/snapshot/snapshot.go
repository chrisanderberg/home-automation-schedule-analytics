package snapshot

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func Export(ctx context.Context, db *sql.DB) (string, error) {
	outputPath := defaultSnapshotPath()
	outputPath, err := filepath.Abs(outputPath)
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
	defer snapDB.Close()

	if _, err := snapDB.ExecContext(ctx, "VACUUM"); err != nil {
		return "", fmt.Errorf("prepare snapshot: %w", err)
	}

	if err := copySQLiteDB(ctx, db, snapDB); err != nil {
		return "", err
	}

	return outputPath, nil
}

func defaultSnapshotPath() string {
	dir := filepath.Join("data", "snapshots")
	name := fmt.Sprintf("snapshot-%s.sqlite", time.Now().UTC().Format("20060102-150405"))
	return filepath.Join(dir, name)
}

func copySQLiteDB(ctx context.Context, source *sql.DB, dest *sql.DB) error {
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

func copyTable(ctx context.Context, source *sql.DB, dest *sql.DB, table string) error {
	rows, err := source.QueryContext(ctx, "SELECT * FROM "+table)
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
	insertSQL := "INSERT INTO " + table + " (" + joinColumns(cols) + ") VALUES (" + placeholders + ")"

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

func joinColumns(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	out := cols[0]
	for i := 1; i < len(cols); i++ {
		out += "," + cols[i]
	}
	return out
}
