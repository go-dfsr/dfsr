package manifest

import (
	"io/ioutil"
	"os"
)

// Manifest provides access to a DFSR conflict and deleted manifest file.
type Manifest struct {
	source Source
}

// New returns a manifest for the given source.
func New(s Source) *Manifest {
	return &Manifest{source: s}
}

// File returns a manifest for the conflict and deleted manifest file located
// at the given path.
func File(path string) *Manifest {
	return New(fileSource{path: path})
}

// BufferedFile returns a manifest for the conflict and deleted manifest file
// located at the given path. The contents of the file will be read immediately
// and buffered in memory.
func BufferedFile(path string) (*Manifest, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return New(&bufferedSource{
		content: content,
		info: Info{
			Modified: fi.ModTime(),
			Size:     fi.Size(),
		},
	}), nil
}

// Info returns information about the manifest.
func (m *Manifest) Info() (Info, error) {
	return m.source.Info()
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
	reader, err := m.source.Reader()
	if err != nil {
		return nil, err
	}
	return NewCursor(reader), nil
}

// AdvancedCursor creates a new cursor for the manifest with the given
// resolver and filter.
func (m *Manifest) AdvancedCursor(resolver Resolver, filter Filter) (*Cursor, error) {
	reader, err := m.source.Reader()
	if err != nil {
		return nil, err
	}
	return NewAdvancedCursor(reader, resolver, filter), nil
}
