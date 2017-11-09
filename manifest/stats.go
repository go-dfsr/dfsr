package manifest

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/bytefmt"
)

// Stats hold cumulative statistics for DFSR manifests.
type Stats struct {
	Entries int       // Number of entries
	Size    int64     // Cumulative size of all entries
	First   time.Time // Timestamp of first entry
	Last    time.Time // Timestamp of last entry
}

// Add updates s to reflect the inclusion of r.
func (s *Stats) Add(r *Resource) {
	t := r.Time

	s.Entries++
	s.Size += r.Size

	if s.Entries == 1 {
		s.First = t
		s.Last = t
	} else {
		if t.Before(s.First) {
			s.First = t
		}
		if t.After(s.Last) {
			s.Last = t
		}
	}
}

// Summary returns a summary of the statistics.
func (s *Stats) Summary() string {
	first := s.First.In(time.Local).Format(time.RFC3339)
	last := s.Last.In(time.Local).Format(time.RFC3339)
	return fmt.Sprintf("Entries: %6d, Size: %8s, First: %v, Last: %v", s.Entries, bytefmt.ByteSize(uint64(s.Size)), first, last)
}
