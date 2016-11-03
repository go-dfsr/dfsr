// +build windows

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop, pause or continue.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	env, err := &environment, error(nil)

	env.IsInteractive, err = svc.IsAnInteractiveSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to determine interactive session status: %v\n", err)
		os.Exit(2)
	}

	if !env.IsInteractive {
		runService(env)
		return
	}

	if len(os.Args) < 2 {
		usage("No command specified.")
	}

	env.Parse(os.Args[2:], flag.ExitOnError)
	err = env.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to detect environment: %v\n", err)
		os.Exit(2)
	}
	env.Analyze()

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		env.IsDebug = true
		runService(env)
		return
	case "install":
		err = installService(env)
	case "remove":
		err = removeService(env)
	case "start":
		err = startService(env)
	case "stop":
		err = controlService(env, svc.Stop, svc.Stopped)
	case "pause":
		err = controlService(env, svc.Pause, svc.Paused)
	case "continue":
		err = controlService(env, svc.Continue, svc.Running)
	default:
		usage(fmt.Sprintf("Invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("Failed to %s %s: %v", cmd, env.ServiceName, err)
	}
	return
}
