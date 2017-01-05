// +build windows

package main

import (
	"flag"
	"fmt"
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
	settings := environment.Settings
	if !environment.IsInteractive {
		elog.Info(1, fmt.Sprintf("Service Args: %v", args))
		settings.Parse(args[1:], flag.ExitOnError)
		elog.Info(1, fmt.Sprintf("Service Settings: %+v", settings))
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
	monChan := mon.Listen(16)

	// Step 4: Create backlog consumers
	if settings.StatHatKey != "" {
		stathatconsumer.New(settings.StatHatKey, settings.StatHatFormat, mon.Listen(16))
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
		case update, running := <-monChan:
			if !running {
				return
			}
			go watchUpdate(update)
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				elog.Info(1, "Received stop command. Stopping service.")
				go func() {
					mon.Close()
					elog.Info(1, "Service stopped.")
				}()
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: acceptedCmds}
				elog.Info(1, "Received pause command. Pausing service.")
				go func() {
					mon.Stop()
					elog.Info(1, "Service paused.")
				}()
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: acceptedCmds}
				//elog.Info(1, "Continued")
				elog.Info(1, "Received continue command. Unpausing service.")
				go func() {
					mon.Start()
					elog.Info(1, "Service unpaused.")
				}()
			default:
				elog.Error(1, fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}
}

func runService(env *Environment) {
	var err error
	if env.IsDebug {
		elog = debug.New(env.ServiceName)
	} else {
		elog, err = eventlog.Open(env.ServiceName)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting %s service.", env.ServiceName))
	run := svc.Run
	if env.IsDebug {
		run = debug.Run
	}
	err = run(env.ServiceName, &dfsrmonitor{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to start %s service: %v", env.ServiceName, err))
		return
	}
	elog.Info(1, fmt.Sprintf("Stopped %s service.", env.ServiceName))
}

func watchUpdate(update *monitor.Update) {
	elog.Info(1, fmt.Sprintf("Polling started at %v", update.Start()))
	for backlog := range update.Listen() {
		if backlog.Err != nil {
			elog.Warning(1, fmt.Sprintf("[%s] Backlog from %s to %s: %v", backlog.Group.Name, backlog.From, backlog.To, backlog.Err))
			continue
		}
		if !backlog.IsZero() {
			elog.Info(1, fmt.Sprintf("[%s] Backlog from %s to %s: %v", backlog.Group.Name, backlog.From, backlog.To, backlog.Sum()))
		}
	}
	elog.Info(1, fmt.Sprintf("Polling finished at %v. Total wall time: %v", update.End(), update.Duration()))
}
