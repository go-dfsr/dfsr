// +build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func installService(env *Environment) error {
	//log.Printf("Environment: %+v", env)

	// Connect to the service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("Unable to connect to local service manager: %v", err)
	}
	defer m.Disconnect()

	// Make sure the service doesn't already exist
	s, err := m.OpenService(env.ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service \"%s\" already exists", env.ServiceName)
	}

	// Ensure the environment is valid
	if err = env.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	// Create the installation directory
	if err = os.MkdirAll(env.InstallDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create installation directory \"%s\": %v", env.InstallDir, err)
	}

	// Copy the service executable to its installation path
	if err = cp(env.ExePath, env.InstallPath); err != nil {
		return fmt.Errorf("unable to copy \"%s\" to \"%s\": %v", env.ExePath, env.InstallPath, err)
	}

	// Ensure the service account has read and execute rights on the new path
	if env.Account != "" {
		if err = grant(env.InstallDir, env.Account); err != nil {
			return fmt.Errorf("unable to grant read and execute rights to \"%s\" for \"%s\": %v", env.Account, env.InstallDir, err)
		}
	}

	// Prep the service configuration
	conf := mgr.Config{
		StartType:   mgr.StartAutomatic,
		DisplayName: env.DisplayName,
		Description: env.Description,
	}
	if env.Account != "" {
		conf.ServiceStartName = env.Account
	}
	if env.Password != "" {
		conf.Password = env.Password
	}

	//log.Printf("Service Configuration: %+v\n", conf)
	//log.Printf("Service Args: %+v\n", env.Settings.Args())

	// Create the service
	s, err = m.CreateService(env.ServiceName, env.InstallPath, conf, env.Settings.Args()...)
	if err != nil {
		return fmt.Errorf("unable to create service %s using \"%s\": %v", env.ServiceName, env.InstallPath, err)
	}
	defer s.Close()

	// Create the event log
	err = eventlog.InstallAsEventCreate(env.ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("creation of event log source failed: %s", err)
	}
	return nil
}

func removeService(env *Environment) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(env.ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", env.ServiceName)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(env.ServiceName)
	if err != nil {
		return fmt.Errorf("removal of event log source failed: %s", err)
	}
	return nil
}
