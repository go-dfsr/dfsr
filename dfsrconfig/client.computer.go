package dfsrconfig

import (
	"strings"

	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/dname"
)

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
