package main

import (
	"sync"

	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/helper"
)

type connection struct {
	From    string
	To      string
	Group   *core.Group
	Backlog []int
	Call    callstat.Call
	Err     error
}

func (c *connection) ComputeBacklog(client *helper.Client, wg *sync.WaitGroup) {
	c.Backlog, c.Call, c.Err = client.Backlog(c.From, c.To, *c.Group.ID)
	wg.Done()
}

func (c *connection) TotalBacklog() (backlog uint) {
	for _, b := range c.Backlog {
		if b > 0 {
			backlog += uint(b)
		}
	}
	return
}
