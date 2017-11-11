package dfsrconfig

import (
	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/dname"
)

// LocalSettings retreives the DFSR local settings for the given distinguished
// name.
func (c *Client) LocalSettings(settingsDN string) (settings dfsr.LocalSettings, err error) {
	s, err := c.client.Open(dname.URL(settingsDN))
	if err != nil {
		return
	}
	defer s.Close()

	return c.localSettings(s)
}

func (c *Client) localSettings(ls *adsi.Object) (settings dfsr.LocalSettings, err error) {
	settings.Version, err = ls.AttrString("msDFSR-Version")
	if err != nil {
		return
	}
	return
}
