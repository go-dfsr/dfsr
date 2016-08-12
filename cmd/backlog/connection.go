package main

import (
	"sync"
	"time"

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
	start := time.Now()
	c.Backlog, c.Err = client.Backlog(c.From, c.To, *c.Group.ID)
	c.BacklogDuration = time.Now().Sub(start)
	wg.Done()
}
