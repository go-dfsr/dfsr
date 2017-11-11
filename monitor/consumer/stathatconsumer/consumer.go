package stathatconsumer

import (
	"fmt"
	"strings"

	"github.com/stathat/go"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/monitor"
)

// Consumer represents a StatHat consumer of DFSR monitor backlog updates.
type Consumer struct {
	ezkey  string
	ch     <-chan *monitor.Update
	format string
}

// New returns a new StatHat consumer of DFSR monitor backlog updates. The
// returned consumer will function until the provided backlog channel is closed.
//
// If the provided ezkey is empty New will panic.
//
// If the provided format is empty New will return a consumer that does nothing.
func New(ezkey string, format string, updates <-chan *monitor.Update) *Consumer {
	if ezkey == "" {
		panic("ezkey not provided to StatHat consumer")
	}
	c := &Consumer{
		ch:     updates,
		ezkey:  ezkey,
		format: format,
	}
	go c.run()
	return c
}

func (c *Consumer) run() {
	for {
		update, ok := <-c.ch
		if !ok {
			return
		}
		for backlog := range update.Listen() {
			if !reportable(backlog) {
				continue
			}
			c.send(backlog)
		}
	}
}

func (c *Consumer) send(backlog *dfsr.Backlog) {
	name := c.statName(backlog)
	if name == "" {
		return
	}
	stathat.PostEZValueTime(name, c.ezkey, float64(backlog.Sum()), backlog.Call.Start.Unix())
}

func (c *Consumer) statName(backlog *dfsr.Backlog) string {
	return fmt.Sprintf(c.format, backlog.Group.Name, backlog.From, backlog.To, nonFQDN(backlog.From), nonFQDN(backlog.To))
}

func nonFQDN(fqdn string) string {
	dot := strings.Index(fqdn, ".")
	if dot < 1 {
		return strings.ToUpper(fqdn)
	}
	return strings.ToUpper(fqdn[0:dot])
}

func reportable(backlog *dfsr.Backlog) bool {
	if backlog.Err != nil {
		return false
	}

	if len(backlog.Folders) == 0 {
		// Indicates replication group query error
		return false
	}

	for f := range backlog.Folders {
		if backlog.Folders[f].Backlog < 0 {
			// Indicates per-folder query error
			return false
		}
	}

	return true
}
