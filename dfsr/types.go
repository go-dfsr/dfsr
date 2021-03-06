package dfsr

import (
	"strings"
	"time"

	"gopkg.in/dfsr.v0/callstat"

	"github.com/gentlemanautomaton/calltracker"
	"github.com/google/uuid"
)

// Tracker represents a call state tracker.
type Tracker interface {
	Add() (call calltracker.TrackedCall)
	Value() (value calltracker.Value)
	Subscribe(s calltracker.Subscriber)
}

// NamingContext represents an active directory naming context.
type NamingContext struct {
	ID          uuid.UUID
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

// MemberInfoMap builds a map of members keyed by GUID. If d is nil it returns
// a nil map.
func (d *Domain) MemberInfoMap() MemberInfoMap {
	if d == nil {
		return nil
	}
	output := make(MemberInfoMap)
	for g := range d.Groups {
		for m := range d.Groups[g].Members {
			member := &d.Groups[g].Members[m]
			output[strings.ToLower(member.ID.String())] = member.MemberInfo
		}
	}
	return output
}

/*
// Site represents an Active Directory site.
type Site struct {
	Name           string
	ID             uuid.UUID
	Members        []Member
	ConfigDuration time.Duration // Time elapsed while retrieving configuration
}
*/

// Group represents a replication group.
type Group struct {
	Name           string
	ID             uuid.UUID
	Folders        []Folder
	Members        []Member
	ConfigDuration time.Duration // Time elapsed while retrieving configuration
}

// Folder represents a replication folder.
//
// TODO: Consider renaming this to ContentSet.
type Folder struct {
	Name string
	ID   uuid.UUID
}

// Member represents a replication member.
type Member struct {
	MemberInfo
	Connections []Connection
}

// MemberInfo represents identifying information about a replication member.
type MemberInfo struct {
	Name     string
	ID       uuid.UUID
	DN       string // Distinguished name of the member
	Computer Computer
	Settings LocalSettings
}

// Computer represents information about a computer.
type Computer struct {
	DN   string // Distinguished name
	Host string
}

// LocalSettings contains the local settings for a replication member. It
// includes configuration particular to that member, such as such as content
// set paths, staging directories, etc.
type LocalSettings struct {
	Version string // DFSR version
	Groups  []Subscriber
}

// Subscriber contains the set of content set subscriptions for a replication
// group member.
type Subscriber struct {
	MemberReference  string
	ReplicationGroup uuid.UUID
	ContentSets      []Subscription
}

// Subscription represents the content set settings for a replication group
// member.
type Subscription struct {
	ReplicationGroup uuid.UUID
	ContentSet       uuid.UUID
	RootPath         string
	RootSize         int // In Mibibytes (TODO: Convert to bytes?)
	StagingPath      string
	StagingSize      int // In Mibibytes (TODO: Convert to bytes?)
	ConflictPath     string
	ConflictSize     int // In Mibibytes (TODO: Convert to bytes?)
	Enabled          bool
	ReadOnly         bool
	Options          int // Enum flags
	CachePolicy      int // Enum flags
	MaxAgeInCache    time.Duration
	MinDurationCache time.Duration
}

// Connection represents a one-way connection between replication members.
type Connection struct {
	Name     string
	ID       uuid.UUID
	MemberDN string
	Enabled  bool
	Computer Computer // Distinguished name of source member in topology, matches DN field of that Member
}

// FolderBacklog represents the backlog for an individual folder.
type FolderBacklog struct {
	Folder  *Folder
	Backlog int
}

// Backlog represents the backlog from one DFSR member to another.
type Backlog struct {
	Group   *Group
	From    string
	To      string
	Folders []FolderBacklog
	Call    callstat.Call
	Err     error
}

// Sum returns the total backlog of all replicated folders. Negatives values,
// which incidate errors, are not included in the summation.
func (b *Backlog) Sum() (backlog uint) {
	for f := range b.Folders {
		if value := b.Folders[f].Backlog; value > 0 {
			backlog += uint(value)
		}
	}
	return
}

// IsZero reports whether b represents a successful backlog query that returned
// a count of zero for all replication folders in the replication group.
func (b *Backlog) IsZero() bool {
	if b.Err != nil {
		return false
	}

	if len(b.Folders) == 0 {
		return false
	}

	for f := range b.Folders {
		if b.Folders[f].Backlog != 0 {
			return false
		}
	}

	return true
}
