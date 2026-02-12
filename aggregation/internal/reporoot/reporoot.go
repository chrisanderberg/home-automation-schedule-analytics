package reporoot

import (
	"os"
	"path/filepath"
)

// Marker identifies a filesystem entry that can signal the repository root.
type Marker struct {
	Name       string
	RequireDir bool
}

// RootMarkers defines the default root markers used by Is.
//
// .reporoot is the preferred explicit sentinel to avoid surprising matches with
// nested project directories. go.work and .git are compatibility fallbacks.
var RootMarkers = []Marker{
	{Name: ".reporoot"},
	{Name: "go.work"},
	{Name: ".git", RequireDir: true},
}

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
	for _, marker := range RootMarkers {
		info, err := os.Stat(filepath.Join(path, marker.Name))
		if err != nil {
			continue
		}
		if marker.RequireDir && !info.IsDir() {
			continue
		}
		return true
	}
	return false
}
