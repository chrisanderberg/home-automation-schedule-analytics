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

// rootMarkers defines the default root markers used by Is.
//
// .reporoot is the preferred explicit sentinel to avoid surprising matches with
// nested project directories. go.work and .git are compatibility fallbacks.
var rootMarkers = []Marker{
	{Name: ".reporoot"},
	{Name: "go.work"},
	{Name: ".git", RequireDir: true},
}

// RootMarkers returns a copy of the default root markers used by Is.
// Callers cannot mutate the internal slice.
func RootMarkers() []Marker {
	return append([]Marker(nil), rootMarkers...)
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
	for _, marker := range rootMarkers {
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
