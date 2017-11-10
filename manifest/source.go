package manifest

import (
	"bytes"
	"io"
	"os"
)

// Source is a source of conflict and deleted manifest data.
type Source interface {
	Reader() (io.ReadCloser, error)
	Info() (Info, error)
}

type fileSource struct {
	path string
}

func (s fileSource) Reader() (io.ReadCloser, error) {
	return os.Open(s.path)
}

func (s fileSource) Info() (info Info, err error) {
	fi, err := os.Stat(s.path)
	if err != nil {
		return
	}
	info.Modified = fi.ModTime()
	info.Size = fi.Size()
	return
}

type bufferedSource struct {
	content []byte
	info    Info
}

func (s *bufferedSource) Reader() (io.ReadCloser, error) {
	r := bytes.NewReader(s.content)
	return closableReader{Reader: r}, nil
}

func (s *bufferedSource) Info() (info Info, err error) {
	return s.info, nil
}

type closableReader struct {
	io.Reader
}

func (r closableReader) Close() error {
	return nil
}
