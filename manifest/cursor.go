package manifest

import (
	"errors"
	"io"
	"strings"
	"sync"
)

// Cursor provides forward-only access to DFSR conflict and deleted manifests.
//
// Cursors should be created with NewCursor. When finished with a cursor it
// should be closed.
type Cursor struct {
	mutex    sync.RWMutex
	reader   io.ReadCloser
	decoder  *Decoder
	resolver Resolver
	filter   Filter
	total    Stats
	filtered Stats
}

// NewCursor returns a new cursor for the DFSR conflict and deleted manifest
// file at path.
//
// When finished with the cursor, it is the caller's responsibiliy to close it.
func NewCursor(reader io.ReadCloser) *Cursor {
	return NewAdvancedCursor(reader, nil, nil)
}

// NewAdvancedCursor returns a new cursor for the DFSR conflict and deleted
// manifest file at path.
//
// If filter is non-nil, the cursor will return resources that are matched by
// the filter. If filter is nil, all resources will be returned.
//
// If resolver is non-nil, the cursor will attempt to populate the partner
// host and distinguished name fields of each resource by querying the resolver.
//
// When finished with the cursor, it is the caller's responsibiliy to close it.
func NewAdvancedCursor(reader io.ReadCloser, resolver Resolver, filter Filter) *Cursor {
	return &Cursor{reader: reader, decoder: NewDecoder(reader), resolver: resolver, filter: filter}
}

// Read returns the next resource record from the cursor. If the cursor includes
// a filter, the next resource record that matches the filter will be
// returned.
//
// Read returns io.EOF when it encounters the end of the manifest.
func (c *Cursor) Read() (resource Resource, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.reader == nil {
		err = errors.New("the cursor has already been closed")
		return
	}

	for {
		resource, err = c.decoder.Read()
		if err != nil {
			return
		}
		if partner, ok := c.resolver.Resolve(strings.ToLower(resource.PartnerGUID)); ok {
			resource.PartnerHost = partner.Computer.Host
			resource.PartnerDN = partner.Computer.DN
		}
		matched := c.process(&resource)
		if matched {
			return
		}
	}
}

// End will cause the cursor to read from its current position until the end of
// the manifest file. This is typically used to tally statistics about the
// manifest without processing individual records. Statistics can be retrieved
// by calling Stats().
//
// If End encounters an error before reaching the end of the file, that error
// will be returned. End returns nil when io.EOF has been reached.
//
// End will not close the cursor.
func (c *Cursor) End() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.reader == nil {
		return errors.New("the cursor has already been closed")
	}

	for {
		resource, err := c.decoder.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if partner, ok := c.resolver.Resolve(strings.ToLower(resource.PartnerGUID)); ok {
			resource.PartnerHost = partner.Computer.Host
			resource.PartnerDN = partner.Computer.DN
		}
		c.process(&resource)
	}
}

// Stats return the current statistics for the cursor. Only the resources
// processed thus far will be reflected in the statistics.
func (c *Cursor) Stats() (filtered, total Stats) {
	c.mutex.RLock()
	filtered, total = c.filtered, c.total
	c.mutex.RUnlock()
	return
}

// Close releases any resources consumed by the cursor.
func (c *Cursor) Close() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.reader == nil {
		return
	}
	err = c.reader.Close()
	c.reader = nil
	return
}

// process will update the cursor's statistics to reflect the inclusion of r.
// It returns true if the resource matches the cursor's filter.
//
// The caller must hold an exclusive lock on the cursor for the duration of
// the call.
func (c *Cursor) process(r *Resource) (matched bool) {
	matched = c.filter.Match(r)
	c.total.Add(r)
	if matched {
		c.filtered.Add(r)
	}
	return
}
