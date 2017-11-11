package dfsrconfig

import (
	"errors"
	"fmt"
	"strings"
	"time"

	adsi "gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/dname"
)

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

	type groupResult struct {
		Group dfsr.Group
		Err   error
	}

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
