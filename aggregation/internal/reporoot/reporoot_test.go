package reporoot

import (
	"os"
	"path/filepath"
	"strings"
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
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(root): %v", err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks(got): %v", err)
	}
	if gotResolved != rootResolved {
		t.Fatalf("Find(%q) = %q, want %q", start, gotResolved, rootResolved)
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

func TestFindResolvesSymlinkedStart(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".reporoot"), []byte(""), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	realStart := filepath.Join(root, "aggregation", "internal")
	if err := os.MkdirAll(realStart, 0o755); err != nil {
		t.Fatalf("mkdir real start: %v", err)
	}

	linkedParent := t.TempDir()
	linkPath := filepath.Join(linkedParent, "linked-start")
	if err := os.Symlink(realStart, linkPath); err != nil {
		if isSymlinkUnsupported(err) {
			t.Skipf("symlinks not permitted in this environment: %v", err)
		}
		t.Fatalf("symlink start: %v", err)
	}

	got, ok := Find(linkPath)
	if !ok {
		t.Fatalf("expected Find(%q) to succeed", linkPath)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks(got): %v", err)
	}
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(root): %v", err)
	}
	if gotResolved != rootResolved {
		t.Fatalf("Find(%q) = %q, want %q", linkPath, gotResolved, rootResolved)
	}
}

// isSymlinkUnsupported returns true when the error indicates symlinks are not
// permitted in this environment (e.g., Windows, restricted filesystems).
func isSymlinkUnsupported(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not permitted") || strings.Contains(s, "operation not supported")
}
