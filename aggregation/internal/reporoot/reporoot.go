package reporoot

import (
	"os"
	"path/filepath"
)

// Find searches upward for the monorepo root marker.
func Find(start string) (string, bool) {
	cur := filepath.Clean(start)
	for {
		if Is(cur) {
			return cur, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false
		}
		cur = parent
	}
}

// Is reports whether the path looks like the repository root.
func Is(path string) bool {
	info, err := os.Stat(filepath.Join(path, "aggregation"))
	return err == nil && info.IsDir()
}
