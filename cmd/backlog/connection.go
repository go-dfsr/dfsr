package main

import (
	"sync"
	"time"

	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/helper"
)

type connection struct {
	From            string
	To              string
	Group           *core.Group
	Backlog         []int
	Err             error
	BacklogDuration time.Duration
}

func (c *connection) ComputeBacklog(client *helper.Client, wg *sync.WaitGroup) {
	var call callstat.Call
	c.Backlog, call, c.Err = client.Backlog(c.From, c.To, *c.Group.ID)
	c.BacklogDuration = call.Duration()
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
