package core

import (
	"time"

	"github.com/go-ole/go-ole"
)

// NamingContext represents an active directory naming context.
type NamingContext struct {
	ID          *ole.GUID
	DN          string // Distinguished name
	Description string
	Path        string
}

// Domain contains the replication group information for an Active Directory
// domain.
type Domain struct {
	NamingContext
	Groups         []Group
	ConfigDuration time.Duration // Time elapsed while retrieving configuration
}

/*
// Site represents an Active Directory site.
type Site struct {
	Name           string
	ID             *ole.GUID
	Members        []Member
	ConfigDuration time.Duration // Time elapsed while retrieving configuration
}
*/

// Group represents a replication group.
type Group struct {
	Name           string
	ID             *ole.GUID
	Folders        []Folder
	Members        []Member
	ConfigDuration time.Duration // Time elapsed while retrieving configuration
}

// Folder represents a replication folder.
type Folder struct {
	Name string
	ID   *ole.GUID
}

// Member represents a replication member.
type Member struct {
	MemberInfo
	Connections []Connection
}

// MemberInfo represents identifying information about a replication member.
type MemberInfo struct {
	Name     string
	ID       *ole.GUID
	Computer Computer
	DN       string // Distinguished name of the member
}

// Computer represents information about a computer.
type Computer struct {
	DN   string // Distinguished name
	Host string
}

// Connection represents a one-way connection between replication members.
type Connection struct {
	Name     string
	ID       *ole.GUID
	MemberDN string
	Enabled  bool
	Computer Computer // Distinguished name of source member in topology, matches DN field of that Member
}

// Backlog represents the backlog from one DFSR member to another.
type Backlog struct {
	Group     *Group
	From      string
	To        string
	Backlog   []int
	Timestamp time.Time
	Duration  time.Duration // Wall time for backlog calculation
	Err       error
}
