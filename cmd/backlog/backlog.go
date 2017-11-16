package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/dfsrconfig"
	"gopkg.in/dfsr.v0/dfsrflag"
	"gopkg.in/dfsr.v0/helper"
)

var (
	domainFlag         string
	groupFlag          dfsrflag.RegexpSlice
	fromFlag           dfsrflag.RegexpSlice
	toFlag             dfsrflag.RegexpSlice
	memberFlag         dfsrflag.RegexpSlice
	loopFlag           dfsrflag.UintOrInf
	timeoutSecondsFlag dfsrflag.UintOrInf
	maxCallSecondsFlag dfsrflag.UintOrInf
	delaySecondsFlag   uint
	cacheSecondsFlag   uint
	skipFlag           dfsrflag.RegexpSlice
	minFlag            uint
	verboseFlag        bool
)

const (
	defaultLoopValue           = 1
	defaultTimeoutSecondsValue = 30
)

func init() {
	flag.StringVar(&domainFlag, "d", "", "domain to query")
	flag.Var(&groupFlag, "g", "group to query")
	flag.Var(&fromFlag, "f", "regex of source hostname")
	flag.Var(&toFlag, "t", "regex of dest hostname")
	flag.Var(&memberFlag, "m", "regex of member hostname (matches either dest or source)")
	flag.Var(&loopFlag, "loop", "number of iterations or \"infinite\"")
	flag.Var(&timeoutSecondsFlag, "timeout", "number of seconds before timeout occurs or \"infininte\"")
	flag.Var(&maxCallSecondsFlag, "unresponsive", "maximum number of seconds before an individual call is considered unresponsive or \"infininte\"")
	flag.UintVar(&delaySecondsFlag, "delay", 5, "number of seconds to delay between loops")
	flag.UintVar(&cacheSecondsFlag, "cache", 5, "number of seconds to cache vectors")
	flag.Var(&skipFlag, "skip", "regex of hostname to skip")
	flag.UintVar(&minFlag, "min", 0, "minimum backlog to display")
	flag.BoolVar(&verboseFlag, "v", false, "verbose")

	rand.Seed(time.Now().UnixNano())
}

func main() {
	flag.Parse()

	if !loopFlag.Present {
		loopFlag.Value = defaultLoopValue
	}

	if !loopFlag.Inf && loopFlag.Value == 0 {
		return
	}

	if !timeoutSecondsFlag.Present {
		timeoutSecondsFlag.Value = defaultTimeoutSecondsValue
	}

	domain, connections, err := setup(domainFlag, groupFlag, fromFlag, toFlag, memberFlag, skipFlag)
	if err != nil {
		log.Fatal(err)
	}

	cfg := helper.DefaultEndpointConfig

	if maxCallSecondsFlag.Present {
		cfg.AcceptableCallDuration = time.Second * time.Duration(maxCallSecondsFlag.Value)
	}

	if cacheSecondsFlag > 0 {
		cfg.Caching = true
		cfg.CacheDuration = time.Duration(cacheSecondsFlag) * time.Second
	}
	cfg.OfflineReconnectionInterval = time.Second * 60

	client := helper.NewClientWithConfig(cfg)

	if loopFlag.Inf {
		for loop := uint(0); ; loop++ {
			run(domain, loop, minFlag, client, connections)
			time.Sleep(time.Duration(delaySecondsFlag) * time.Second)
		}
	} else {
		for loop := uint(0); loop < loopFlag.Value; loop++ {
			run(domain, loop, minFlag, client, connections)
			if loop+1 < loopFlag.Value {
				fmt.Println("")
				time.Sleep(time.Duration(delaySecondsFlag) * time.Second)
			}
		}
	}
}

func run(domain string, iteration uint, min uint, client *helper.Client, connections []dfsr.Backlog) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if !timeoutSecondsFlag.Inf && timeoutSecondsFlag.Value != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSecondsFlag.Value)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(connections))

	//fmt.Printf("[query %v] %s\n", iteration, domain)
	fmt.Printf("%-50s %-50s %-50s %-15s %s\n", "Group", "Source", "Destination", "Backlog", "Time")
	fmt.Printf("%-50s %-50s %-50s %-15s %s\n", "-----", "------", "-----------", "-------", "----")

	start := time.Now()

	for i := 0; i < len(connections); i++ {
		go computeBacklog(ctx, client, &connections[i], &wg)
	}

	wg.Wait()

	finish := time.Now()

	for i := 0; i < len(connections); i++ {
		c := &connections[i]

		if c.Sum() < min {
			continue
		}

		fmt.Printf("%-50s %-50s %-50s ", c.Group.Name, c.From, c.To)
		if c.Err != nil {
			fmt.Printf("%-15v ", c.Err)
			//fmt.Printf("%-15v ", c.Call)
		} else {
			fmt.Printf("%-15s ", fmt.Sprint(c.Sum()))
		}
		fmt.Printf("%v\n", c.Call.Duration())
		if verboseFlag {
			fmt.Printf("Call: %v\n", c.Call)
		}
	}

	fmt.Printf("Total Time: %v\n", finish.Sub(start))
}

func setup(domain string, groupRegex, fromRegex, toRegex, memberRegex, skipRegex dfsrflag.RegexpSlice) (dom string, connections []dfsr.Backlog, err error) {
	client, err := adsi.NewClient()
	if err != nil {
		return "", nil, err
	}
	defer client.Close()

	if domain == "" {
		domain, err = dnc(client)
		if err != nil {
			return "", nil, err
		}
	}
	dom = domain

	d, err := dfsrconfig.Domain(client, domain)
	if err != nil {
		return domain, nil, err
	}

	for g := 0; g < len(d.Groups); g++ {
		group := &d.Groups[g]
		if !isMatch(group.Name, groupRegex, true) {
			continue
		}

		for m := 0; m < len(group.Members); m++ {
			member := &group.Members[m]
			to := member.Computer.Host
			if to == "" {
				continue
			}
			if isMatch(to, skipRegex, false) {
				continue
			}
			if !isMatch(to, toRegex, true) {
				continue
			}

			for c := 0; c < len(member.Connections); c++ {
				conn := &member.Connections[c]
				from := conn.Computer.Host
				if from == "" {
					continue
				}
				if !conn.Enabled {
					continue
				}
				if isMatch(from, skipRegex, false) {
					continue
				}
				if !isMatch(from, fromRegex, true) {
					continue
				}
				if !isMatch(from, memberRegex, true) && !isMatch(to, memberRegex, true) {
					continue
				}

				connections = append(connections, dfsr.Backlog{
					Group: group,
					From:  from,
					To:    to,
				})
			}
		}
	}
	return
}

func computeBacklog(ctx context.Context, client *helper.Client, backlog *dfsr.Backlog, wg *sync.WaitGroup) {
	var values []int
	values, backlog.Call, backlog.Err = client.Backlog(ctx, backlog.From, backlog.To, backlog.Group.ID)
	if n := len(values); n == len(backlog.Group.Folders) {
		backlog.Folders = make([]dfsr.FolderBacklog, n)
		for v := 0; v < n; v++ {
			backlog.Folders[v].Folder = &backlog.Group.Folders[v]
			backlog.Folders[v].Backlog = values[v]
		}
	}
	wg.Done()
}
