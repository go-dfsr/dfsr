package globalsettings

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/config/membercache"
	"gopkg.in/dfsr.v0/core"
)

// GlobalSettings provides a means of querying DFSR global settings.
type GlobalSettings struct {
	client   *adsi.Client
	domainDN string
	mc       *membercache.Cache // Maps distinguished names to MemberInfo
}

// New returns a new DFSR global settings configuration manager for the given
// domain.
//
// The provided ADSI client is retained by the global settings and will be used
// internally to peform the necessary LDAP queries. It is the caller's
// responsibility to explicitly close the ADSI client at an appropriate time
// when finished with the global settings.
func New(client *adsi.Client, domain string) *GlobalSettings {
	return &GlobalSettings{
		client:   client,
		domainDN: domainDN(domain),
		mc:       membercache.New(),
	}
}

// Domain will fetch DFSR configuration data from the domain.
func (gs *GlobalSettings) Domain() (domain core.Domain, err error) {
	start := time.Now()

	nc, err := gs.NamingContext()
	if err != nil {
		return
	}

	groups, err := gs.Groups()
	if err != nil {
		return
	}

	return core.Domain{
		NamingContext:  nc,
		Groups:         groups,
		ConfigDuration: time.Now().Sub(start),
	}, nil
}

// NamingContext returns information about the default naming context for the
// domain.
func (gs *GlobalSettings) NamingContext() (nc core.NamingContext, err error) {
	domain, err := gs.client.Open(ldap(gs.domainDN))
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
func (gs *GlobalSettings) Groups() (groups []core.Group, err error) {
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

	var results []chan groupResult

	for g, gerr := iter.Next(); gerr == nil; g, gerr = iter.Next() {
		ch := make(chan groupResult, 1)
		results = append(results, ch)

		go func(ch chan groupResult, g *adsi.Object) {
			defer g.Close()
			defer close(ch)
			group, werr := gs.group(g)
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
func (gs *GlobalSettings) groups() (groups []core.Group, err error) {
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
func (gs *GlobalSettings) GroupByName(groupName string) (group core.Group, err error) {
	groupName = strings.ToLower(groupName)

	container, err := gs.openContainer(makeDN("cn", "DFSR-GlobalSettings", "System"))
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
			return gs.group(g)
		}
	}

	err = errors.New("Replication Group not found.")
	return
}

// Group retreives the DFSR group configuration for the given distinguished
// name.
func (gs *GlobalSettings) Group(groupDN string) (group core.Group, err error) {
	g, err := gs.client.Open(ldap(groupDN))
	if err != nil {
		return
	}
	defer g.Close()

	return gs.group(g)
}

func (gs *GlobalSettings) group(g *adsi.Object) (group core.Group, err error) {
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

	group.Folders, err = gs.folders(content)
	if err != nil {
		return
	}

	topology, err := gc.Object("msDFSR-Topology", "cn=Topology")
	if err != nil {
		return
	}
	defer topology.Close()

	group.Members, err = gs.members(topology)
	if err != nil {
		return
	}

	group.ConfigDuration = time.Now().Sub(start)

	return
}

func (gs *GlobalSettings) folders(content *adsi.Container) (folders []core.Folder, err error) {
	iter, err := content.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for f, err := iter.Next(); err == nil; f, err = iter.Next() {
		defer f.Close()

		folder, err := gs.folder(f)
		if err != nil {
			return nil, err
		}

		folders = append(folders, folder)
	}

	return
}

func (gs *GlobalSettings) folder(f *adsi.Object) (folder core.Folder, err error) {
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

func (gs *GlobalSettings) members(topology *adsi.Object) (members []core.Member, err error) {
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

		member, err := gs.member(m, "")
		if err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return
}

// Member retreives the DFSR member configuration for the given distinguished
// name. The member's connection list is included in the returned data.
func (gs *GlobalSettings) Member(memberDN string) (member core.Member, err error) {
	m, err := gs.client.Open(ldap(memberDN))
	if err != nil {
		return
	}
	defer m.Close()

	return gs.member(m, memberDN)
}

func (gs *GlobalSettings) member(m *adsi.Object, dn string) (member core.Member, err error) {
	member.MemberInfo, err = gs.memberInfo(m, dn)
	if err != nil {
		return
	}

	serverref, _ := m.AttrString("serverReference")
	if serverref == "" {
		// Standard DFSR membership
		member.Connections, err = gs.connections(m)
		return
	}

	// Domain System Volume membership
	server, err := gs.client.Open(ldap(serverref))
	if err != nil {
		return
	}

	member.Connections, err = gs.connections(server)
	return
}

// MemberInfo retreives the DFSR member configuration for the given
// distinguished name. The member's connection list is not included in the
// returned data.
func (gs *GlobalSettings) MemberInfo(memberDN string) (member core.MemberInfo, err error) {
	member, ok := gs.mc.Retrieve(memberDN)
	if ok {
		return
	}

	m, err := gs.client.Open(ldap(memberDN))
	if err != nil {
		return
	}
	defer m.Close()

	return gs.memberInfo(m, memberDN)
}

func (gs *GlobalSettings) memberInfo(m *adsi.Object, dn string) (member core.MemberInfo, err error) {
	if dn == "" {
		path, perr := m.Path()
		if err != nil {
			err = perr
			return
		}
		member.DN = strings.TrimPrefix(path, "LDAP://")
	} else {
		member.DN = dn
	}

	class, err := m.Class()
	if err != nil {
		return
	}

	var compref string
	switch class {
	case "nTDSDSA":
		m, err = gs.openParent(m)
		defer m.Close()
		fallthrough
	case "server":
		compref, err = m.AttrString("serverReference")
		if err != nil {
			return
		}
	case "msDFSR-Member":
		compref, err = m.AttrString("msDFSR-ComputerReference")
		if err != nil {
			return
		}
	default:
		err = errors.New("Unknown Active Directory membership class")
		return
	}

	member.Name, err = m.Name()
	if err != nil {
		return
	}
	member.Name = strings.TrimPrefix(member.Name, "CN=")

	member.ID, err = m.GUID()
	if err != nil {
		return
	}

	member.Computer, err = gs.Computer(compref)

	gs.mc.Set(member) // Add member info to the cache
	return
}

func (gs *GlobalSettings) connections(member *adsi.Object) (connections []core.Connection, err error) {
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

	for c, err := iter.Next(); err == nil; c, err = iter.Next() {
		defer c.Close()

		conn, err := gs.connection(c)
		if err != nil {
			return nil, err
		}

		connections = append(connections, conn)
	}

	return
}

func (gs *GlobalSettings) connection(c *adsi.Object) (conn core.Connection, err error) {
	class, err := c.Class()
	if err != nil {
		return
	}

	conn.Name, err = c.Name()
	if err != nil {
		return
	}
	conn.Name = strings.TrimPrefix(conn.Name, "CN=")

	conn.ID, err = c.GUID()
	if err != nil {
		return
	}

	conn.MemberDN, err = c.AttrString("fromServer")
	if err != nil {
		return
	}

	if class == "msDFSR-Member" {
		// Standard DFSR connection
		conn.Enabled, err = c.AttrBool("msDFSR-Enabled")
		if err != nil {
			return
		}
	} else if class == "nTDSConnection" {
		// Domain System Volume membership
		conn.Enabled = true // These members are always enabled
	}

	mi, err := gs.MemberInfo(conn.MemberDN)
	if err != nil {
		return
	}

	conn.Computer = mi.Computer

	return
}

// Computer retrieves the DNS host name for the given distinguished name.
func (gs *GlobalSettings) Computer(dn string) (computer core.Computer, err error) {
	c, err := gs.client.Open(ldap(dn))
	if err != nil {
		return
	}
	defer c.Close()

	return gs.computer(c)
}

func (gs *GlobalSettings) computer(c *adsi.Object) (computer core.Computer, err error) {
	computer.DN, err = c.Path()
	if err != nil {
		return
	}
	computer.DN = strings.TrimPrefix(computer.DN, "LDAP://")

	computer.Host, err = c.AttrString("dNSHostName")
	if err != nil {
		return
	}

	return
}

func (gs *GlobalSettings) openParent(o *adsi.Object) (parent *adsi.Object, err error) {
	path, err := o.Parent()
	if err != nil {
		return
	}

	return gs.client.Open(path)
}

func (gs *GlobalSettings) openContainer(partialDN string) (*adsi.Container, error) {
	path := ldap(combineDN(partialDN, gs.domainDN))
	return gs.client.OpenContainer(path)
}
