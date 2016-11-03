// +build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hectane/go-acl"
	"golang.org/x/sys/windows"
)

func exePath() (string, error) {
	prog := os.Args[0]
	path, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}

	err = checkPath(path)
	if err == nil {
		return path, nil
	}

	if filepath.Ext(path) == "" {
		path += ".exe"
		err = checkPath(path)
		if err == nil {
			return path, nil
		}
	}

	return "", err
}

func checkPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode().IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

func programFilesDir() (string, error) {
	dir := os.Getenv("PROGRAMFILES")
	if len(dir) == 0 {
		return "", errors.New("Unable to determine ProgramFiles location")
	}
	return dir, nil
}

// cp will copy a file from src to dst if the paths are not identical.
//
// Original Source: https://gist.github.com/elazarl/5507969
func cp(src, dst string) error {
	if strings.ToLower(src) == strings.ToLower(dst) {
		return nil
	}
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

// Grant will grant the given account read and execute rights to the specified
// path.
func grant(path, account string) error {
	return acl.Apply(path, false, true, acl.GrantName(windows.GENERIC_READ|windows.GENERIC_EXECUTE, account))
}

func makeArg(name, value string) string {
	//return fmt.Sprintf("-%s=%s", name, syscall.EscapeArg(value))
	return fmt.Sprintf("-%s=%s", name, value)
}
