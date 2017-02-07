package helper

import (
	"context"
	"strings"
	"sync"

	"github.com/go-ole/go-ole"
	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/versionvector"
)

// Client provides a threadsafe and efficient means of querying DFSR backlog
// and report information. It maintains an expiring cache of version vectors
// and attempts to manage DFSR queries in such a way that they do not overburden
// the target servers.
//
// Client maintains an internal map of DFSR endpoints and monitors their health.
// Queries against endpoints that are known to be offline will return a failure
// immediately.
type Client struct {
	mutex     sync.RWMutex
	config    EndpointConfig
	endpoints map[string]*Endpoint // Maps lower-case FQDNs to the Reporter inferface for each server
}

// NewClient creates a new Client that is capable of querying DFSR members via
// the DFSR Helper protocol. The returned Client will use the configuration
// values present in DefaultEndpointConfiguration.
func NewClient() *Client {
	return NewClientWithConfig(DefaultEndpointConfig)
}

// NewClientWithConfig creates a new Client that is capable of querying DFSR
// members via the DFSR Helper protocol. The returned Client will use the
// provided endpoint configuration values.
func NewClientWithConfig(config EndpointConfig) *Client {
	return &Client{
		config:    config,
		endpoints: make(map[string]*Endpoint),
	}
}

// Config returns the current configuration of the client.
func (c *Client) Config() (config EndpointConfig) {
	c.mutex.RLock()
	config = c.config
	c.mutex.RUnlock()
	return
}

// UpdateConfig updates the client configuration.
func (c *Client) UpdateConfig(config EndpointConfig) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.config = config
	if c.endpoints == nil {
		return // Already closed
	}
	for _, e := range c.endpoints {
		// TODO: Close in parallel
		e.UpdateConfig(config)
	}
}

// Close will release any resources consumed by the Client.
func (c *Client) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.endpoints == nil {
		return // Already closed
	}
	for _, e := range c.endpoints {
		// TODO: Close in parallel
		e.Close()
	}
	c.endpoints = nil
}

// Backlog returns the outgoing backlog from one DSFR member to another. The
// backlog of each replicated folder within the requested group is returned.
// The members are identified by their fully qualified domain names.
func (c *Client) Backlog(ctx context.Context, from, to string, group ole.GUID) (backlog []int, call callstat.Call, err error) {
	call.Begin("Client.Backlog")
	defer call.Complete(err)

	f, err := c.endpoint(from)
	if err != nil {
		return
	}

	t, err := c.endpoint(to)
	if err != nil {
		return
	}

	v, vcall, err := t.Vector(ctx, group)
	call.Add(&vcall)
	if err != nil {
		return
	}
	defer v.Close()

	backlog, bcall, err := f.Backlog(ctx, v)
	call.Add(&bcall)
	return
}

// Vector returns the current reference version vector of the requested
// replication group on the specified DFSR member. The member is identified by
// its fully qualified domain name.
func (c *Client) Vector(ctx context.Context, server string, group *ole.GUID) (vector *versionvector.Vector, call callstat.Call, err error) {
	call.Begin("Client.Vector")
	defer call.Complete(err)

	e, err := c.endpoint(server)
	if err != nil {
		return
	}

	vector, vcall, err := e.Vector(ctx, *group)
	call.Add(&vcall)
	return
}

// Report generates a report for the requested replication group.
func (c *Client) Report(ctx context.Context, server string, group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error) {
	call.Begin("Client.Report")
	defer call.Complete(err)

	e, err := c.endpoint(server)
	if err != nil {
		return
	}

	data, report, rcall, err := e.Report(ctx, group, vector, backlog, files)
	call.Add(&rcall)
	return
}

func (c *Client) endpoint(fqdn string) (*Endpoint, error) {
	fqdn = strings.ToLower(fqdn)

	// Existing entries
	c.mutex.RLock()
	if c.endpoints == nil {
		c.mutex.RUnlock()
		return nil, ErrClosed
	}
	e, found := c.endpoints[fqdn]
	c.mutex.RUnlock()
	if found {
		return e, nil
	}

	// New entries
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.endpoints == nil {
		return nil, ErrClosed
	}
	e, found = c.endpoints[fqdn]
	if found {
		return e, nil
	}
	e = NewEndpoint(fqdn, c.config)
	c.endpoints[fqdn] = e
	return e, nil
}
