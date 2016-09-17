package stathatconsumer

import (
	"fmt"
	"strings"

	"github.com/stathat/go"

	"gopkg.in/dfsr.v0/core"
)

// Consumer represents a StatHat consumer of DFSR monitor backlog updates.
type Consumer struct {
	ezkey  string
	c      <-chan *core.Backlog
	format string
}

// New returns a new StatHat consumer of DFSR monitor backlog updates. The
// returned consumer will function until the provided backlog channel is closed.
//
// If the provided ezkey is empty New will panic.
//
// If the provided format is empty New will return a consumer that does nothing.
func New(ezkey string, format string, backlog <-chan *core.Backlog) *Consumer {
	if ezkey == "" {
		panic("ezkey not provided to StatHat consumer")
	}
	c := &Consumer{
		c:      backlog,
		ezkey:  ezkey,
		format: format,
	}
	go c.run()
	return c
}

func (c *Consumer) run() {
	for {
		backlog, ok := <-c.c
		if !ok {
			return
		}
		if !reportable(backlog) {
			continue
		}
		c.send(backlog)
	}
}

func (c *Consumer) send(backlog *core.Backlog) {
	var total int
	for _, value := range backlog.Backlog {
		if value < 0 {
			// Indicates per-folder error, skip when tallying
			continue
		}
		total += value
	}
	name := c.statName(backlog)
	if name == "" {
		return
	}
	stathat.PostEZValueTime(name, c.ezkey, float64(total), backlog.Timestamp.Unix())
}

func (c *Consumer) statName(backlog *core.Backlog) string {
	return fmt.Sprintf(c.format, backlog.Group.Name, backlog.From, backlog.To, nonFQDN(backlog.From), nonFQDN(backlog.To))
}

func nonFQDN(fqdn string) string {
	dot := strings.Index(fqdn, ".")
	if dot < 1 {
		return strings.ToUpper(fqdn)
	}
	return strings.ToUpper(fqdn[0:dot])
}

func reportable(backlog *core.Backlog) bool {
	if backlog.Err != nil {
		return false
	}

	if len(backlog.Backlog) == 0 {
		// Indicates replication group query error
		return false
	}

	for _, value := range backlog.Backlog {
		if value < 0 {
			// Indicates per-folder query error
			return false
		}
	}

	return true
}
