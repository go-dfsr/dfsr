package dfsrconfig

import (
	"strings"
	"time"

	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/dfsrconfig/membercache"
	"gopkg.in/dfsr.v0/dname"
)

// Client is capable of perfroming LDAP queries to retrieve DFSR configuration.
type Client struct {
	client   *adsi.Client
	domainDN string
	mc       *membercache.Cache // Maps distinguished names to MemberInfo
}

// NewClient returns a new DFSR configuration client for the given domain.
//
// The provided ADSI client is retained by the global settings and will be used
// internally to peform the necessary LDAP queries. It is the caller's
// responsibility to explicitly close the ADSI client at an appropriate time
// when finished with the global settings.
func NewClient(client *adsi.Client, domain string) *Client {
	return &Client{
		client:   client,
		domainDN: dname.Domain(domain),
		mc:       membercache.New(),
	}
}

// Domain will fetch DFSR configuration data from the domain.
func (c *Client) Domain() (domain dfsr.Domain, err error) {
	start := time.Now()

	nc, err := c.NamingContext()
	if err != nil {
		return
	}

	groups, err := c.Groups()
	if err != nil {
		return
	}

	return dfsr.Domain{
		NamingContext:  nc,
		Groups:         groups,
		ConfigDuration: time.Now().Sub(start),
	}, nil
}

// NamingContext returns information about the default naming context for the
// domain.
func (c *Client) NamingContext() (nc dfsr.NamingContext, err error) {
	domain, err := c.client.Open(dname.URL(c.domainDN))
	if err != nil {
		return
	}
	defer domain.Close()

	nc.ID, err = domain.GUID()
	if err != nil {
		return
	}

	nc.Path, err = domain.Path()
	if err != nil {
		return
	}

	nc.DN = strings.TrimPrefix(nc.Path, "LDAP://")

	nc.Description, err = domain.AttrString("description")
	if err != nil {
		return
	}
	return
}

func (c *Client) openParent(o *adsi.Object) (parent *adsi.Object, err error) {
	path, err := o.Parent()
	if err != nil {
		return
	}

	return c.client.Open(path)
}

func (c *Client) openContainer(partialDN string) (*adsi.Container, error) {
	path := dname.URL(dname.Combine(partialDN, c.domainDN))
	return c.client.OpenContainer(path)
}
