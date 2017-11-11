package dfsrconfig

import (
	"strings"

	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
)

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
