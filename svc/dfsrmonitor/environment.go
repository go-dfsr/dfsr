// +build windows

package main

import (
	"errors"
	"flag"
	"path/filepath"
	"strings"

	"github.com/gentlemanautomaton/bindflag"
)

// Environment represents the service environment.
type Environment struct {
	ServiceName    string
	DisplayName    string
	Description    string
	IsInteractive  bool
	IsDebug        bool
	Account        string // Service account for installation
	Password       string // Service account password for installation
	ExePath        string // Currently running executable path
	InstallInPlace bool
	InstallPath    string
	InstallDir     string
	ProgramFiles   string // Program Files directory path
	Settings
}

var environment = Environment{
	ServiceName: DefaultServiceName,
	DisplayName: DefaultDisplayName,
	Description: DefaultDescription,
	Settings:    DefaultSettings,
}

// Bind will link the environment to the provided flag set.
func (e *Environment) Bind(fs *flag.FlagSet) {
	fs.Var(bindflag.String(&e.DisplayName), "name", "service display name")
	fs.Var(bindflag.String(&e.Description), "desc", "service description")
	fs.Var(bindflag.String(&e.Account), "account", "service account for installation")
	fs.Var(bindflag.String(&e.Password), "password", "service account password")
	fs.Var(bindflag.Bool(&e.InstallInPlace), "inplace", "use the current executable location as the installed service path")
	fs.Var(bindflag.String(&e.InstallPath), "path", "service path for installation")
	e.Settings.Bind(fs)
}

// Parse parses the given argument list and applies the specified values.
func (e *Environment) Parse(args []string, errorHandling flag.ErrorHandling) (err error) {
	fs := flag.NewFlagSet("", errorHandling)
	e.Bind(fs)
	return fs.Parse(args)
}

// Detect will inspect the environment variables and apply any relevant values.
func (e *Environment) Detect() (err error) {
	var err1, err2 error
	e.ExePath, err2 = exePath()
	e.ProgramFiles, err1 = programFilesDir()
	if err1 != nil {
		return err1
	}
	return err2
}

// Analyze will attempt to set an appropriate installation path if one has not
// been explicitly provided.
func (e *Environment) Analyze() {
	if e.InstallInPlace {
		e.InstallPath = e.ExePath
	}

	// If a path was provided start by cleaning it
	if e.InstallPath != "" {
		e.InstallPath = filepath.Clean(e.InstallPath)
	}

	// If a path wasn't provided try to come up with a default
	if e.InstallPath == "" && e.ProgramFiles != "" {
		if e.DisplayName != "" {
			e.InstallPath = filepath.Join(e.ProgramFiles, e.DisplayName)
		} else {
			e.InstallPath = filepath.Join(e.ProgramFiles, e.ServiceName)
		}
	}

	// Append the current filename if the path is a directory
	if e.InstallPath != "" && filepath.Ext(e.InstallPath) == "" {
		_, fileName := filepath.Split(e.ExePath)
		e.InstallPath = filepath.Join(e.InstallPath, fileName)
	}

	if e.InstallPath == "" {
		e.InstallPath = e.ExePath
	}

	e.InstallDir = filepath.Dir(e.InstallPath)
}

// Validate ensures that the environment has the necessary parameters for
// service installation.
func (e *Environment) Validate() error {
	if strings.Contains(e.InstallPath, "\"") {
		return errors.New("Installation path contains quotation marks.")
	}

	if e.ServiceName == "" {
		return errors.New("Unable to determine service name.")
	}

	if e.ExePath == "" {
		return errors.New("Unable to determine executable path.")
	}

	if e.InstallPath == "" {
		return errors.New("Unable to determine installation path.")
	}

	if e.InstallDir == "" {
		return errors.New("Unable to determine installation directory.")
	}

	return nil
}
