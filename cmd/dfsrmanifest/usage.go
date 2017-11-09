package main

import (
	"flag"
	"fmt"
	"os"
)

// makeUsage prepares a usage string for the given executable name and command.
func makeUsage(program, command string) string {
	const (
		args     = "[-i regexp] [-e regexp] [-after date] [-before date] <path> [path...]"
		commands = "summary, list or dump"
		indent   = "       "
	)
	if command == "" {
		return fmt.Sprintf("usage: %s <command> %s\n%swhere <command> is one of %s\n", program, args, indent, commands)
	}
	return fmt.Sprintf("usage: %s %s %s\n", program, command, args)
}

func makeUsageFunc(fs *flag.FlagSet, program, command string) func() {
	usage := makeUsage(program, command)
	return func() {
		fmt.Fprint(os.Stderr, usage)
		fs.PrintDefaults()
	}
}
