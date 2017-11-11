package dfsrconfig

import (
	"errors"
	"strings"

	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/dname"
)

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

	connContainer := m

	if serverref, _ := m.AttrString("serverReference"); serverref != "" {
		// Domain System Volume membership has an extra level of indirection
		connContainer, err = c.client.Open(dname.URL(serverref))
		if err != nil {
			return
		}
	}

	member.Connections, err = c.connections(connContainer)
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
