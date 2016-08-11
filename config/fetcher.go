package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-ole/go-ole"

	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/core"
)

const groupQueryDelay = 25 * time.Millisecond // Group query delay to avoid rate-limiting by LDAP servers

type fetcher struct {
	client   *adsi.Client
	domain   *adsi.Container // root container of the domain
	settings *adsi.Container // DFSR settings of the domain
	mc       *memberCache    // Maps member distinguished names to MemberInfo
}

func newFetcher(client *adsi.Client, domain string) (*fetcher, error) {
	root, err := client.OpenContainer(domainPath(domainDN(domain)))
	if err != nil {
		return nil, err
	}

	settings, err := root.Container("", makeDN("cn", "DFSR-GlobalSettings", "System"))
	if err != nil {
		defer root.Close()
		return nil, err
	}

	return &fetcher{
		client:   client,
		domain:   root,
		settings: settings,
		mc:       newMemberCache(),
	}, nil
}

func (fetch *fetcher) Close() {
	if fetch.settings != nil {
		fetch.settings.Close()
		fetch.settings = nil
	}

	if fetch.domain != nil {
		fetch.domain.Close()
		fetch.domain = nil
	}
}

func (fetch *fetcher) NamingContext() (nc core.NamingContext, err error) {
	d, err := fetch.domain.ToObject()
	if err != nil {
		return
	}

	nc.ID, err = d.GUID()
	if err != nil {
		return
	}

	nc.Path, err = d.Path()
	if err != nil {
		return
	}

	nc.DN = strings.TrimPrefix(nc.Path, "LDAP://")

	nc.Description, err = d.AttrString("description")
	if err != nil {
		return
	}
	return
}

// Domain will fetch DFSR configuration data from the fetcher's domain using the
// provided ADSI client.
func (fetch *fetcher) Domain() (domain core.Domain, err error) {
	start := time.Now()

	groups, err := fetch.groups()
	if err != nil {
		return
	}

	nc, err := fetch.NamingContext()
	if err != nil {
		return
	}

	return core.Domain{
		NamingContext:  nc,
		Groups:         groups,
		ConfigDuration: time.Now().Sub(start),
	}, nil
}

func (fetch *fetcher) groups() (groups []core.Group, err error) {
	iter, err := fetch.settings.Children()
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
			group, werr := fetch.group(g)
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
func (fetch *fetcher) groups() (groups []core.Group, err error) {
	iter, err := fetch.settings.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for g, err := iter.Next(); err == nil; g, err = iter.Next() {
		defer g.Close()

		group, err := fetch.group(g)
		if err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	return
}
*/

func (fetch *fetcher) GroupByName(groupName string) (group core.Group, err error) {
	groupName = strings.ToLower(groupName)

	iter, err := fetch.settings.Children()
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
			return fetch.group(g)
		}
	}

	err = errors.New("Replication Group not found.")
	return
}

func (fetch *fetcher) GroupByGUID(guid *ole.GUID) (group core.Group, err error) {
	g, err := fetch.settings.Object("msDFSR-ReplicationGroup", makeDN("CN", guid.String()[:]))
	if err != nil {
		return
	}

	return fetch.group(g)
}

func (fetch *fetcher) Group(dn string) (group core.Group, err error) {
	path := "LDAP://" + dn

	g, err := fetch.client.Open(path)
	if err != nil {
		return
	}
	defer g.Close()

	return fetch.group(g)
}

func (fetch *fetcher) group(g *adsi.Object) (group core.Group, err error) {
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

	group.Folders, err = fetch.folders(content)
	if err != nil {
		return
	}

	topology, err := gc.Object("msDFSR-Topology", "cn=Topology")
	if err != nil {
		return
	}
	defer topology.Close()

	group.Members, err = fetch.members(topology)
	if err != nil {
		return
	}

	group.ConfigDuration = time.Now().Sub(start)

	return
}

func (fetch *fetcher) folders(content *adsi.Container) (folders []core.Folder, err error) {
	iter, err := content.Children()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for f, err := iter.Next(); err == nil; f, err = iter.Next() {
		defer f.Close()

		folder, err := fetch.folder(f)
		if err != nil {
			return nil, err
		}

		folders = append(folders, folder)
	}

	return
}

func (fetch *fetcher) folder(f *adsi.Object) (folder core.Folder, err error) {
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

func (fetch *fetcher) members(topology *adsi.Object) (members []core.Member, err error) {
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

		member, err := fetch.member(m, "")
		if err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return
}

func (fetch *fetcher) Member(dn string) (member core.Member, err error) {
	path := "LDAP://" + dn
	m, err := fetch.client.Open(path)
	if err != nil {
		return
	}
	defer m.Close()

	return fetch.member(m, dn)
}

func (fetch *fetcher) member(m *adsi.Object, dn string) (member core.Member, err error) {
	member.MemberInfo, err = fetch.memberInfo(m, dn)
	if err != nil {
		return
	}

	member.Connections, err = fetch.connections(m)
	if err != nil {
		return
	}

	return
}

func (fetch *fetcher) MemberInfo(dn string) (member core.MemberInfo, err error) {
	member, ok := fetch.mc.Retrieve(dn)
	if ok {
		return
	}

	m, err := fetch.client.Open("LDAP://" + dn)
	if err != nil {
		return
	}
	defer m.Close()

	return fetch.memberInfo(m, dn)
}

func (fetch *fetcher) memberInfo(m *adsi.Object, dn string) (member core.MemberInfo, err error) {
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

	member.Name, err = m.Name()
	if err != nil {
		return
	}
	member.Name = strings.TrimPrefix(member.Name, "CN=")

	member.ID, err = m.GUID()
	if err != nil {
		return
	}

	compref, err := m.AttrString("msDFSR-ComputerReference")
	if err != nil {
		return
	}

	member.Computer, err = fetch.Computer(compref)

	fetch.mc.Add(member) // Add member info to the cache
	return
}

func (fetch *fetcher) connections(member *adsi.Object) (connections []core.Connection, err error) {
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

		conn, err := fetch.connection(c)
		if err != nil {
			return nil, err
		}

		connections = append(connections, conn)
	}

	return
}

func (fetch *fetcher) connection(c *adsi.Object) (conn core.Connection, err error) {
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

	mi, err := fetch.MemberInfo(conn.MemberDN)
	if err != nil {
		return
	}

	conn.Computer = mi.Computer

	return
}

func (fetch *fetcher) Computer(dn string) (computer core.Computer, err error) {
	path := "LDAP://" + dn

	c, err := fetch.client.Open(path)
	if err != nil {
		return
	}
	defer c.Close()

	return fetch.computer(c)
}

func (fetch *fetcher) computer(c *adsi.Object) (computer core.Computer, err error) {
	computer.DN, err = c.Path()
	if err != nil {
		return
	}

	computer.Host, err = c.AttrString("dNSHostName")
	if err != nil {
		return
	}

	return
}
