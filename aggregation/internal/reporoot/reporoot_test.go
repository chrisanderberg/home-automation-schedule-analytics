package reporoot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWithFileMarker(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".reporoot"), []byte(""), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	if !Is(tmp) {
		t.Fatalf("expected Is(%q) to be true with .reporoot marker", tmp)
	}
}

func TestIsWithDirectoryMarker(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir marker: %v", err)
	}

	if !Is(tmp) {
		t.Fatalf("expected Is(%q) to be true with .git directory marker", tmp)
	}
}

func TestIsRejectsFileWhenRequireDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".git"), []byte(""), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	if Is(tmp) {
		t.Fatalf("expected Is(%q) to be false when .git is a file", tmp)
	}
}

func TestIsDoesNotMatchWithoutMarkers(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "aggregation"), 0o755); err != nil {
		t.Fatalf("mkdir aggregation: %v", err)
	}

	if Is(tmp) {
		t.Fatalf("expected Is(%q) to be false without configured markers", tmp)
	}
}

func TestFindSearchesUpwardToMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.23"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	start := filepath.Join(root, "aggregation", "internal", "reporoot")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("mkdir start: %v", err)
	}

	got, ok := Find(start)
	if !ok {
		t.Fatalf("expected Find(%q) to succeed", start)
	}
	if got != root {
		t.Fatalf("Find(%q) = %q, want %q", start, got, root)
	}
}

func TestFindReturnsFalseWithoutMarker(t *testing.T) {
	tmp := t.TempDir()
	start := filepath.Join(tmp, "a", "b")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("mkdir start: %v", err)
	}

	got, ok := Find(start)
	if ok {
		t.Fatalf("expected Find(%q) to fail without any root marker, got %q", start, got)
	}
	if got != "" {
		t.Fatalf("expected empty root when not found, got %q", got)
	}
}
