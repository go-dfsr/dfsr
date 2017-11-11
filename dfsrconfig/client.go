package dfsrconfig

import (
	"errors"
	"fmt"
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

// Groups retreives the DFSR group configuration for all groups contained in the
// domain.
func (c *Client) Groups() (groups []dfsr.Group, err error) {
	container, err := c.openContainer(dname.Make("cn", "DFSR-GlobalSettings", "System"))
	if err != nil {
		return nil, err
	}
	defer container.Close()

	iter, err := container.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var results []chan groupResult

	for g, gerr := iter.Next(); gerr == nil; g, gerr = iter.Next() {
		ch := make(chan groupResult, 1)
		results = append(results, ch)

		go func(ch chan groupResult, g *adsi.Object) {
			defer g.Close()
			defer close(ch)
			group, werr := c.group(g)
			ch <- groupResult{Group: group, Err: werr}
		}(ch, g)

		time.Sleep(groupQueryDelay) // Try to avoid rate-limiting
	}

	for i := 0; i < len(results); i++ {
		result := <-results[i]
		if err != nil {
			continue // Already hit an error, just drain the channels
		}

		if result.Err != nil {
			err = fmt.Errorf("Error retrieving configuration for replication group %v: %v", i, result.Err.Error())
			groups = nil
		} else {
			groups = append(groups, result.Group)
		}
	}

	return
}

/*
func (gs *GlobalSettings) groups() (groups []dfsr.Group, err error) {
	container, err := gs.openContainer(makeDN("cn", "DFSR-GlobalSettings", "System"))
	if err != nil {
		return nil, err
	}
	defer container.Close()

	iter, err := container.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for g, err := iter.Next(); err == nil; g, err = iter.Next() {
		defer g.Close()

		group, err := gs.group(g)
		if err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	return
}
*/

// GroupByName retreives the DFSR group configuration for the given name.
func (c *Client) GroupByName(groupName string) (group dfsr.Group, err error) {
	groupName = strings.ToLower(groupName)

	container, err := c.openContainer(dname.Make("cn", "DFSR-GlobalSettings", "System"))
	if err != nil {
		return
	}
	defer container.Close()

	iter, err := container.Children()
	if err != nil {
		return
	}
	defer iter.Close()

	for g, gerr := iter.Next(); gerr == nil; g, gerr = iter.Next() {
		defer g.Close()

		candidate, cerr := g.Name()
		if err != nil {
			err = cerr
			return
		}
		candidate = strings.ToLower(candidate)

		if candidate == groupName || strings.TrimPrefix(candidate, "cn=") == groupName {
			return c.group(g)
		}
	}

	err = errors.New("replication group not found")
	return
}

// Group retreives the DFSR group configuration for the given distinguished
// name.
func (c *Client) Group(groupDN string) (group dfsr.Group, err error) {
	g, err := c.client.Open(dname.URL(groupDN))
	if err != nil {
		return
	}
	defer g.Close()

	return c.group(g)
}

func (c *Client) group(g *adsi.Object) (group dfsr.Group, err error) {
	start := time.Now()

	group.Name, err = g.Name()
	if err != nil {
		return
	}
	group.Name = strings.TrimPrefix(group.Name, "CN=")

	group.ID, err = g.GUID()
	if err != nil {
		return
	}

	gc, err := g.ToContainer()
	if err != nil {
		return
	}
	defer gc.Close()

	content, err := gc.Container("msDFSR-Content", "cn=Content")
	if err != nil {
		return
	}
	defer content.Close()

	group.Folders, err = c.folders(content)
	if err != nil {
		return
	}

	topology, err := gc.Object("msDFSR-Topology", "cn=Topology")
	if err != nil {
		return
	}
	defer topology.Close()

	group.Members, err = c.members(topology)
	if err != nil {
		return
	}

	group.ConfigDuration = time.Now().Sub(start)

	return
}

func (c *Client) folders(content *adsi.Container) (folders []dfsr.Folder, err error) {
	iter, err := content.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for f, err := iter.Next(); err == nil; f, err = iter.Next() {
		defer f.Close()

		folder, err := c.folder(f)
		if err != nil {
			return nil, err
		}

		folders = append(folders, folder)
	}

	return
}

func (c *Client) folder(f *adsi.Object) (folder dfsr.Folder, err error) {
	folder.Name, err = f.Name()
	if err != nil {
		return
	}
	folder.Name = strings.TrimPrefix(folder.Name, "CN=")

	folder.ID, err = f.GUID()
	if err != nil {
		return
	}

	return
}

func (c *Client) members(topology *adsi.Object) (members []dfsr.Member, err error) {
	tc, err := topology.ToContainer()
	if err != nil {
		return nil, err
	}
	defer tc.Close()

	iter, err := tc.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for m, err := iter.Next(); err == nil; m, err = iter.Next() {
		defer m.Close()

		member, err := c.member(m, "")
		if err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return
}

// Member retreives the DFSR member configuration for the given distinguished
// name. The member's connection list is included in the returned data.
func (c *Client) Member(memberDN string) (member dfsr.Member, err error) {
	m, err := c.client.Open(dname.URL(memberDN))
	if err != nil {
		return
	}
	defer m.Close()

	return c.member(m, memberDN)
}

func (c *Client) member(m *adsi.Object, dn string) (member dfsr.Member, err error) {
	member.MemberInfo, err = c.memberInfo(m, dn)
	if err != nil {
		return
	}

	serverref, _ := m.AttrString("serverReference")
	if serverref == "" {
		// Standard DFSR membership
		member.Connections, err = c.connections(m)
		return
	}

	// Domain System Volume membership
	server, err := c.client.Open(dname.URL(serverref))
	if err != nil {
		return
	}

	member.Connections, err = c.connections(server)
	return
}

// MemberInfo retreives the DFSR member configuration for the given
// distinguished name. The member's connection list is not included in the
// returned data.
func (c *Client) MemberInfo(memberDN string) (member dfsr.MemberInfo, err error) {
	member, ok := c.mc.Retrieve(memberDN)
	if ok {
		return
	}

	m, err := c.client.Open(dname.URL(memberDN))
	if err != nil {
		return
	}
	defer m.Close()

	return c.memberInfo(m, memberDN)
}

func (c *Client) memberInfo(obj *adsi.Object, dn string) (member dfsr.MemberInfo, err error) {
	if dn == "" {
		path, perr := obj.Path()
		if err != nil {
			err = perr
			return
		}
		member.DN = strings.TrimPrefix(path, "LDAP://")
	} else {
		member.DN = dn
	}

	class, err := obj.Class()
	if err != nil {
		return
	}

	var compref string
	switch class {
	case "nTDSDSA":
		obj, err = c.openParent(obj)
		if err != nil {
			return
		}
		defer obj.Close()
		fallthrough
	case "server":
		compref, err = obj.AttrString("serverReference")
		if err != nil {
			return
		}
	case "msDFSR-Member":
		compref, err = obj.AttrString("msDFSR-ComputerReference")
		if err != nil {
			return
		}
	default:
		err = errors.New("unknown active directory membership class")
		return
	}

	member.Name, err = obj.Name()
	if err != nil {
		return
	}
	member.Name = strings.TrimPrefix(member.Name, "CN=")

	member.ID, err = obj.GUID()
	if err != nil {
		return
	}

	member.Computer, err = c.Computer(compref)

	c.mc.Set(member) // Add member info to the cache
	return
}

func (c *Client) connections(member *adsi.Object) (connections []dfsr.Connection, err error) {
	mc, err := member.ToContainer()
	if err != nil {
		return nil, err
	}
	defer mc.Close()

	iter, err := mc.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for child, err := iter.Next(); err == nil; child, err = iter.Next() {
		defer child.Close()

		conn, err := c.connection(child)
		if err != nil {
			return nil, err
		}

		connections = append(connections, conn)
	}

	return
}

func (c *Client) connection(obj *adsi.Object) (conn dfsr.Connection, err error) {
	class, err := obj.Class()
	if err != nil {
		return
	}

	conn.Name, err = obj.Name()
	if err != nil {
		return
	}
	conn.Name = strings.TrimPrefix(conn.Name, "CN=")

	conn.ID, err = obj.GUID()
	if err != nil {
		return
	}

	conn.MemberDN, err = obj.AttrString("fromServer")
	if err != nil {
		return
	}

	if class == "msDFSR-Connection" {
		// Standard DFSR connection
		conn.Enabled, err = obj.AttrBool("msDFSR-Enabled")
		if err != nil {
			return
		}
	} else if class == "nTDSConnection" {
		// Domain System Volume membership
		conn.Enabled = true // These members are always enabled
	}

	mi, err := c.MemberInfo(conn.MemberDN)
	if err != nil {
		return
	}

	conn.Computer = mi.Computer

	return
}

// Computer retrieves the DNS host name for the given distinguished name.
func (c *Client) Computer(dn string) (computer dfsr.Computer, err error) {
	comp, err := c.client.Open(dname.URL(dn))
	if err != nil {
		return
	}
	defer comp.Close()

	return c.computer(comp)
}

func (c *Client) computer(comp *adsi.Object) (computer dfsr.Computer, err error) {
	computer.DN, err = comp.Path()
	if err != nil {
		return
	}
	computer.DN = strings.TrimPrefix(computer.DN, "LDAP://")

	computer.Host, err = comp.AttrString("dNSHostName")
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
