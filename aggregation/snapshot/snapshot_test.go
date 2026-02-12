package snapshot

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"home-automation-analytics/aggregation/storage"
)

// TestSnapshotExportCreatesConsistentCopy verifies snapshot export creates a
// file in the runtime snapshots directory and preserves persisted row counts.
func TestSnapshotExportCreatesConsistentCopy(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := storage.InitSchema(ctx, db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	control := storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 2}
	if err := storage.UpsertControl(ctx, db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}

	key := storage.AggregateKey{ControlID: "c1", ModelID: "m1", QuarterIndex: 0}
	if _, err := storage.GetOrCreateAggregate(ctx, db, key, 2); err != nil {
		t.Fatalf("get or create aggregate: %v", err)
	}

	outputDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(outputDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	snapshotPath, err := Export(ctx, db)
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}

	if !strings.HasSuffix(filepath.Dir(snapshotPath), filepath.Join("data", "snapshots")) {
		t.Fatalf("snapshot dir mismatch: got %s", filepath.Dir(snapshotPath))
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}

	snapDB, err := sql.Open("sqlite", snapshotPath)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer snapDB.Close()

	var count int
	row := snapDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM controls")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count controls: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 control, got %d", count)
	}

	row = snapDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM aggregates")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count aggregates: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 aggregate, got %d", count)
	}
}

// TestDefaultSnapshotPathUsesDataSnapshotsDir verifies default snapshot naming
// and placement follow the required data/snapshots convention.
func TestDefaultSnapshotPathUsesDataSnapshotsDir(t *testing.T) {
	path := defaultSnapshotPath()

	expectedDir := filepath.Join("data", "snapshots")
	if filepath.Dir(path) != expectedDir {
		t.Fatalf("default dir mismatch: got %s want %s", filepath.Dir(path), expectedDir)
	}

	name := filepath.Base(path)
	if !strings.HasPrefix(name, "snapshot-") {
		t.Fatalf("snapshot name should start with snapshot-: %s", name)
	}
	if !strings.HasSuffix(name, ".sqlite") {
		t.Fatalf("snapshot name should end with .sqlite: %s", name)
	}
}

// TestExportForTestUsesDeterministicTestPath verifies test snapshot export
// uses the required deterministic test-data naming convention.
func TestExportForTestUsesDeterministicTestPath(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := storage.InitSchema(ctx, db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	outputDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(outputDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	snapshotPath, err := ExportForTest(ctx, db, "kitchen-test", "case-1")
	if err != nil {
		t.Fatalf("export snapshot for test: %v", err)
	}
	wantSuffix := filepath.Join("test-data", "snapshots", "kitchen-test-case-1-snapshot.sqlite")
	if !strings.HasSuffix(snapshotPath, wantSuffix) {
		t.Fatalf("snapshot path mismatch: got %s want suffix %s", snapshotPath, wantSuffix)
	}
}

// TestExportForTestRejectsPathTraversal verifies snapshot test naming rejects
// path separators so callers cannot escape test-data/snapshots.
func TestExportForTestRejectsPathTraversal(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := storage.InitSchema(ctx, db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	if _, err := ExportForTest(ctx, db, "../case", "snapshot"); err == nil {
		t.Fatalf("expected path traversal testName to fail")
	}
	if _, err := ExportForTest(ctx, db, "case", "../snapshot"); err == nil {
		t.Fatalf("expected path traversal snapshotName to fail")
	}
}
