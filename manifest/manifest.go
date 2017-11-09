package manifest

import "os"

// Manifest provides access to a DFSR conflict and deleted manifest file.
type Manifest struct {
	path string
}

// New returns a manifest for the conflict and deleted manifest XML file at the
// given path.
func New(path string) *Manifest {
	return &Manifest{path: path}
}

// Info returns information about the manifest.
func (m *Manifest) Info() (info Info, err error) {
	fi, err := os.Stat(m.path)
	if err != nil {
		return
	}
	info.Modified = fi.ModTime()
	info.Size = fi.Size()
	return
}

// Stats return statistics for the manifest. The returned total reflects all
// resources recorded in the manifest. If the provided cursor options describe
// filtering rules, filtered reflects only those resources that matched the
// filter.
func (m *Manifest) Stats(filter Filter) (filtered Stats, total Stats, err error) {
	cursor, err := m.AdvancedCursor(nil, filter)
	if err != nil {
		return
	}
	defer cursor.Close()

	err = cursor.End()
	if err != nil {
		return
	}

	filtered, total = cursor.Stats()

	return
}

// Cursor creates a new cursor for the manifest.
func (m *Manifest) Cursor() (*Cursor, error) {
	return NewCursor(m.path)
}

// AdvancedCursor creates a new cursor for the manifest with the given
// resolver and filter.
func (m *Manifest) AdvancedCursor(resolver Resolver, filter Filter) (*Cursor, error) {
	return NewAdvancedCursor(m.path, resolver, filter)
}
