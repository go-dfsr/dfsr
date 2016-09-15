// +build windows

package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/dfsr.v0/config"
	"gopkg.in/dfsr.v0/monitor"
	"gopkg.in/dfsr.v0/monitor/consumer/stathatconsumer"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type dfsrmonitor struct{}

const acceptedCmds = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

func (m *dfsrmonitor) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	changes <- svc.Status{State: svc.StartPending}

	// TODO: Move all of this initialization code into its own goroutine with a context for cancellation

	// Step 1: Parse settings
	settings := DefaultSettings
	if isIntSess {
		settings.Parse(os.Args[2:])
	} else {
		settings.Parse(args)
	}

	// Step 2: Create and start configuration monitor
	elog.Info(EventInitProgress, "Creating configuration monitor.")
	cfg := config.NewDomainMonitor(settings.Domain, settings.ConfigPollingInterval)
	if err := cfg.Start(); err != nil {
		elog.Error(EventInitFailure, fmt.Sprintf("Configuration initialization failure: %v", err))
		return true, ErrConfigInitFailure
	}
	defer cfg.Close()

	cfg.Update()
	if err := cfg.WaitReady(); err != nil { // TODO: Support some sort of timeout
		elog.Error(EventInitFailure, fmt.Sprintf("Configuration initialization failure: %v", err))
		return true, ErrConfigInitFailure
	}

	// Step 3: Create backlog monitor
	elog.Info(EventInitProgress, "Creating backlog monitor.")
	mon := monitor.New(cfg, settings.BacklogPollingInterval, settings.VectorCacheDuration, settings.Limit)
	monChan := mon.Listen(16, time.Second*5)

	// Step 4: Create backlog consumers
	if settings.StatHatKey != "" {
		stathatconsumer.New(settings.StatHatKey, settings.StatHatFormat, mon.Listen(16, time.Second*30))
	}

	// Step 5: Start backlog monitor
	if err := mon.Start(); err != nil {
		elog.Error(EventInitFailure, fmt.Sprintf("Monitor initialization failure: %v", err))
		return true, 1
	}
	defer mon.Close()

	elog.Info(EventInitComplete, "Initialization complete.")

	changes <- svc.Status{State: svc.Running, Accepts: acceptedCmds}

	mon.Update() // Kick off an initial poll right away

	for {
		select {
		case backlog, running := <-monChan:
			if !running {
				return
			}
			if backlog.Err == nil {
				//elog.Info(1, fmt.Sprintf("[%s]Backlog from %s to %s: %v", backlog.Group.Name, backlog.From, backlog.To, backlog.Backlog))
			} else {
				//elog.Info(1, fmt.Sprintf("[%s]Backlog from %s to %s: %v", backlog.Group.Name, backlog.From, backlog.To, backlog.Err))
			}
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				//elog.Info(1, "Stop or Shutdown")
				go mon.Close()
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: acceptedCmds}
				//elog.Info(1, "Paused")
				mon.Stop()
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: acceptedCmds}
				//elog.Info(1, "Continued")
				mon.Start()
			default:
				elog.Error(1, fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}
}

func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting %s service.", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &dfsrmonitor{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to start %s service: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("Stopped %s service.", name))
}
