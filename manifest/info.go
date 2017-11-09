package manifest

import "time"

// Info provides information about a DFSR conflict and deleted manifest.
type Info struct {
	Modified time.Time // The last time the manifest was updated
	Size     int64     // The size of the manifest in bytes
}
