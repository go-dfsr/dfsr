package manifest

import (
	"os"
	"path/filepath"
	"strings"
)

// Find attempts to locate the DFSR manifest for the given path. It walks its
// way up the file tree looking for the DfsrPrivate directory. An empty string
// will be returned if it is unsuccessful.
func Find(path string) string {
	path, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if isManifest(path) {
		return path
	}
	var last string
	for path != last {
		last = path
		if candidate := filepath.Join(path, StandardFile); isManifest(candidate) {
			return candidate
		}
		if candidate := filepath.Join(path, StandardPath); isManifest(candidate) {
			return candidate
		}
		path = filepath.Dir(path)
	}
	return ""
}

func isManifest(path string) bool {
	if !isManifestPath(path) {
		return false
	}
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsRegular()
}

func isManifestPath(path string) bool {
	_, file := filepath.Split(path)
	return strings.EqualFold(file, StandardFile)
}
